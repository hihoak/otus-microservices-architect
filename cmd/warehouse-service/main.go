package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hihoak/otus-microservices-architect/cmd/order-service/sagas/create_order"
	"github.com/hihoak/otus-microservices-architect/cmd/warehouse-service/domain/items"
	"github.com/hihoak/otus-microservices-architect/cmd/warehouse-service/domain/items/repo"
	kafka2 "github.com/hihoak/otus-microservices-architect/internal/adapters/kafka"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/repository/postgres"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/config"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/logger"
	"github.com/hihoak/otus-microservices-architect/pkg/kafka_deduplicator"
	"log"
	"net/http"
	"strconv"
)

var postgresClient *postgres.PostgresRepository
var itemsRepository *repo.PostgresItemsRepository
var createOrderSagaProducer *kafka2.ClientCreateOrderSagaCommand

type CreateItemBody struct {
	Count uint64 `json:"count"`
}

type ReserveItemBody struct {
	Count uint64 `json:"count"`
}

func main() {
	ctx := context.Background()

	appRouter := gin.Default()

	var err error
	postgresClient, err = postgres.NewPostgresRepository(ctx, config.Cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("init postgres client: %v", err)
	}

	itemsRepository = repo.NewPostgresItemsRepository(postgresClient)
	createOrderSagaProducer = kafka2.NewKafkaCreateOrderSagaProducer()

	appRouter.POST("/items", func(c *gin.Context) {
		body := CreateItemBody{}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		item, err := itemsRepository.CreateItem(ctx, items.NewItem(body.Count))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, item)
	})

	appRouter.PUT("/items/:id/reserve", func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("wrong id format it must be int: %s", err.Error())})
			return
		}

		body := ReserveItemBody{}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if err := reserveStock(ctx, id, body.Count); err != nil {
			if errors.Is(err, items.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			if errors.Is(err, items.ErrNotEnoughItems) {
				c.JSON(http.StatusNotAcceptable, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{})
	})

	appRouter.GET("/items/:id", func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("wrong id format it must be int: %s", err.Error())})
			return
		}

		item, err := itemsRepository.GetItemByID(ctx, id)
		if err != nil {
			if errors.Is(err, items.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, item)
	})

	listenCreateOrderSagaEvents(ctx)

	logger.Log.Info("starting service on address 0.0.0.0:9000...")
	appRouter.Run(fmt.Sprintf(":%s", "9000"))
}

func reserveStock(ctx context.Context, id uint64, count uint64) error {
	item, err := itemsRepository.GetItemByID(ctx, id)
	if err != nil {
		return err
	}

	if err := item.Reserve(count); err != nil {
		return err
	}

	err = itemsRepository.UpdateItem(ctx, item)
	if err != nil {
		return err
	}
	return nil
}

func undoReserveStock(ctx context.Context, id uint64, count uint64) error {
	item, err := itemsRepository.GetItemByID(ctx, id)
	if err != nil {
		return err
	}

	item.Add(count)

	err = itemsRepository.UpdateItem(ctx, item)
	if err != nil {
		return err
	}
	return nil
}

func listenCreateOrderSagaEvents(ctx context.Context) {
	// make a new reader that consumes from topic-A
	consumer := kafka2.NewKafkaCreateOrderSagaConsumer("warehouse-service")
	deduplicator := kafka_deduplicator.NewKafkaDeduplicator(postgresClient, "processed_create_order_saga_commands_warehouse_service")

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
			case create_order.ReserveStockCommand:
				var sendFail bool
				if err = deduplicator.WithDeduplicate(ctx, fmt.Sprintf("%d:%s", msg.ID, msg.Name), func(ctx context.Context) error {
					return postgresClient.BeginTxFunc(ctx, func(ctx context.Context) error {
						var reserveErr error
						for itemID, count := range msg.Order.ItemIDsWithStocks {
							if reserveErr = reserveStock(ctx, uint64(itemID), uint64(count)); reserveErr != nil {
								if errors.Is(reserveErr, items.ErrNotFound) || errors.Is(reserveErr, items.ErrNotEnoughItems) {
									sendFail = true
									logger.Log.Info("unable to reserve stock: %w", reserveErr)
									break
								}
								return fmt.Errorf("failed to reserve stock: %w", reserveErr)
							}
						}
						return nil
					})
				}); err != nil {
					logger.Log.Error("failed to reserve stock:", err)
					continue readMessagesLoop
				}

				if sendFail {
					err := createOrderSagaProducer.WriteEvent(ctx, string(create_order.ReserveStockFailedEvent), &msg.Order)
					if err != nil {
						logger.Log.Error("failed to write event:", err)
					}
				} else {
					err := createOrderSagaProducer.WriteEvent(ctx, string(create_order.ReserveStockSucceededEvent), &msg.Order)
					if err != nil {
						logger.Log.Error("failed to write event:", err)
					}
				}
			case create_order.UndoReserveStockCommand:
				if err = deduplicator.WithDeduplicate(ctx, fmt.Sprintf("%d:%s", msg.ID, msg.Name), func(ctx context.Context) error {
					return postgresClient.BeginTxFunc(ctx, func(ctx context.Context) error {
						for itemID, count := range msg.Order.ItemIDsWithStocks {
							if undoReserveErr := undoReserveStock(ctx, uint64(itemID), uint64(count)); undoReserveErr != nil {
								return fmt.Errorf("failed to reserve stock: %w", undoReserveErr)
							}
						}
						return nil
					})
				}); err != nil {
					logger.Log.Error("failed to reserve stock:", err)
					continue readMessagesLoop
				}

				err := createOrderSagaProducer.WriteEvent(ctx, string(create_order.UndoReserveStockSucceededEvent), &msg.Order)
				if err != nil {
					logger.Log.Error("failed to write event:", err)
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
