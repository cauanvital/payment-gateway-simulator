package database

import (
	"context"

	"github.com/cauanvital/payment-gateway-simulator/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool abre o pool de conexões com o Postgres e valida a conectividade
// com um Ping, falhando rápido se o banco estiver inacessível no start.
func NewPool(ctx context.Context, cfg config.DBConfig) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, cfg.DSN())
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
