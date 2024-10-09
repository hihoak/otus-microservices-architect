package repo

import (
	"context"
	"errors"
	"fmt"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/repository/postgres"
	"github.com/hihoak/otus-microservices-architect/internal/domain/user"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v4"
)

type PostgresUsersRepository struct {
	client *postgres.PostgresRepository
}

func NewPostgresUsersRepository(client *postgres.PostgresRepository) *PostgresUsersRepository {
	return &PostgresUsersRepository{client: client}
}

func (p *PostgresUsersRepository) GetUser(ctx context.Context, id user.UserID) (*user.User, error) {
	selectBuilder := sqlbuilder.NewSelectBuilder()
	sql, args := selectBuilder.Select(
		"id", "first_name", "sur_name", "age",
	).
		From("users").
		Where(selectBuilder.Equal("id", id)).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	var u user.User
	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	err = pgxscan.ScanOne(&u, rows)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("can't find user by id %q: %s: %w", id, err.Error(), user.ErrNotFound)
		}
		return nil, fmt.Errorf("unexpected err when finding auto user by id %d: %w", id, err)
	}

	return &u, nil
}

func (p *PostgresUsersRepository) CreateUser(ctx context.Context, u user.User) error {
	insertBuilder := sqlbuilder.NewInsertBuilder()
	sql, args := insertBuilder.InsertInto("users").
		Cols("first_name", "sur_name", "age").
		Values(u.Firstname, u.Surname, u.Age).
		SQL(fmt.Sprintf("RETURNING %s", "id")).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	return nil
}

func (p *PostgresUsersRepository) UpdateUser(ctx context.Context, u *user.User) error {
	updateBuilder := sqlbuilder.NewUpdateBuilder()
	sql, args := updateBuilder.Update("users").
		Set(
			updateBuilder.Equal("id", u.ID),
			updateBuilder.Equal("first_name", u.Firstname),
			updateBuilder.Equal("sur_name", u.Surname),
			updateBuilder.Equal("age", u.Age),
		).
		Where(updateBuilder.Equal("id", u.ID)).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	return nil
}

func (p *PostgresUsersRepository) DeleteUser(ctx context.Context, id user.UserID) error {
	updateBuilder := sqlbuilder.NewDeleteBuilder()
	sql, args := updateBuilder.DeleteFrom("users").
		Where(updateBuilder.Equal("id", id)).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	return nil
}

func (p *PostgresUsersRepository) ListUser(ctx context.Context) ([]user.User, error) {
	updateBuilder := sqlbuilder.NewSelectBuilder()
	sql, args := updateBuilder.Select("id", "first_name", "sur_name", "age").
		From("users").
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	users := []user.User{}
	rows, err := p.client.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("unexpected err when creating query: %w", err)
	}
	defer rows.Close()

	err = pgxscan.ScanAll(&users, rows)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("unexpected err when list users: %w", err)
	}

	return users, nil
}
