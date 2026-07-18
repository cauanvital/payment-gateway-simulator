package repository

import (
	"context"

	"github.com/cauanvital/payment-gateway-simulator/internal/database/sqlc"
	"github.com/cauanvital/payment-gateway-simulator/internal/models"
	"github.com/google/uuid"
)

type TerminalRepository struct {
	q *sqlc.Queries
}

func NewTerminalRepository(q *sqlc.Queries) *TerminalRepository {
	return &TerminalRepository{q: q}
}

func (r *TerminalRepository) Create(ctx context.Context, merchantID uuid.UUID, serial string) (*models.Terminal, error) {
	row, err := r.q.CreateTerminal(ctx, sqlc.CreateTerminalParams{MerchantID: merchantID, Serial: serial})
	if err != nil {
		return nil, err
	}
	terminal := toDomainTerminal(row)
	return &terminal, nil
}

func (r *TerminalRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Terminal, error) {
	row, err := r.q.GetTerminalByID(ctx, id)
	if err != nil {
		return nil, err
	}
	terminal := toDomainTerminal(row)
	return &terminal, nil
}

func (r *TerminalRepository) GetBySerial(ctx context.Context, serial string) (*models.Terminal, error) {
	row, err := r.q.GetTerminalBySerial(ctx, serial)
	if err != nil {
		return nil, err
	}
	terminal := toDomainTerminal(row)
	return &terminal, nil
}

func (r *TerminalRepository) List(ctx context.Context, merchantID uuid.UUID) ([]models.Terminal, error) {
	rows, err := r.q.ListTerminals(ctx, merchantID)
	if err != nil {
		return nil, err
	}

	terminals := make([]models.Terminal, len(rows))
	for i, row := range rows {
		terminals[i] = toDomainTerminal(row)
	}
	return terminals, nil
}

func (r *TerminalRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.TerminalStatus) (*models.Terminal, error) {
	row, err := r.q.UpdateTerminalStatus(ctx, sqlc.UpdateTerminalStatusParams{ID: id, Status: sqlc.TerminalStatus(status)})
	if err != nil {
		return nil, err
	}
	terminal := toDomainTerminal(row)
	return &terminal, nil
}

func toDomainTerminal(row sqlc.Terminal) models.Terminal {
	return models.Terminal{
		ID:         row.ID,
		Serial:     row.Serial,
		MerchantID: row.MerchantID,
		Status:     models.TerminalStatus(row.Status),
		CreatedAt:  row.CreatedAt.Time,
	}
}
