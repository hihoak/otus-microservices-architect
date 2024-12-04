package delivery_slots

import (
	"errors"
	"time"
)

var (
	ErrNotEnoughDeliveries = errors.New("not enough deliveries to this time")
	ErrNotFound            = errors.New("delivery slot not found")
)

type DeliverySlotID int64

type DeliverySlot struct {
	ID             DeliverySlotID `db:"id"`
	FromTime       time.Time      `db:"from_time"`
	ToTime         time.Time      `db:"to_time"`
	DeliveriesLeft int64          `db:"deliveries_left"`
}

func NewDeliverySlot(fromTime, toTime time.Time, deliveriesLeft int64) DeliverySlot {
	return DeliverySlot{
		FromTime:       fromTime,
		ToTime:         toTime,
		DeliveriesLeft: deliveriesLeft,
	}
}

func (d *DeliverySlot) Reserve() error {
	if d.DeliveriesLeft <= 0 {
		return ErrNotEnoughDeliveries
	}
	d.DeliveriesLeft -= 1
	return nil
}
