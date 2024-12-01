package order

import (
	"errors"
	"github.com/hihoak/otus-microservices-architect/cmd/order-service/sagas/create_order"
)

var (
	ErrNotFound = errors.New("order not found")
)

type OrderID int64

type Order struct {
	ID                OrderID         `db:"id"`
	Status            string          `db:"status"`
	UserID            int64           `db:"user_id"`
	Price             int64           `db:"price"`
	ItemIDsWithStocks map[int64]int64 `db:"item_ids_with_stocks"`
	DeliverySlotID    int64           `db:"delivery_slot_id"`
}

func NewOrder(userID int64, price int64, itemIDsWithStocks map[int64]int64, deliverySlotID int64) Order {
	return Order{
		Status:            create_order.WithdrawMoneyPending,
		UserID:            userID,
		Price:             price,
		ItemIDsWithStocks: itemIDsWithStocks,
		DeliverySlotID:    deliverySlotID,
	}
}
