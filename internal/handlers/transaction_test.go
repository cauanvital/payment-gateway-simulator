package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cauanvital/payment-gateway-simulator/internal/models"
	"github.com/cauanvital/payment-gateway-simulator/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type transactionServiceFake struct {
	createInput   service.IdempotencyRequest
	captureInput  service.IdempotencyRequest
	refundInput   service.IdempotencyRequest
	createResult  *service.IdempotentResponse
	captureResult *service.IdempotentResponse
	refundResult  *service.IdempotentResponse
	createErr     error
	captureErr    error
	refundErr     error
}

func (f *transactionServiceFake) Create(_ context.Context, _ string, _ int64, _ string, _ models.PaymentMethod, _ string, request service.IdempotencyRequest) (*service.IdempotentResponse, error) {
	f.createInput = request
	return f.createResult, f.createErr
}
func (f *transactionServiceFake) Get(context.Context, uuid.UUID) (*models.Transaction, []models.TransactionEvent, error) {
	return nil, nil, service.ErrTransactionNotFound
}
func (f *transactionServiceFake) Capture(_ context.Context, _ uuid.UUID, request service.IdempotencyRequest) (*service.IdempotentResponse, error) {
	f.captureInput = request
	return f.captureResult, f.captureErr
}
func (f *transactionServiceFake) Refund(_ context.Context, _ uuid.UUID, request service.IdempotencyRequest) (*service.IdempotentResponse, error) {
	f.refundInput = request
	return f.refundResult, f.refundErr
}

func TestTransactionHandlerCreateRequiresIdempotencyKey(t *testing.T) {
	t.Parallel()

	handler := NewTransactionHandler(&transactionServiceFake{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(`{"terminal_serial":"POS-001"}`))
	recorder := httptest.NewRecorder()

	handler.Create(recorder, request)

	assertJSONError(t, recorder, http.StatusBadRequest, "Idempotency-Key is required")
}

func TestTransactionHandlerCreateReturnsServiceResponse(t *testing.T) {
	t.Parallel()

	fake := &transactionServiceFake{createResult: &service.IdempotentResponse{
		StatusCode: http.StatusCreated,
		Body:       json.RawMessage(`{"id":"transaction-1","status":"AUTHORIZED"}`),
	}}
	handler := NewTransactionHandler(fake, slog.New(slog.NewTextHandler(io.Discard, nil)))
	body := `{"terminal_serial":"POS-001","amount":1500,"currency":"BRL","payment_method":"CREDIT_CARD","card":"4111111111111111"}`
	request := httptest.NewRequest(http.MethodPost, "/transactions/", strings.NewReader(body))
	request.Header.Set("Idempotency-Key", "payment-001")
	recorder := httptest.NewRecorder()

	handler.Create(recorder, request)

	if recorder.Code != http.StatusCreated || recorder.Body.String() != string(fake.createResult.Body) {
		t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
	}
	if fake.createInput.Key != "payment-001" || fake.createInput.Endpoint != "POST /transactions" || fake.createInput.RequestHash == "" {
		t.Fatalf("unexpected idempotency input %#v", fake.createInput)
	}
}

func TestTransactionHandlerCaptureMapsIdempotencyConflict(t *testing.T) {
	t.Parallel()

	fake := &transactionServiceFake{captureErr: service.ErrIdempotencyKeyReused}
	handler := NewTransactionHandler(fake, slog.New(slog.NewTextHandler(io.Discard, nil)))
	id := uuid.New()
	request := requestWithURLParam(http.MethodPost, "/transactions/"+id.String()+"/capture", "id", id.String())
	request.Header.Set("Idempotency-Key", "capture-001")
	recorder := httptest.NewRecorder()

	handler.Capture(recorder, request)

	assertJSONError(t, recorder, http.StatusConflict, service.ErrIdempotencyKeyReused.Error())
	if fake.captureInput.Endpoint != "POST /transactions/{id}/capture" || fake.captureInput.RequestHash == "" {
		t.Fatalf("unexpected idempotency input %#v", fake.captureInput)
	}
}

func TestTransactionHandlerRejectsInvalidTransactionID(t *testing.T) {
	t.Parallel()

	handler := NewTransactionHandler(&transactionServiceFake{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := requestWithURLParam(http.MethodPost, "/transactions/nope/refund", "id", "nope")
	recorder := httptest.NewRecorder()

	handler.Refund(recorder, request)

	assertJSONError(t, recorder, http.StatusBadRequest, "invalid transaction id")
}

func TestNewIdempotencyRequestUsesMethodPathAndBody(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPost, "/transactions/one/capture", nil)
	request.Header.Set("Idempotency-Key", "  key-1  ")
	first, err := newIdempotencyRequest(request, "POST /transactions/{id}/capture", []byte(`{"value":1}`))
	if err != nil {
		t.Fatalf("newIdempotencyRequest() error = %v", err)
	}
	second, err := newIdempotencyRequest(request, "POST /transactions/{id}/capture", []byte(`{"value":2}`))
	if err != nil {
		t.Fatalf("newIdempotencyRequest() error = %v", err)
	}
	if first.Key != "key-1" || first.RequestHash == second.RequestHash {
		t.Fatalf("requests = %#v and %#v", first, second)
	}
}

func requestWithURLParam(method, target, key, value string) *http.Request {
	request := httptest.NewRequest(method, target, nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add(key, value)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}

func assertJSONError(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus int, wantMessage string) {
	t.Helper()
	if recorder.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, wantStatus, recorder.Body.String())
	}
	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if response["error"] != wantMessage {
		t.Fatalf("error = %q, want %q", response["error"], wantMessage)
	}
}

func TestTransactionHandlerGetReturnsEvents(t *testing.T) {
	t.Parallel()

	createdAt := time.Now().UTC().Round(0)
	id := uuid.New()
	txService := &transactionGetFake{transaction: &models.Transaction{ID: id, Status: models.TransactionAuthorized, CreatedAt: createdAt, UpdatedAt: createdAt}, events: []models.TransactionEvent{{ID: uuid.New(), Event: "authorized", Payload: json.RawMessage(`{}`), CreatedAt: createdAt}}}
	handler := NewTransactionHandler(txService, slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := requestWithURLParam(http.MethodGet, "/transactions/"+id.String(), "id", id.String())
	recorder := httptest.NewRecorder()

	handler.Get(recorder, request)

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"events":[`) {
		t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
	}
}

type transactionGetFake struct {
	transaction *models.Transaction
	events      []models.TransactionEvent
}

func (f *transactionGetFake) Create(context.Context, string, int64, string, models.PaymentMethod, string, service.IdempotencyRequest) (*service.IdempotentResponse, error) {
	return nil, errors.New("not implemented")
}
func (f *transactionGetFake) Get(context.Context, uuid.UUID) (*models.Transaction, []models.TransactionEvent, error) {
	return f.transaction, f.events, nil
}
func (f *transactionGetFake) Capture(context.Context, uuid.UUID, service.IdempotencyRequest) (*service.IdempotentResponse, error) {
	return nil, errors.New("not implemented")
}
func (f *transactionGetFake) Refund(context.Context, uuid.UUID, service.IdempotencyRequest) (*service.IdempotentResponse, error) {
	return nil, errors.New("not implemented")
}
