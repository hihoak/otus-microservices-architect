package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
)

type PostgresRepository struct {
	Conn *pgx.Conn
}

func NewPostgresRepository(ctx context.Context, dsn string) (*PostgresRepository, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connection to postgres: %w", err)
	}

	return &PostgresRepository{
		Conn: conn,
	}, nil
}
