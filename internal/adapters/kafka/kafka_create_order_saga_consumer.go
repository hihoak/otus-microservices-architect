package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/segmentio/kafka-go"
	"time"
)

var (
	ErrUnmarshall = errors.New("error unmarshalling json")
)

type KafkaCreateOrderSagaConsumer struct {
	readerCreateOrderSaga *kafka.Reader
}

func NewKafkaCreateOrderSagaConsumer(consumerGroupName string) *KafkaCreateOrderSagaConsumer {
	readerCreateOrderSaga := kafka.NewReader(kafka.ReaderConfig{
		Brokers:               []string{"kafka.kafka.svc.cluster.local:9092"},
		GroupID:               consumerGroupName,
		Topic:                 "create-order-saga-commands",
		MaxWait:               time.Millisecond * 200,
		JoinGroupBackoff:      time.Millisecond * 500,
		HeartbeatInterval:     time.Millisecond * 200,
		WatchPartitionChanges: true,
	})

	return &KafkaCreateOrderSagaConsumer{
		readerCreateOrderSaga: readerCreateOrderSaga,
	}
}

func (c *KafkaCreateOrderSagaConsumer) Close() error {
	if err := c.readerCreateOrderSaga.Close(); err != nil {
		return fmt.Errorf("failed to close readerCreateOrderSaga: %w", err)
	}
	return nil
}

func (c *KafkaCreateOrderSagaConsumer) FetchMessage(ctx context.Context) (CreateOrderSagaCommand, kafka.Message, error) {
	msg, err := c.readerCreateOrderSaga.FetchMessage(ctx)
	if err != nil {
		return CreateOrderSagaCommand{}, msg, fmt.Errorf("failed to fetch message: %w", err)
	}
	var message CreateOrderSagaCommand
	if err := json.Unmarshal(msg.Value, &message); err != nil {
		return CreateOrderSagaCommand{}, msg, fmt.Errorf("failed to unmarshal message: %w: %w", err, ErrUnmarshall)
	}

	return message, msg, nil
}

func (c *KafkaCreateOrderSagaConsumer) CommitMessage(ctx context.Context, message kafka.Message) error {
	return c.readerCreateOrderSaga.CommitMessages(ctx, message)
}
