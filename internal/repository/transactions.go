package repository

import (
	"context"

	"github.com/cauanvital/payment-gateway-simulator/internal/database/sqlc"
	"github.com/cauanvital/payment-gateway-simulator/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type TransactionRepository struct {
	q *sqlc.Queries
}

func NewTransactionRepository(q *sqlc.Queries) *TransactionRepository {
	return &TransactionRepository{q: q}
}

func (r *TransactionRepository) Create(
	ctx context.Context,
	merchantID uuid.UUID,
	terminalID uuid.UUID,
	amount int64,
	currency string,
	paymentMethod models.PaymentMethod,
) (*models.Transaction, error) {
	row, err := r.q.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		MerchantID:    merchantID,
		TerminalID:    terminalID,
		Amount:        amount,
		Currency:      currency,
		PaymentMethod: sqlc.PaymentMethod(paymentMethod),
	})
	if err != nil {
		return nil, err
	}
	transaction := toDomainTransaction(row)
	return &transaction, nil
}

func (r *TransactionRepository) Get(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	row, err := r.q.GetTransaction(ctx, id)
	if err != nil {
		return nil, err
	}
	transaction := toDomainTransaction(row)
	return &transaction, nil
}

func (r *TransactionRepository) Authorize(ctx context.Context, id uuid.UUID, authorizationCode string) (*models.Transaction, error) {
	row, err := r.q.AuthorizeTransaction(ctx, sqlc.AuthorizeTransactionParams{
		ID:                id,
		AuthorizationCode: pgtype.Text{String: authorizationCode, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	transaction := toDomainTransaction(row)
	return &transaction, nil
}

func (r *TransactionRepository) Decline(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	row, err := r.q.DeclineTransaction(ctx, id)
	if err != nil {
		return nil, err
	}
	transaction := toDomainTransaction(row)
	return &transaction, nil
}

func (r *TransactionRepository) Capture(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	row, err := r.q.CaptureTransaction(ctx, id)
	if err != nil {
		return nil, err
	}
	transaction := toDomainTransaction(row)
	return &transaction, nil
}

func (r *TransactionRepository) Refund(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	row, err := r.q.RefundTransaction(ctx, id)
	if err != nil {
		return nil, err
	}
	transaction := toDomainTransaction(row)
	return &transaction, nil
}

func toDomainTransaction(row sqlc.Transaction) models.Transaction {
	transaction := models.Transaction{
		ID:            row.ID,
		MerchantID:    row.MerchantID,
		TerminalID:    row.TerminalID,
		Amount:        row.Amount,
		Currency:      row.Currency,
		PaymentMethod: models.PaymentMethod(row.PaymentMethod),
		Status:        models.TransactionStatus(row.Status),
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}
	if row.AuthorizationCode.Valid {
		transaction.AuthorizationCode = &row.AuthorizationCode.String
	}
	return transaction
}
