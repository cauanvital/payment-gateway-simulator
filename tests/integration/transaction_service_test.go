//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/cauanvital/payment-gateway-simulator/internal/models"
	"github.com/cauanvital/payment-gateway-simulator/internal/payment"
	"github.com/cauanvital/payment-gateway-simulator/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	integrationPool     *pgxpool.Pool
	integrationSetupErr error
	integrationSetup    sync.Once
)

func TestTransactionLifecycleAndEvents(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	_, serial := createMerchantAndTerminal(t, pool, models.TerminalActive)
	transactions := service.NewTransactionService(pool, &payment.PaymentStateMachine{}, &payment.Authorizer{})

	created, err := transactions.Create(ctx, serial, 1500, "BRL", models.MethodCreditCard, "4111111111111111", idempotency("create-lifecycle", "POST /transactions", "hash-create"))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	transaction := decodeTransaction(t, created.Body)
	if transaction.Status != models.TransactionAuthorized {
		t.Fatalf("created status = %s, want AUTHORIZED", transaction.Status)
	}

	captured, err := transactions.Capture(ctx, transaction.ID, idempotency("capture-lifecycle", "POST /transactions/{id}/capture", "hash-capture"))
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if got := decodeTransaction(t, captured.Body).Status; got != models.TransactionCaptured {
		t.Fatalf("captured status = %s, want CAPTURED", got)
	}

	refunded, err := transactions.Refund(ctx, transaction.ID, idempotency("refund-lifecycle", "POST /transactions/{id}/refund", "hash-refund"))
	if err != nil {
		t.Fatalf("Refund() error = %v", err)
	}
	if got := decodeTransaction(t, refunded.Body).Status; got != models.TransactionRefunded {
		t.Fatalf("refunded status = %s, want REFUNDED", got)
	}

	_, events, err := transactions.Get(ctx, transaction.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	wantEvents := []string{"created", "authorized", "captured", "refunded"}
	if len(events) != len(wantEvents) {
		t.Fatalf("events = %#v, want %d events", events, len(wantEvents))
	}
	for i, want := range wantEvents {
		if events[i].Event != want {
			t.Fatalf("event[%d] = %q, want %q", i, events[i].Event, want)
		}
	}
}

func TestTransactionIdempotencyAndRollback(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	merchantID, serial := createMerchantAndTerminal(t, pool, models.TerminalActive)
	transactions := service.NewTransactionService(pool, &payment.PaymentStateMachine{}, &payment.Authorizer{})
	request := idempotency("same-payment", "POST /transactions", "same-body-hash")

	first, err := transactions.Create(ctx, serial, 1500, "BRL", models.MethodCreditCard, "4111111111111111", request)
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	second, err := transactions.Create(ctx, serial, 1500, "BRL", models.MethodCreditCard, "4111111111111111", request)
	if err != nil {
		t.Fatalf("second Create() error = %v", err)
	}
	if !second.Replayed || !jsonEquivalent(t, first.Body, second.Body) {
		t.Fatalf("idempotent responses = %#v and %#v", first, second)
	}
	if got := count(t, pool, "SELECT count(*) FROM transactions WHERE merchant_id = $1", merchantID); got != 1 {
		t.Fatalf("transactions = %d, want 1", got)
	}

	_, err = transactions.Create(ctx, serial, 2000, "BRL", models.MethodCreditCard, "4111111111111111", idempotency("same-payment", "POST /transactions", "different-body-hash"))
	if !errors.Is(err, service.ErrIdempotencyKeyReused) {
		t.Fatalf("reused key error = %v, want %v", err, service.ErrIdempotencyKeyReused)
	}

	_, blockedSerial := createMerchantAndTerminal(t, pool, models.TerminalBlocked)
	_, err = transactions.Create(ctx, blockedSerial, 100, "BRL", models.MethodCreditCard, "4111111111111111", idempotency("blocked-terminal", "POST /transactions", "blocked-hash"))
	if !errors.Is(err, service.ErrTerminalBlocked) {
		t.Fatalf("blocked terminal error = %v, want %v", err, service.ErrTerminalBlocked)
	}
	if got := count(t, pool, "SELECT count(*) FROM idempotency_keys WHERE key = $1", "blocked-terminal"); got != 0 {
		t.Fatalf("idempotency key persisted after rollback: %d rows", got)
	}
}

func TestTransactionConcurrentCapture(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	_, serial := createMerchantAndTerminal(t, pool, models.TerminalActive)
	transactions := service.NewTransactionService(pool, &payment.PaymentStateMachine{}, &payment.Authorizer{})
	created, err := transactions.Create(ctx, serial, 1500, "BRL", models.MethodCreditCard, "4111111111111111", idempotency("concurrent-create", "POST /transactions", "concurrent-create-hash"))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	id := decodeTransaction(t, created.Body).ID

	start := make(chan struct{})
	errorsChannel := make(chan error, 2)
	var group sync.WaitGroup
	for _, key := range []string{"capture-a", "capture-b"} {
		group.Add(1)
		go func(key string) {
			defer group.Done()
			<-start
			_, err := transactions.Capture(ctx, id, idempotency(key, "POST /transactions/{id}/capture", key+"-hash"))
			errorsChannel <- err
		}(key)
	}
	close(start)
	group.Wait()
	close(errorsChannel)

	successes := 0
	invalidTransitions := 0
	for err := range errorsChannel {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, payment.ErrInvalidTransition):
			invalidTransitions++
		default:
			t.Fatalf("Capture() error = %v", err)
		}
	}
	if successes != 1 || invalidTransitions != 1 {
		t.Fatalf("successes=%d invalidTransitions=%d, want one of each", successes, invalidTransitions)
	}
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("set TEST_DATABASE_URL or run make test-integration")
	}
	integrationSetup.Do(func() {
		integrationPool, integrationSetupErr = pgxpool.New(context.Background(), os.Getenv("TEST_DATABASE_URL"))
		if integrationSetupErr == nil {
			integrationSetupErr = applyMigrations(context.Background(), integrationPool)
		}
	})
	if integrationSetupErr != nil {
		t.Fatalf("integration database setup: %v", integrationSetupErr)
	}
	return integrationPool
}

