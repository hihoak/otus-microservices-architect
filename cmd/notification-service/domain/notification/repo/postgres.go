package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/hihoak/otus-microservices-architect/cmd/notification-service/domain/notification"
	"github.com/hihoak/otus-microservices-architect/cmd/order-service/domain/order"
	"github.com/hihoak/otus-microservices-architect/cmd/order-service/sagas/create_order"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/kafka"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/repository/postgres"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v4"
	"strconv"
)

type PostgresNotificationsRepository struct {
	client *postgres.PostgresRepository
}

func NewPostgresNotificationsRepository(client *postgres.PostgresRepository) *PostgresNotificationsRepository {
	return &PostgresNotificationsRepository{client: client}
}

func (p *PostgresNotificationsRepository) CreateNotification(ctx context.Context, notification notification.Notification) error {
	insertBuilder := sqlbuilder.NewInsertBuilder()
	sql, args := insertBuilder.InsertInto("notifications").
		Cols("user_id", "text", "date").
		Values(notification.UserID, notification.Text, notification.Date).
		SQL(fmt.Sprintf("RETURNING %s", "id")).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	return nil
}

func (p *PostgresNotificationsRepository) ListByUserID(ctx context.Context, userId int64) ([]notification.Notification, error) {
	selectBuilder := sqlbuilder.NewSelectBuilder()
	sql, args := selectBuilder.Select("id", "user_id", "text", "date").
		From("notifications").
		Where(selectBuilder.Equal("user_id", userId)).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	acc := []notification.Notification{}

	err = pgxscan.ScanAll(&acc, rows)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("unexpected err when finding notifications by user_id %d: %w", userId, err)
	}

	return acc, nil
}

func (p *PostgresNotificationsRepository) CreateEvent(ctx context.Context, eventName create_order.CreateOrderSagaEventType, ord *order.Order) error {
	insertBuilder := sqlbuilder.NewInsertBuilder()
	body, err := json.Marshal(kafka.CreateOrderSagaCommand{
		ID:    int64(ord.ID),
		Name:  eventName,
		Order: *ord,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal CreateOrderSagaCommand: %w", err)
	}

	sql, args := insertBuilder.InsertInto("transactional_outbox_create_order_saga_events_notification_svc").
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
