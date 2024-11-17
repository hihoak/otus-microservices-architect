package account

import "errors"

var (
	ErrUnsufficientFunds = errors.New("unsufficient funds")
	ErrNotFound          = errors.New("account not found")
)

type AccountID int64

type Account struct {
	ID     AccountID `db:"id"`
	UserID int64     `db:"user_id"`
	Amount int64     `db:"amount"`
}

func NewAccount(userID int64) Account {
	return Account{
		UserID: userID,
		Amount: 0,
	}
}

func (a *Account) TopUp(amount int64) {
	a.Amount += amount
}

func (a *Account) Withdraw(amount int64) error {
	if a.Amount-amount < 0 {
		return ErrUnsufficientFunds
	}
	a.Amount -= amount
	return nil
}
