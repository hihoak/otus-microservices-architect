package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hihoak/otus-microservices-architect/internal/domain/user"
	"github.com/segmentio/kafka-go"
	"strconv"
)

type ClientUsersEvents struct {
	writerUsersEvents *kafka.Writer
}

func NewKafkaUsersEvents() *ClientUsersEvents {
	writerUsersEvents := &kafka.Writer{
		Addr:     kafka.TCP("kafka.kafka.svc.cluster.local:9092"),
		Topic:    "users-events",
		Balancer: &kafka.LeastBytes{},
	}
	return &ClientUsersEvents{writerUsersEvents: writerUsersEvents}
}

func (c *ClientUsersEvents) Close() error {
	if err := c.writerUsersEvents.Close(); err != nil {
		return fmt.Errorf("failed to close writerUsersEvents: %w", err)
	}
	return nil
}

const UserCreatedEvent = "UserCreated"

type UserEvent struct {
	EventType string    `json:"event_type"`
	User      user.User `json:"user"`
}

func (c *ClientUsersEvents) WriteUserCreatedEvent(ctx context.Context, user *user.User) error {
	body, err := json.Marshal(UserEvent{
		EventType: UserCreatedEvent,
		User:      *user,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}
	err = c.writerUsersEvents.WriteMessages(ctx, kafka.Message{Key: []byte(strconv.Itoa(int(user.ID))), Value: body})
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	return nil
}
