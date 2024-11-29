package repo

import (
	"context"
	"errors"
	"fmt"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/hihoak/otus-microservices-architect/cmd/order-service/domain/order"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/repository/postgres"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v4"
)

type PostgresOrdersRepository struct {
	client *postgres.PostgresRepository
}

func NewPostgresOrdersRepository(client *postgres.PostgresRepository) *PostgresOrdersRepository {
	return &PostgresOrdersRepository{client: client}
}

func (p *PostgresOrdersRepository) CreateOrder(ctx context.Context, ord order.Order) (*order.Order, error) {
	insertBuilder := sqlbuilder.NewInsertBuilder()
	sql, args := insertBuilder.InsertInto("orders").
		Cols("user_id", "price").
		Values(ord.UserID, ord.Price).
		SQL(fmt.Sprintf("RETURNING %s", "id")).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	var id int64
	err = pgxscan.ScanOne(&id, rows)
	if err != nil {
		return nil, fmt.Errorf("unexpected err when list users: %w", err)
	}

	ord.ID = order.OrderID(id)

	return &ord, nil
}

func (p *PostgresOrdersRepository) GetOrderByID(ctx context.Context, id order.OrderID) (*order.Order, error) {
	selectBuilder := sqlbuilder.NewSelectBuilder()
	sql, args := selectBuilder.Select("id", "user_id", "price").
		From("orders").
		Where(selectBuilder.Equal("id", id)).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	acc := order.Order{}

	err = pgxscan.ScanOne(&acc, rows)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("can't find order by id %q: %s: %w", id, err.Error(), order.ErrNotFound)
		}
		return nil, fmt.Errorf("unexpected err when finding order by id %d: %w", id, err)
	}

	return &acc, nil
}
