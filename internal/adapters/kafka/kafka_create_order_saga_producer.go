package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hihoak/otus-microservices-architect/cmd/order-service/domain/order"
	"github.com/hihoak/otus-microservices-architect/cmd/order-service/sagas/create_order"
	"github.com/segmentio/kafka-go"
	"strconv"
)

type CreateOrderSagaCommand struct {
	ID    int64                                 `json:"id"`
	Name  create_order.CreateOrderSagaEventType `json:"name"`
	Order order.Order                           `json:"order"`
}

type ClientCreateOrderSagaCommand struct {
	writerCreateOrderSaga *kafka.Writer
}

func NewKafkaCreateOrderSagaProducer() *ClientCreateOrderSagaCommand {
	writerCreateOrderSaga := &kafka.Writer{
		Addr:     kafka.TCP("kafka.kafka.svc.cluster.local:9092"),
		Topic:    "create-order-saga-commands",
		Balancer: &kafka.LeastBytes{},
	}
	return &ClientCreateOrderSagaCommand{writerCreateOrderSaga: writerCreateOrderSaga}
}

func (c *ClientCreateOrderSagaCommand) Close() error {
	if err := c.writerCreateOrderSaga.Close(); err != nil {
		return fmt.Errorf("failed to close writerCreateOrderSaga: %w", err)
	}
	return nil
}

func (c *ClientCreateOrderSagaCommand) WriteEvent(ctx context.Context, key, message string) error {
	err := c.writerCreateOrderSaga.WriteMessages(ctx, kafka.Message{Key: []byte(key), Value: []byte(message)})
	if err != nil {
		return fmt.Errorf("failed to write message CreateOrderSagaCommand: %w", err)
	}
	return nil
}

func (c *ClientCreateOrderSagaCommand) WriteOrderEvent(ctx context.Context, name string, order *order.Order) error {
	body, err := json.Marshal(CreateOrderSagaCommand{
		ID:    int64(order.ID),
		Name:  create_order.CreateOrderSagaEventType(name),
		Order: *order,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal CreateOrderSagaCommand: %w", err)
	}
	err = c.writerCreateOrderSaga.WriteMessages(ctx, kafka.Message{Key: []byte(strconv.Itoa(int(order.ID))), Value: body})
	if err != nil {
		return fmt.Errorf("failed to write message CreateOrderSagaCommand: %w", err)
	}
	return nil
}
