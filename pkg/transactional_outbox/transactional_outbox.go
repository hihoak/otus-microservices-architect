package transactional_outbox

import (
	"context"
	"fmt"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/repository/postgres"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/logger"
	"log"
	"time"
)

type KafkaProducer interface {
	WriteEvent(ctx context.Context, key, message string) error
}

type TransactionalOutbox struct {
	ctx            context.Context
	cancel         context.CancelFunc
	postgresClient *postgres.PostgresRepository
	repo           *transactionalOutboxRepo
	kafkaProducer  KafkaProducer
	tableName      string
}

func NewTransactionalOutbox(ctx context.Context, tableName string, producer KafkaProducer, postgresClient *postgres.PostgresRepository) *TransactionalOutbox {
	ctx, cancel := context.WithCancel(ctx)
	return &TransactionalOutbox{
		ctx:            ctx,
		cancel:         cancel,
		repo:           newTransactionalOutboxRepo(postgresClient),
		tableName:      tableName,
		postgresClient: postgresClient,
		kafkaProducer:  producer,
	}
}

func (t *TransactionalOutbox) Start() {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
	mainLoop:
		for {
			select {
			case <-t.ctx.Done():
				break mainLoop
			case <-ticker.C:
				logger.Log.Info("start outbox")
				if err := t.postgresClient.BeginTxFunc(t.ctx, func(ctx context.Context) error {
					tasks, err := t.repo.GetTasks(ctx, t.tableName)
					if err != nil {
						return fmt.Errorf("get tasks: %w", err)
					}
					if len(tasks) == 0 {
						return nil
					}
					var processedTasks int64
					for _, task := range tasks {
						if err := t.kafkaProducer.WriteEvent(ctx, task.Key, task.Message); err != nil {
							logger.Log.Error("failed to push events: ", err)
							break
						}
						processedTasks++
					}
					err = t.repo.DeleteTasks(ctx, t.tableName, tasks[:processedTasks])
					if err != nil {
						return fmt.Errorf("delete tasks: %w", err)
					}
					logger.Log.Info("processed outbox tasks: ", processedTasks)
					return nil
				}); err != nil {
					logger.Log.Error("Failed to do transactional outbox transaction: ", err)
					break mainLoop
				}
			}
		}
		log.Fatal("transactional outbox got an error")
	}()
	return
}

func (t *TransactionalOutbox) Close() {
	t.cancel()
}
