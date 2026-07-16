package repository

import (
	"context"

	"github.com/cauanvital/payment-gateway-simulator/internal/database/sqlc"
	"github.com/cauanvital/payment-gateway-simulator/internal/models"
	"github.com/google/uuid"
)

type MerchantRepository struct {
	q *sqlc.Queries
}

func NewMerchantRepository(q *sqlc.Queries) *MerchantRepository {
	return &MerchantRepository{q: q}
}

func (r *MerchantRepository) Create(ctx context.Context, name string) (*models.Merchant, error) {
	row, err := r.q.CreateMerchant(ctx, name)
	if err != nil {
		return nil, err
	}
	merchant := toDomainMerchant(row)
	return &merchant, nil
}

func (r *MerchantRepository) Get(ctx context.Context, id uuid.UUID) (*models.Merchant, error) {
	row, err := r.q.GetMerchant(ctx, id)
	if err != nil {
		return nil, err
	}
	merchant := toDomainMerchant(row)
	return &merchant, nil
}

func (r *MerchantRepository) List(ctx context.Context) ([]models.Merchant, error) {
	rows, err := r.q.ListMerchants(ctx)
	if err != nil {
		return nil, err
	}

	merchants := make([]models.Merchant, len(rows))
	for i, row := range rows {
		merchants[i] = toDomainMerchant(row)
	}
	return merchants, nil
}

func toDomainMerchant(row sqlc.Merchant) models.Merchant {
	return models.Merchant{
		ID:        row.ID,
		Name:      row.Name,
		CreatedAt: row.CreatedAt.Time,
	}
}
