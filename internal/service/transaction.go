package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cauanvital/payment-gateway-simulator/internal/database/sqlc"
	"github.com/cauanvital/payment-gateway-simulator/internal/models"
	"github.com/cauanvital/payment-gateway-simulator/internal/payment"
	"github.com/cauanvital/payment-gateway-simulator/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrTerminalBlocked = errors.New("current terminal is blocked")
var ErrTransactionNotFound = errors.New("transaction not found")
var ErrIdempotencyKeyReused = errors.New("idempotency key already used with a different request")
var ErrIdempotencyResponseUnavailable = errors.New("idempotency response is unavailable")

type IdempotencyRequest struct {
	Key         string
	Endpoint    string
	RequestHash string
}

type IdempotentResponse struct {
	StatusCode int
	Body       json.RawMessage
	Replayed   bool
}

type TransactionService struct {
	pool         *pgxpool.Pool
	stateMachine *payment.PaymentStateMachine
	authorizer   *payment.Authorizer
}

func NewTransactionService(
	pool *pgxpool.Pool,
	sm *payment.PaymentStateMachine,
	authorizer *payment.Authorizer,
) *TransactionService {
	return &TransactionService{
		pool:         pool,
		stateMachine: sm,
		authorizer:   authorizer,
	}
}

func (s *TransactionService) Create(
	ctx context.Context,
	terminalSerial string,
	amount int64,
	currency string,
	method models.PaymentMethod,
	card string,
	idempotency IdempotencyRequest,
) (*IdempotentResponse, error) {
	return s.executeIdempotent(ctx, idempotency, 201, func(q *sqlc.Queries) (*models.Transaction, error) {
		terminalRepo := repository.NewTerminalRepository(q)
		terminal, err := terminalRepo.GetBySerial(ctx, terminalSerial)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrTerminalNotFound
			}
			return nil, err
		}
		if terminal.Status != models.TerminalActive {
			return nil, ErrTerminalBlocked
		}

		txRepo := repository.NewTransactionRepository(q)
		eventRepo := repository.NewTransactionEventRepository(q)

		transaction, err := txRepo.Create(ctx, terminal.MerchantID, terminal.ID, amount, currency, method)
		if err != nil {
			return nil, err
		}
		if _, err := eventRepo.Create(ctx, transaction.ID, "created", nil); err != nil {
			return nil, err
		}

		decision := s.authorizer.Authorize(transaction, card)
		if decision.Approved {
			if err := s.stateMachine.Authorize(transaction); err != nil {
				return nil, err
			}
			transaction, err = txRepo.Authorize(ctx, transaction.ID, decision.AuthCode)
			if err != nil {
				return nil, err
			}
			if _, err := eventRepo.Create(ctx, transaction.ID, "authorized", nil); err != nil {
				return nil, err
			}

			if method != models.MethodCreditCard {
				if err := s.stateMachine.Capture(transaction); err != nil {
					return nil, err
				}
				transaction, err = txRepo.Capture(ctx, transaction.ID)
				if err != nil {
					return nil, err
				}
				if _, err := eventRepo.Create(ctx, transaction.ID, "captured", nil); err != nil {
					return nil, err
				}
			}
		} else {
			if err := s.stateMachine.Decline(transaction); err != nil {
				return nil, err
			}
			transaction, err = txRepo.Decline(ctx, transaction.ID)
			if err != nil {
				return nil, err
			}

			reasonPayload, err := json.Marshal(map[string]string{"reason": decision.Reason})
			if err != nil {
				return nil, err
			}
			if _, err := eventRepo.Create(ctx, transaction.ID, "declined", reasonPayload); err != nil {
				return nil, err
			}
		}

		return transaction, nil
	})
}

func (s *TransactionService) Get(
	ctx context.Context,
	transactionID uuid.UUID,
) (*models.Transaction, []models.TransactionEvent, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	q := sqlc.New(tx)
	txRepo := repository.NewTransactionRepository(q)
	eventRepo := repository.NewTransactionEventRepository(q)

	transaction, err := txRepo.Get(ctx, transactionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrTransactionNotFound
		}
		return nil, nil, err
	}

	transactionEvents, err := eventRepo.List(ctx, transactionID)
	if err != nil {
		return nil, nil, err
	}

	return transaction, transactionEvents, nil
}

func (s *TransactionService) Capture(
	ctx context.Context,
	id uuid.UUID,
	idempotency IdempotencyRequest,
) (*IdempotentResponse, error) {
	return s.executeIdempotent(ctx, idempotency, 200, func(q *sqlc.Queries) (*models.Transaction, error) {
		txRepo := repository.NewTransactionRepository(q)
		eventRepo := repository.NewTransactionEventRepository(q)

		transaction, err := txRepo.GetForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrTransactionNotFound
			}
			return nil, err
		}

		if err := s.stateMachine.Capture(transaction); err != nil {
			return nil, err
		}

		updated, err := txRepo.Capture(ctx, id)
		if err != nil {
			return nil, err
		}
		if _, err := eventRepo.Create(ctx, id, "captured", nil); err != nil {
			return nil, err
		}
		return updated, nil
	})
}

func (s *TransactionService) Refund(
	ctx context.Context,
	id uuid.UUID,
	idempotency IdempotencyRequest,
) (*IdempotentResponse, error) {
	return s.executeIdempotent(ctx, idempotency, 200, func(q *sqlc.Queries) (*models.Transaction, error) {
		txRepo := repository.NewTransactionRepository(q)
		eventRepo := repository.NewTransactionEventRepository(q)

		transaction, err := txRepo.GetForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrTransactionNotFound
			}
			return nil, err
		}

		if err := s.stateMachine.Refund(transaction); err != nil {
			return nil, err
		}

		updated, err := txRepo.Refund(ctx, id)
		if err != nil {
			return nil, err
		}
		if _, err := eventRepo.Create(ctx, id, "refunded", nil); err != nil {
			return nil, err
		}
		return updated, nil
	})
}

func (s *TransactionService) executeIdempotent(
	ctx context.Context,
	idempotency IdempotencyRequest,
	statusCode int,
	operation func(*sqlc.Queries) (*models.Transaction, error),
) (*IdempotentResponse, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	q := sqlc.New(tx)
	idempotencyRepo := repository.NewIdempotencyKeyRepository(q)

	_, err = idempotencyRepo.TryCreate(ctx, idempotency.Key, idempotency.Endpoint, idempotency.RequestHash)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}

		saved, err := idempotencyRepo.Get(ctx, idempotency.Key, idempotency.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("getting idempotency key: %w", err)
		}
		if saved.RequestHash != idempotency.RequestHash {
			return nil, ErrIdempotencyKeyReused
		}
		if saved.StatusCode == 0 {
			return nil, ErrIdempotencyResponseUnavailable
		}

		return &IdempotentResponse{
			StatusCode: int(saved.StatusCode),
			Body:       saved.Response,
			Replayed:   true,
		}, nil
	}

	transaction, err := operation(q)
	if err != nil {
		return nil, err
	}

	response, err := json.Marshal(models.NewTransactionResponse(*transaction))
	if err != nil {
		return nil, err
	}
	if _, err := idempotencyRepo.Complete(
		ctx,
		idempotency.Key,
		idempotency.Endpoint,
		statusCode,
		response,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &IdempotentResponse{StatusCode: statusCode, Body: response}, nil
}
