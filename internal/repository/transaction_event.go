package repository

import (
	"context"
	"encoding/json"

	"github.com/cauanvital/payment-gateway-simulator/internal/database/sqlc"
	"github.com/cauanvital/payment-gateway-simulator/internal/models"
	"github.com/google/uuid"
)

type TransactionEventRepository struct {
	q *sqlc.Queries
}

func NewTransactionEventRepository(q *sqlc.Queries) *TransactionEventRepository {
	return &TransactionEventRepository{q: q}
}

func (r *TransactionEventRepository) Create(ctx context.Context, transactionID uuid.UUID, event string, payload json.RawMessage) (*models.TransactionEvent, error) {
	if payload == nil {
		payload = json.RawMessage("{}")
	}
	row, err := r.q.CreateTransactionEvent(ctx, sqlc.CreateTransactionEventParams{
		TransactionID: transactionID,
		Event:         event,
		Payload:       []byte(payload),
	})
	if err != nil {
		return nil, err
	}
	transactionEvent := toDomainTransactionEvent(row)
	return &transactionEvent, nil
}

func (r *TransactionEventRepository) List(ctx context.Context, transactionID uuid.UUID) ([]models.TransactionEvent, error) {
	rows, err := r.q.ListTransactionEvents(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	transactionEvents := make([]models.TransactionEvent, len(rows))
	for i, row := range rows {
		transactionEvents[i] = toDomainTransactionEvent(row)
	}

	return transactionEvents, nil
}

func toDomainTransactionEvent(row sqlc.TransactionEvent) models.TransactionEvent {
	return models.TransactionEvent{
		ID:            row.ID,
		TransactionID: row.TransactionID,
		Event:         row.Event,
		Payload:       row.Payload,
		CreatedAt:     row.CreatedAt.Time,
	}
}
