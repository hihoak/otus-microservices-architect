package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hihoak/otus-microservices-architect/cmd/notification-service/domain/notification"
	"github.com/hihoak/otus-microservices-architect/cmd/notification-service/domain/notification/repo"
	"github.com/hihoak/otus-microservices-architect/cmd/order-service/sagas/create_order"
	kafka2 "github.com/hihoak/otus-microservices-architect/internal/adapters/kafka"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/repository/postgres"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/config"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/logger"
	"github.com/hihoak/otus-microservices-architect/pkg/kafka_deduplicator"
	"github.com/hihoak/otus-microservices-architect/pkg/transactional_outbox"
	"log"
	"net/http"
	"strconv"
	"time"
)

type CreateNotificationBody struct {
	UserID int64  `json:"user_id"`
	Text   string `json:"text"`
}

var notificationsRepository *repo.PostgresNotificationsRepository
var createOrderSagaProducer *kafka2.ClientCreateOrderSagaCommand
var postgresClient *postgres.PostgresRepository

func main() {
	ctx := context.Background()

	appRouter := gin.Default()

	var err error
	postgresClient, err = postgres.NewPostgresRepository(ctx, config.Cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("init postgres client: %v", err)
	}

	createOrderSagaProducer = kafka2.NewKafkaCreateOrderSagaProducer()

	notificationsRepository = repo.NewPostgresNotificationsRepository(postgresClient)

	outbox := transactional_outbox.NewTransactionalOutbox(ctx, "transactional_outbox_create_order_saga_events_notification_svc", createOrderSagaProducer, postgresClient)
	outbox.Start()
	defer outbox.Close()

	appRouter.POST("/notifications", func(c *gin.Context) {
		body := CreateNotificationBody{}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if err := notificationsRepository.CreateNotification(ctx, notification.NewNotification(body.UserID, body.Text, time.Now())); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{})
	})

	appRouter.GET("/notifications", func(c *gin.Context) {
		userIDRaw, ok := c.GetQuery("user_id")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required query parameter"})
			return
		}
		userID, err := strconv.Atoi(userIDRaw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("wrong id format it must be int: %s", err.Error())})
			return
		}

		notifications, err := notificationsRepository.ListByUserID(ctx, int64(userID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"notifications": notifications,
		})
	})

	listenCreateOrderSagaEvents(ctx)

	logger.Log.Info("starting service on address 0.0.0.0:9000...")
	appRouter.Run(fmt.Sprintf(":%s", "9000"))
}

func listenCreateOrderSagaEvents(ctx context.Context) {
	// make a new reader that consumes from topic-A
	consumer := kafka2.NewKafkaCreateOrderSagaConsumer("notifications-service")
	deduplicator := kafka_deduplicator.NewKafkaDeduplicator(postgresClient, "processed_create_order_saga_commands_notification_service")

	go func() {
		defer func() {
			if err := consumer.Close(); err != nil {
				logger.Log.Error("failed to close reader:", err)
			}
		}()

	readMessagesLoop:
		for {
			msg, kfkMsg, err := consumer.FetchMessage(ctx)
			if err != nil {
				if errors.Is(err, kafka2.ErrUnmarshall) {
					if err := consumer.CommitMessage(ctx, kfkMsg); err != nil {
						logger.Log.Error("failed to commit message:", err)
					}
					continue
				}
				logger.Log.Error("failed to fetch message:", err)
				break
			}

			logger.Log.Info("consumed event: %v", msg)
			switch msg.Name {
			case create_order.NotifyCommand:
				if err = deduplicator.WithDeduplicate(ctx, fmt.Sprintf("%d:%s", msg.ID, msg.Name), func(ctx context.Context) error {
					return postgresClient.BeginTxFunc(ctx, func(ctx context.Context) error {
						if err = notificationsRepository.CreateNotification(ctx, notification.NewNotification(msg.Order.UserID, "success to process order", time.Now())); err != nil {
							return err
						}
						if err = notificationsRepository.CreateEvent(ctx, create_order.NotifySucceededEvent, &msg.Order); err != nil {
							return err
						}
						return nil
					})
				}); err != nil {
					logger.Log.Error("failed to create notification:", err)
					continue readMessagesLoop
				}
			default:
				logger.Log.Debug("unsupported event type: %v", msg.Name)
			}

			if err := consumer.CommitMessage(ctx, kfkMsg); err != nil {
				logger.Log.Error("error committing messages: %v", err)
			}
		}

		log.Fatalf("restart because of kafka error")
	}()
}
