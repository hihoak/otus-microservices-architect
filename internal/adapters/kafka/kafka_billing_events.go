package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"strconv"
)

type ClientBillingEvents struct {
	writerBillingEvents *kafka.Writer
}

func NewKafkaBillingEvents() *ClientBillingEvents {
	writerBillingEvents := &kafka.Writer{
		Addr:     kafka.TCP("kafka.kafka.svc.cluster.local:9092"),
		Topic:    "billing-events",
		Balancer: &kafka.LeastBytes{},
	}
	return &ClientBillingEvents{writerBillingEvents: writerBillingEvents}
}

func (c *ClientBillingEvents) Close() error {
	if err := c.writerBillingEvents.Close(); err != nil {
		return fmt.Errorf("failed to close writerBillingEvents: %w", err)
	}
	return nil
}

const (
	MoneyWithdrawSucceededEvent = "MoneyWithdrawSucceeded"
	MoneyWithdrawFailedEvent    = "MoneyWithdrawFailed"
)

type BillingEvent struct {
	EventType string `json:"event_type"`
	UserID    int64  `json:"user_id"`
}

func (c *ClientBillingEvents) WriteBillingEvent(ctx context.Context, eventType string, userID int64) error {
	body, err := json.Marshal(BillingEvent{
		EventType: eventType,
		UserID:    userID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal Billing event: %w", err)
	}
	err = c.writerBillingEvents.WriteMessages(ctx, kafka.Message{Key: []byte(strconv.Itoa(int(userID))), Value: body})
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	return nil
}
