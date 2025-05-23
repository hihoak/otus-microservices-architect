package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/hihoak/otus-microservices-architect/cmd/order-service/domain/order"
	"github.com/hihoak/otus-microservices-architect/cmd/order-service/sagas/create_order"
	"github.com/hihoak/otus-microservices-architect/cmd/warehouse-service/domain/items"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/kafka"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/repository/postgres"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v4"
	"strconv"
)

type PostgresItemsRepository struct {
	client *postgres.PostgresRepository
}

func NewPostgresItemsRepository(client *postgres.PostgresRepository) *PostgresItemsRepository {
	return &PostgresItemsRepository{client: client}
}

func (p *PostgresItemsRepository) CreateItem(ctx context.Context, item items.Item) (items.Item, error) {
	insertBuilder := sqlbuilder.NewInsertBuilder()
	sql, args := insertBuilder.InsertInto("items").
		Cols("count").
		Values(item.Count).
		SQL(fmt.Sprintf("RETURNING %s", "id")).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return items.Item{}, fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	var id int64
	err = pgxscan.ScanOne(&id, rows)
	if err != nil {
		return items.Item{}, fmt.Errorf("unexpected err when scan item id: %w", err)
	}

	item.ID = items.ItemID(id)
	return item, nil
}

func (p *PostgresItemsRepository) GetItemByID(ctx context.Context, id uint64) (*items.Item, error) {
	selectBuilder := sqlbuilder.NewSelectBuilder()
	sql, args := selectBuilder.Select("id", "count").
		From("items").
		Where(selectBuilder.Equal("id", id)).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	item := items.Item{}

	err = pgxscan.ScanOne(&item, rows)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("can't find item by id %q: %s: %w", id, err.Error(), items.ErrNotFound)
		}
		return nil, fmt.Errorf("unexpected err when finding item by id %d: %w", id, err)
	}

	return &item, nil
}

func (p *PostgresItemsRepository) UpdateItem(ctx context.Context, item *items.Item) error {
	updateBuilder := sqlbuilder.NewUpdateBuilder()
	sql, args := updateBuilder.Update("items").
		Set(
			updateBuilder.Equal("id", item.ID),
			updateBuilder.Equal("count", item.Count),
		).
		Where(updateBuilder.Equal("id", item.ID)).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	return nil
}

func (p *PostgresItemsRepository) CreateEvent(ctx context.Context, eventName create_order.CreateOrderSagaEventType, ord *order.Order) error {
	insertBuilder := sqlbuilder.NewInsertBuilder()
	body, err := json.Marshal(kafka.CreateOrderSagaCommand{
		ID:    int64(ord.ID),
		Name:  eventName,
		Order: *ord,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal CreateOrderSagaCommand: %w", err)
	}

	sql, args := insertBuilder.InsertInto("transactional_outbox_create_order_saga_events_warehouse_service").
		Cols("key", "message").
		Values(strconv.Itoa(int(ord.ID)), body).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	return nil
}
