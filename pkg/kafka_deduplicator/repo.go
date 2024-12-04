package kafka_deduplicator

import (
	"context"
	"errors"
	"fmt"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v4"
	"time"
)

type Postgres interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
}

type kafkaDeduplicatorRepo struct {
	postgres Postgres
}

func newKafkaDeduplicatorRepo(postgres Postgres) *kafkaDeduplicatorRepo {
	return &kafkaDeduplicatorRepo{
		postgres: postgres,
	}
}

func (r *kafkaDeduplicatorRepo) GetEvent(ctx context.Context, tableName, id string) (bool, error) {
	selectBuilder := sqlbuilder.NewSelectBuilder()
	sql, args := selectBuilder.Select(
		"id",
	).
		From(tableName).
		Where(selectBuilder.Equal("id", id)).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	var resID string
	rows, err := r.postgres.Query(ctx, sql, args...)
	if err != nil {
		return false, fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	err = pgxscan.ScanOne(&resID, rows)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("unexpected err when finding event id: %s: %w", id, err)
	}

	return true, nil
}

func (r *kafkaDeduplicatorRepo) InsertEvent(ctx context.Context, tableName, id string) error {
	insertBuilder := sqlbuilder.NewInsertBuilder()
	sql, args := insertBuilder.InsertInto(tableName).
		Cols("id", "processed_at").
		Values(id, time.Now().UTC()).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := r.postgres.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	return nil
}
