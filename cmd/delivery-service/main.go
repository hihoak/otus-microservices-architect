package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	delivery_slots "github.com/hihoak/otus-microservices-architect/cmd/delivery-service/domain/delivery-slots"
	"github.com/hihoak/otus-microservices-architect/cmd/delivery-service/domain/delivery-slots/repo"
	"github.com/hihoak/otus-microservices-architect/cmd/order-service/sagas/create_order"
	"github.com/hihoak/otus-microservices-architect/cmd/warehouse-service/domain/items"
	kafka2 "github.com/hihoak/otus-microservices-architect/internal/adapters/kafka"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/repository/postgres"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/config"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/logger"
	"github.com/hihoak/otus-microservices-architect/pkg/kafka_deduplicator"
	"log"
	"net/http"
	"strconv"
	"time"
)

var deliverySlotsRepository *repo.PostgresDeliverySlotsRepository
var createOrderSagaProducer *kafka2.ClientCreateOrderSagaCommand
var postgresClient *postgres.PostgresRepository

type CreateDeliverySlotBody struct {
	FromTime       string `json:"from_time"`
	ToTime         string `json:"to_time"`
	DeliveriesLeft int64  `json:"deliveries_left"`
}

func main() {
	ctx := context.Background()

	appRouter := gin.Default()
	var err error
	postgresClient, err = postgres.NewPostgresRepository(ctx, config.Cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("init postgres client: %v", err)
	}

	deliverySlotsRepository = repo.NewPostgresDeliverySlotsRepository(postgresClient)
	createOrderSagaProducer = kafka2.NewKafkaCreateOrderSagaProducer()

	appRouter.POST("/delivery-slots", func(c *gin.Context) {
		body := CreateDeliverySlotBody{}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		fromTime, err := time.Parse(time.RFC3339, body.FromTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		toTime, err := time.Parse(time.RFC3339, body.FromTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		deliverySlot, err := deliverySlotsRepository.CreateDeliverySlot(ctx, delivery_slots.NewDeliverySlot(fromTime, toTime, body.DeliveriesLeft))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, deliverySlot)
	})

	appRouter.GET("/delivery-slots/:id", func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("wrong id format it must be int: %s", err.Error())})
			return
		}

		deliverySlot, err := deliverySlotsRepository.GetDeliverySlotByID(ctx, id)
		if err != nil {
			if errors.Is(err, items.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, deliverySlot)
	})

	listenCreateOrderSagaEvents(ctx)

	logger.Log.Info("starting service on address 0.0.0.0:9000...")
	appRouter.Run(fmt.Sprintf(":%s", "9000"))
}

func reserveDeliverySlot(ctx context.Context, id uint64) error {
	deliverySlot, err := deliverySlotsRepository.GetDeliverySlotByID(ctx, id)
	if err != nil {
		return err
	}
	if err := deliverySlot.Reserve(); err != nil {
		return err
	}
	if err := deliverySlotsRepository.UpdateDeliverySlot(ctx, deliverySlot); err != nil {
		return err
	}
	return nil
}

func listenCreateOrderSagaEvents(ctx context.Context) {
	// make a new reader that consumes from topic-A
	consumer := kafka2.NewKafkaCreateOrderSagaConsumer("delivery-slots-service")
	deduplicator := kafka_deduplicator.NewKafkaDeduplicator(postgresClient, "processed_create_order_saga_commands_delivery_service")

	go func() {
		defer func() {
			if err := consumer.Close(); err != nil {
				logger.Log.Error("failed to close reader:", err)
			}
		}()

	readMessagesLoop:
		for {
			msg, kfkMsg, err := consumer.FetchMessage(context.Background())
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
			case create_order.ReserveSlotCommand:
				var sendFail bool
				if err = deduplicator.WithDeduplicate(ctx, fmt.Sprintf("%d:%s", msg.ID, msg.Name), func(ctx context.Context) error {
					return postgresClient.BeginTxFunc(ctx, func(ctx context.Context) error {
						if err = reserveDeliverySlot(ctx, uint64(msg.Order.DeliverySlotID)); err != nil {
							if !(errors.Is(err, delivery_slots.ErrNotFound) || errors.Is(err, delivery_slots.ErrNotEnoughDeliveries)) {
								return fmt.Errorf("failed to reserve slot: %w", err)
							}
							sendFail = true
							logger.Log.Info("it is not possible to reserve slot: ", err)
							return nil
						}
						return nil
					})
				}); err != nil {
					logger.Log.Error("failed to reserve slot:", err)
					continue readMessagesLoop
				}

				if sendFail {
					err := createOrderSagaProducer.WriteEvent(ctx, string(create_order.ReserveSlotFailedEvent), &msg.Order)
					if err != nil {
						logger.Log.Error("failed to write event:", err)
					}
				} else {
					err := createOrderSagaProducer.WriteEvent(ctx, string(create_order.ReserveSlotSucceededEvent), &msg.Order)
					if err != nil {
						logger.Log.Error("failed to write event:", err)
					}
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