func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, "CREATE TABLE IF NOT EXISTS integration_migrations (version TEXT PRIMARY KEY)"); err != nil {
		return err
	}
	root, err := projectRoot()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Join(root, "migrations"))
	if err != nil {
		return err
	}
	var names []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".up.sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		var applied bool
		if err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM integration_migrations WHERE version = $1)", name).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		sql, err := os.ReadFile(filepath.Join(root, "migrations", name))
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, "INSERT INTO integration_migrations (version) VALUES ($1)", name); err != nil {
			return err
		}
	}
	return nil
}

func projectRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", os.ErrNotExist
		}
		directory = parent
	}
}

func createMerchantAndTerminal(t *testing.T, pool *pgxpool.Pool, status models.TerminalStatus) (uuid.UUID, string) {
	t.Helper()
	merchantID := uuid.New()
	terminalID := uuid.New()
	serial := "test-" + uuid.NewString()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "INSERT INTO merchants (id, name) VALUES ($1, $2)", merchantID, "Integration Merchant"); err != nil {
		t.Fatalf("create merchant: %v", err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO terminals (id, merchant_id, serial, status) VALUES ($1, $2, $3, $4)", terminalID, merchantID, serial, status); err != nil {
		t.Fatalf("create terminal: %v", err)
	}
	return merchantID, serial
}

func idempotency(key, endpoint, hash string) service.IdempotencyRequest {
	return service.IdempotencyRequest{Key: key, Endpoint: endpoint, RequestHash: hash}
}

func decodeTransaction(t *testing.T, body []byte) models.TransactionResponse {
	t.Helper()
	var transaction models.TransactionResponse
	if err := json.Unmarshal(body, &transaction); err != nil {
		t.Fatalf("decode transaction response: %v", err)
	}
	return transaction
}

func jsonEquivalent(t *testing.T, left, right []byte) bool {
	t.Helper()

	var leftValue any
	if err := json.Unmarshal(left, &leftValue); err != nil {
		t.Fatalf("decode first JSON response: %v", err)
	}
	var rightValue any
	if err := json.Unmarshal(right, &rightValue); err != nil {
		t.Fatalf("decode replayed JSON response: %v", err)
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

func count(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()
	var result int64
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&result); err != nil {
		t.Fatalf("count query: %v", err)
	}
	return int(result)
}
