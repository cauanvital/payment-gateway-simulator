package repository

import "github.com/cauanvital/payment-gateway-simulator/internal/database/sqlc"

type PaymentEventRepository struct {
	q *sqlc.Queries
}

func NewPaymentEventRepository(q *sqlc.Queries) *PaymentEventRepository {
	return &PaymentEventRepository{q: q}
}
