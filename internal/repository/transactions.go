package repository

import "github.com/cauanvital/payment-gateway-simulator/internal/database/sqlc"

type PaymentRepository struct {
	q *sqlc.Queries
}

func NewPaymentRepository(q *sqlc.Queries) *PaymentRepository {
	return &PaymentRepository{q: q}
}
