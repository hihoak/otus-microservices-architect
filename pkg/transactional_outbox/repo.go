package transactional_outbox

import (
	"context"
	"fmt"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jackc/pgx/v4"
)

type Postgres interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
}

type transactionalOutboxRepo struct {
	postgres Postgres
}

func newTransactionalOutboxRepo(postgres Postgres) *transactionalOutboxRepo {
	return &transactionalOutboxRepo{
		postgres: postgres,
	}
}

type Task struct {
	ID      int64  `database:"id"`
	Key     string `database:"key"`
	Message string `database:"message"`
}

func (t *transactionalOutboxRepo) GetTasks(ctx context.Context, tableName string) ([]Task, error) {
	selectBuilder := sqlbuilder.NewSelectBuilder()
	sql, args := selectBuilder.Select(
		"id", "key", "message",
	).
		From(tableName).
		OrderBy("id").
		Limit(100).
		BuildWithFlavor(sqlbuilder.PostgreSQL)

	tasks := []Task{}
	if err := pgxscan.Select(ctx, t.postgres, &tasks, sql, args...); err != nil {
		return nil, fmt.Errorf("select tasks: %w", err)
	}

	return tasks, nil
}

func (t *transactionalOutboxRepo) DeleteTasks(ctx context.Context, tableName string, tasks []Task) error {
	deleteBuilder := sqlbuilder.NewDeleteBuilder()
	tasksID := make([]interface{}, len(tasks))
	for idx, task := range tasks {
		tasksID[idx] = task.ID
	}
	sql, args := deleteBuilder.
		DeleteFrom(tableName).
		Where(deleteBuilder.In("id", tasksID...)).BuildWithFlavor(sqlbuilder.PostgreSQL)
	rows, err := t.postgres.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("delete tasks: %w", err)
	}
	defer rows.Close()
	return nil
}
