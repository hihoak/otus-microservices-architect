package repo

import (
	"context"
	"errors"
	"fmt"
	"github.com/georgysavva/scany/pgxscan"
	delivery_slots "github.com/hihoak/otus-microservices-architect/cmd/delivery-service/domain/delivery-slots"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/repository/postgres"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v4"
)

type PostgresDeliverySlotsRepository struct {
	client *postgres.PostgresRepository
}

func NewPostgresDeliverySlotsRepository(client *postgres.PostgresRepository) *PostgresDeliverySlotsRepository {
	return &PostgresDeliverySlotsRepository{client: client}
}

func (p *PostgresDeliverySlotsRepository) CreateDeliverySlot(ctx context.Context, deliverySlot delivery_slots.DeliverySlot) (delivery_slots.DeliverySlot, error) {
	insertBuilder := sqlbuilder.NewInsertBuilder()
	sql, args := insertBuilder.InsertInto("delivery_slots").
		Cols("from_time", "to_time", "deliveries_left").
		Values(deliverySlot.FromTime, deliverySlot.ToTime, deliverySlot.DeliveriesLeft).
		SQL(fmt.Sprintf("RETURNING %s", "id")).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return delivery_slots.DeliverySlot{}, fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	var id int64
	err = pgxscan.ScanOne(&id, rows)
	if err != nil {
		return delivery_slots.DeliverySlot{}, fmt.Errorf("unexpected err when scan delivery slot id: %w", err)
	}

	deliverySlot.ID = delivery_slots.DeliverySlotID(id)
	return deliverySlot, nil
}

func (p *PostgresDeliverySlotsRepository) GetDeliverySlotByID(ctx context.Context, id uint64) (*delivery_slots.DeliverySlot, error) {
	selectBuilder := sqlbuilder.NewSelectBuilder()
	sql, args := selectBuilder.Select("id", "from_time", "to_time", "deliveries_left").
		From("delivery_slots").
		Where(selectBuilder.Equal("id", id)).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	deliverySlot := delivery_slots.DeliverySlot{}

	err = pgxscan.ScanOne(&deliverySlot, rows)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("can't find delivery slot by id %q: %s: %w", id, err.Error(), delivery_slots.ErrNotFound)
		}
		return nil, fmt.Errorf("unexpected err when finding delivery slot by id %d: %w", id, err)
	}

	return &deliverySlot, nil
}

func (p *PostgresDeliverySlotsRepository) UpdateDeliverySlot(ctx context.Context, deliverySlot *delivery_slots.DeliverySlot) error {
	updateBuilder := sqlbuilder.NewUpdateBuilder()
	sql, args := updateBuilder.Update("delivery_slots").
		Set(
			updateBuilder.Equal("id", deliverySlot.ID),
			updateBuilder.Equal("from_time", deliverySlot.FromTime),
			updateBuilder.Equal("to_time", deliverySlot.ToTime),
			updateBuilder.Equal("deliveries_left", deliverySlot.DeliveriesLeft),
		).
		Where(updateBuilder.Equal("id", deliverySlot.ID)).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	return nil
}
