package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type tXKey struct{}

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
	tx, ok := repo.txFromContext(ctx)
	if !ok {
		return repo.pool.Query(ctx, sql, args...)
	}
	return tx.Query(ctx, sql, args...)
}

func (repo *PostgresRepository) BeginTxFunc(ctx context.Context, f func(ctx context.Context) error) error {
	tx, err := repo.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:       pgx.RepeatableRead,
		AccessMode:     pgx.ReadWrite,
		DeferrableMode: pgx.NotDeferrable,
	})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	ctx = context.WithValue(ctx, tXKey{}, tx)

	err = f(ctx)

	if err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return fmt.Errorf("function: %w: rollback: %w", err, rollbackErr)
		}
		return fmt.Errorf("function: %w", err)
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return fmt.Errorf("commit: %w", commitErr)
	}

	return nil
}

func (repo *PostgresRepository) txFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(tXKey{}).(pgx.Tx)
	if !ok {
		return nil, false
	}
	return tx, true
}
