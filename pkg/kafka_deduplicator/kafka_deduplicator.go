package kafka_deduplicator

import (
	"context"
	"fmt"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/repository/postgres"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/logger"
)

type KafkaDeduplicator struct {
	tableName string
	repo      *kafkaDeduplicatorRepo
}

func NewKafkaDeduplicator(postgres Postgres, tableName string) *KafkaDeduplicator {
	return &KafkaDeduplicator{
		tableName: tableName,
		repo:      newKafkaDeduplicatorRepo(postgres),
	}
}

func (d *KafkaDeduplicator) WithDeduplicate(ctx context.Context, eventID string, f func(ctx context.Context) error) error {
	exists, err := d.repo.GetEvent(ctx, d.tableName, eventID)
	if err != nil {
		return fmt.Errorf("get event: %w", err)
	}
	if exists {
		logger.Log.Info("event %q deduplicated", eventID)
		return nil
	}
	if err = f(context.WithValue(ctx, postgres.DeduplicationFuncKey{}, postgres.DeduplicationFunc(func(ctx context.Context) error {
		insertErr := d.repo.InsertEvent(ctx, d.tableName, eventID)
		if insertErr != nil {
			return fmt.Errorf("insert deduplication event: %w", insertErr)
		}
		return nil
	}))); err != nil {
		return fmt.Errorf("with deduplication: %w", err)
	}
	return nil
}
