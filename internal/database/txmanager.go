package database

import (
	"context"

	"github.com/cauanvital/payment-gateway-simulator/internal/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

func (tm *TxManager) WithinTx(ctx context.Context, fn func(q *sqlc.Queries) error) error {
	tx, err := tm.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := fn(sqlc.New(tx)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
