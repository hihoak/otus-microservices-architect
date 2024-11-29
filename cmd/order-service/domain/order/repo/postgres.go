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
	"strconv"
	"strings"
)

type PostgresOrdersRepository struct {
	client *postgres.PostgresRepository
}

type RepoOrder struct {
	ID                order.OrderID `db:"id"`
	Status            string        `json:"status"`
	UserID            int64         `db:"user_id"`
	Price             int64         `db:"price"`
	ItemIDsWithStocks string        `db:"item_ids_with_stocks"`
	DeliverySlotID    int64         `db:"delivery_slot_id"`
}

func NewPostgresOrdersRepository(client *postgres.PostgresRepository) *PostgresOrdersRepository {
	return &PostgresOrdersRepository{client: client}
}

func convertHstoreToStr(dict map[int64]int64) string {
	builder := strings.Builder{}
	idx := 0
	for k, v := range dict {
		builder.WriteString(fmt.Sprintf("%d=>%d", k, v))
		if idx < len(dict)-1 {
			idx++
			builder.WriteString(",")
		}
	}
	return builder.String()
}

func fromHstoreToMap(rawHstore string) (map[int64]int64, error) {
	res := make(map[int64]int64)
	for _, data := range strings.Split(rawHstore, ",") {
		keyValue := strings.SplitN(data, "=>", 2)
		if len(keyValue) != 2 {
			return nil, fmt.Errorf("invalid hstore format: must be 2 length")
		}
		key, err := strconv.ParseInt(strings.Trim(strings.TrimSpace(keyValue[0]), "\""), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("key must be an integer: %v: %w", strings.Trim(strings.TrimSpace(keyValue[0]), "\""), err)
		}
		value, err := strconv.ParseInt(strings.Trim(strings.TrimSpace(keyValue[1]), "\""), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("value must be an integer: %v: %w", strings.Trim(strings.TrimSpace(keyValue[1]), "\""), err)
		}

		res[key] = value
	}
	return res, nil
}

func (p *PostgresOrdersRepository) CreateOrder(ctx context.Context, ord order.Order) (*order.Order, error) {
	insertBuilder := sqlbuilder.NewInsertBuilder()
	sql, args := insertBuilder.InsertInto("orders").
		Cols("status", "user_id", "price", "item_ids_with_stocks", "delivery_slot_id").
		Values(ord.Status, ord.UserID, ord.Price, convertHstoreToStr(ord.ItemIDsWithStocks), ord.DeliverySlotID).
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
	sql, args := selectBuilder.Select("id", "status", "user_id", "price", "item_ids_with_stocks", "delivery_slot_id").
		From("orders").
		Where(selectBuilder.Equal("id", id)).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	repoOrder := RepoOrder{}

	err = pgxscan.ScanOne(&repoOrder, rows)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("can't find order by id %q: %s: %w", id, err.Error(), order.ErrNotFound)
		}
		return nil, fmt.Errorf("unexpected err when finding order by id %d: %w", id, err)
	}

	itemIDsWithStocks, err := fromHstoreToMap(repoOrder.ItemIDsWithStocks)
	if err != nil {
		return nil, fmt.Errorf("unexpected err when finding item_ids_with_stocks: %w", err)
	}

	ord := order.Order{
		ID:                repoOrder.ID,
		Status:            repoOrder.Status,
		UserID:            repoOrder.UserID,
		Price:             repoOrder.Price,
		ItemIDsWithStocks: itemIDsWithStocks,
		DeliverySlotID:    repoOrder.DeliverySlotID,
	}

	return &ord, nil
}

func (p *PostgresOrdersRepository) UpdateOrder(ctx context.Context, ord order.Order) error {
	updateBuilder := sqlbuilder.NewUpdateBuilder()
	sql, args := updateBuilder.Update("orders").
		Set(updateBuilder.Equal("id", ord.ID),
			updateBuilder.Equal("status", ord.Status),
			updateBuilder.Equal("user_id", ord.UserID),
			updateBuilder.Equal("price", ord.Price),
			updateBuilder.Equal("item_ids_with_stocks", convertHstoreToStr(ord.ItemIDsWithStocks)),
			updateBuilder.Equal("delivery_slot_id", ord.DeliverySlotID),
		).Where(updateBuilder.Equal("id", ord.ID)).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	if err = rows.Err(); err != nil {
		return fmt.Errorf("unexpected err when fetching rows: %w", err)
	}

	return nil
}
