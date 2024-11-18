package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hihoak/otus-microservices-architect/cmd/order-service/domain/order"
	"github.com/segmentio/kafka-go"
	"strconv"
	"time"
)

type ClientOrdersEvents struct {
	writerOrdersEvents *kafka.Writer
}

func NewKafkaOrdersEvents() *ClientOrdersEvents {
	writerOrdersEvents := &kafka.Writer{
		Addr:            kafka.TCP("kafka.kafka.svc.cluster.local:9092"),
		Topic:           "orders-events",
		Balancer:        &kafka.LeastBytes{},
		WriteBackoffMin: time.Millisecond * 50,
		WriteBackoffMax: time.Millisecond * 100,
		BatchSize:       1,
	}
	return &ClientOrdersEvents{writerOrdersEvents: writerOrdersEvents}
}

func (c *ClientOrdersEvents) Close() error {
	if err := c.writerOrdersEvents.Close(); err != nil {
		return fmt.Errorf("failed to close writerOrdersEvents: %w", err)
	}
	return nil
}

const OrderCreatedEvent = "OrderCreated"

type OrderEvent struct {
	EventType string      `json:"event_type"`
	Order     order.Order `json:"order"`
}

func (c *ClientOrdersEvents) WriteOrderCreatedEvent(ctx context.Context, order *order.Order) error {
	body, err := json.Marshal(OrderEvent{
		EventType: OrderCreatedEvent,
		Order:     *order,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal order event: %w", err)
	}
	err = c.writerOrdersEvents.WriteMessages(ctx, kafka.Message{Key: []byte(strconv.Itoa(int(order.ID))), Value: body})
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	return nil
}
