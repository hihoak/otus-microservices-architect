package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hihoak/otus-microservices-architect/cmd/notification-service/domain/notification"
	"github.com/hihoak/otus-microservices-architect/cmd/notification-service/domain/notification/repo"
	kafka2 "github.com/hihoak/otus-microservices-architect/internal/adapters/kafka"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/repository/postgres"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/config"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/logger"
	"github.com/segmentio/kafka-go"
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

func main() {
	ctx := context.Background()

	appRouter := gin.Default()

	postgresClient, err := postgres.NewPostgresRepository(ctx, config.Cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("init postgres client: %v", err)
	}

	notificationsRepository = repo.NewPostgresNotificationsRepository(postgresClient)

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

	listenBillingEvents(ctx)

	logger.Log.Info("starting service on address 0.0.0.0:9000...")
	appRouter.Run(fmt.Sprintf(":%s", "9000"))
}

func listenBillingEvents(ctx context.Context) {
	// make a new reader that consumes from topic-A
	billingEventsReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"kafka.kafka.svc.cluster.local:9092"},
		GroupID:  "notification-service",
		Topic:    "billing-events",
		MaxBytes: 10e6, // 10MB
	})

	go func() {
		defer func() {
			if err := billingEventsReader.Close(); err != nil {
				logger.Log.Error("failed to close reader:", err)
			}
		}()

		for {
			m, err := billingEventsReader.FetchMessage(context.Background())
			if err != nil {
				logger.Log.Error("failed to fetch message:", err)
				break
			}
			var unmarshalledEvent kafka2.BillingEvent
			err = json.Unmarshal(m.Value, &unmarshalledEvent)
			if err != nil {
				logger.Log.Error("error unmarshalling event: %v", err)
				if err := billingEventsReader.CommitMessages(ctx, m); err != nil {
					logger.Log.Error("error committing messages: %v", err)
				}
				continue
			}

			logger.Log.Info("consumed event: %v", unmarshalledEvent)
			switch unmarshalledEvent.EventType {
			case kafka2.MoneyWithdrawFailedEvent:
				err := notificationsRepository.CreateNotification(ctx, notification.NewNotification(unmarshalledEvent.UserID, "failed to withdraw money", time.Now()))
				if err != nil {
					logger.Log.Error("failed to create notification: %v", err)
					continue
				}
			case kafka2.MoneyWithdrawSucceededEvent:
				err := notificationsRepository.CreateNotification(ctx, notification.NewNotification(unmarshalledEvent.UserID, "success to withdraw money", time.Now()))
				if err != nil {
					logger.Log.Error("failed to create notification: %v", err)
					continue
				}
			default:
				log.Printf("unknown billing event type: %v", unmarshalledEvent.EventType)
			}
			if err := billingEventsReader.CommitMessages(ctx, m); err != nil {
				logger.Log.Error("error committing messages: %v", err)
			}
		}

		log.Fatalf("restart because of kafka error")
	}()
}
