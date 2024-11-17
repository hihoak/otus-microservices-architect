package order

import "errors"

var (
	ErrNotFound = errors.New("order not found")
)

type OrderID int64

type Order struct {
	ID     OrderID `db:"id"`
	UserID int64   `db:"user_id"`
	Price  int64   `db:"price"`
}

func NewOrder(userID int64, price int64) Order {
	return Order{
		UserID: userID,
		Price:  price,
	}
}
