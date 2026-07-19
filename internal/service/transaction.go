package service

import (
	"context"
	"encoding/json"
	"errors"

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

type TransactionService struct {
	pool         *pgxpool.Pool
	terminalRepo TerminalRepository
	stateMachine *payment.PaymentStateMachine
	authorizer   *payment.Authorizer
}

func NewTransactionService(
	pool *pgxpool.Pool,
	terminalRepo TerminalRepository,
	sm *payment.PaymentStateMachine,
	authorizer *payment.Authorizer,
) *TransactionService {
	return &TransactionService{
		pool:         pool,
		terminalRepo: terminalRepo,
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
) (*models.Transaction, error) {
	terminal, err := s.terminalRepo.GetBySerial(ctx, terminalSerial)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTerminalNotFound
		}
		return nil, err
	}
	if terminal.Status != models.TerminalActive {
		return nil, ErrTerminalBlocked
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	q := sqlc.New(tx)
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

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return transaction, nil
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

func (s *TransactionService) Capture(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	q := sqlc.New(tx)
	txRepo := repository.NewTransactionRepository(q)
	eventRepo := repository.NewTransactionEventRepository(q)

	transaction, err := txRepo.Get(ctx, id)
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

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *TransactionService) Refund(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	q := sqlc.New(tx)
	txRepo := repository.NewTransactionRepository(q)
	eventRepo := repository.NewTransactionEventRepository(q)

	transaction, err := txRepo.Get(ctx, id)
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

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return updated, nil
}
