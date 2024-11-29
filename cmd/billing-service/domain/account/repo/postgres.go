package repo

import (
	"context"
	"errors"
	"fmt"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/hihoak/otus-microservices-architect/cmd/billing-service/domain/account"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/repository/postgres"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v4"
)

type PostgresAccountsRepository struct {
	client *postgres.PostgresRepository
}

func NewPostgresAccountsRepository(client *postgres.PostgresRepository) *PostgresAccountsRepository {
	return &PostgresAccountsRepository{client: client}
}

func (p *PostgresAccountsRepository) CreateAccount(ctx context.Context, acc account.Account) error {
	insertBuilder := sqlbuilder.NewInsertBuilder()
	sql, args := insertBuilder.InsertInto("accounts").
		Cols("user_id", "amount").
		Values(acc.UserID, acc.Amount).
		SQL(fmt.Sprintf("RETURNING %s", "id")).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	return nil
}

func (p *PostgresAccountsRepository) GetAccountByID(ctx context.Context, id int64) (*account.Account, error) {
	selectBuilder := sqlbuilder.NewSelectBuilder()
	sql, args := selectBuilder.Select("id", "user_id", "amount").
		From("accounts").
		Where(selectBuilder.Equal("id", id)).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	acc := account.Account{}

	err = pgxscan.ScanOne(&acc, rows)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("can't find account by id %q: %s: %w", id, err.Error(), account.ErrNotFound)
		}
		return nil, fmt.Errorf("unexpected err when finding account by id %d: %w", id, err)
	}

	return &acc, nil
}

func (p *PostgresAccountsRepository) GetAccountByUserID(ctx context.Context, user_id int64) (*account.Account, error) {
	selectBuilder := sqlbuilder.NewSelectBuilder()
	sql, args := selectBuilder.Select("id", "user_id", "amount").
		From("accounts").
		Where(selectBuilder.Equal("user_id", user_id)).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	acc := account.Account{}

	err = pgxscan.ScanOne(&acc, rows)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("can't find account by user_id %q: %s: %w", user_id, err.Error(), account.ErrNotFound)
		}
		return nil, fmt.Errorf("unexpected err when finding account by user_id %d: %w", user_id, err)
	}

	return &acc, nil
}

func (p *PostgresAccountsRepository) UpdateAccount(ctx context.Context, acc *account.Account) error {
	updateBuilder := sqlbuilder.NewUpdateBuilder()
	sql, args := updateBuilder.Update("accounts").
		Set(
			updateBuilder.Equal("user_id", acc.UserID),
			updateBuilder.Equal("amount", acc.Amount),
		).
		Where(updateBuilder.Equal("id", acc.ID)).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	return nil
}
