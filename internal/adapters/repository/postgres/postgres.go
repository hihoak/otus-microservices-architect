package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(ctx context.Context, dsn string) (*PostgresRepository, error) {
	pool, err := pgxpool.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connection to postgres: %w", err)
	}

	return &PostgresRepository{
		pool: pool,
	}, nil
}

func (repo *PostgresRepository) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return repo.pool.Query(ctx, sql, args...)
}
