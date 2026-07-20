package repository

import (
	"context"
	"encoding/json"

	"github.com/cauanvital/payment-gateway-simulator/internal/database/sqlc"
	"github.com/cauanvital/payment-gateway-simulator/internal/models"
)

type IdempotencyKeyRepository struct {
	q *sqlc.Queries
}

func NewIdempotencyKeyRepository(q *sqlc.Queries) *IdempotencyKeyRepository {
	return &IdempotencyKeyRepository{q: q}
}

func (r *IdempotencyKeyRepository) TryCreate(
	ctx context.Context,
	key string,
	endpoint string,
	requestHash string,
) (*models.IdempotencyKey, error) {
	row, err := r.q.TryCreateIdempotencyKey(ctx, sqlc.TryCreateIdempotencyKeyParams{
		Key:         key,
		Endpoint:    endpoint,
		RequestHash: requestHash,
	})
	if err != nil {
		return nil, err
	}
	result := toDomainIdempotencyKey(row)
	return &result, nil
}

func (r *IdempotencyKeyRepository) Get(
	ctx context.Context,
	key string,
	endpoint string,
) (*models.IdempotencyKey, error) {
	row, err := r.q.GetIdempotencyKey(ctx, sqlc.GetIdempotencyKeyParams{
		Key:      key,
		Endpoint: endpoint,
	})
	if err != nil {
		return nil, err
	}
	result := toDomainIdempotencyKey(row)
	return &result, nil
}

func (r *IdempotencyKeyRepository) Complete(
	ctx context.Context,
	key string,
	endpoint string,
	statusCode int,
	response json.RawMessage,
) (*models.IdempotencyKey, error) {
	row, err := r.q.CompleteIdempotencyKey(ctx, sqlc.CompleteIdempotencyKeyParams{
		StatusCode: int16(statusCode),
		Response:   response,
		Key:        key,
		Endpoint:   endpoint,
	})
	if err != nil {
		return nil, err
	}
	result := toDomainIdempotencyKey(row)
	return &result, nil
}

func toDomainIdempotencyKey(row sqlc.IdempotencyKey) models.IdempotencyKey {
	return models.IdempotencyKey{
		Key:         row.Key,
		Endpoint:    row.Endpoint,
		RequestHash: row.RequestHash,
		StatusCode:  row.StatusCode,
		Response:    row.Response,
		CreatedAt:   row.CreatedAt.Time,
	}
}
