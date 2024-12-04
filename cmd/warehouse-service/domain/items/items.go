package items

import "errors"

var (
	ErrNotEnoughItems = errors.New("not enough items")
	ErrNotFound       = errors.New("item not found")
)

type ItemID int64

type Item struct {
	ID    ItemID `db:"id"`
	Count uint64 `db:"count"`
}

func NewItem(count uint64) Item {
	return Item{
		Count: count,
	}
}

func (i *Item) Reserve(reserveCount uint64) error {
	if reserveCount > i.Count {
		return ErrNotEnoughItems
	}
	i.Count -= reserveCount
	return nil
}

func (i *Item) Add(count uint64) {
	i.Count += count
}
