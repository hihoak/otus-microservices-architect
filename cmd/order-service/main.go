package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hihoak/otus-microservices-architect/cmd/order-service/domain/order"
	repo2 "github.com/hihoak/otus-microservices-architect/cmd/order-service/domain/order/repo"
	"github.com/hihoak/otus-microservices-architect/cmd/order-service/sagas"
	"github.com/hihoak/otus-microservices-architect/cmd/order-service/sagas/create_order"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/kafka"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/repository/postgres"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/config"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/logger"
	"log"
	"net/http"
	"strconv"
)

type CreateOrderBody struct {
	UserID            int64           `json:"user_id"`
	Price             int64           `json:"price"`
	ItemIDsWithStocks map[int64]int64 `json:"item_ids_with_stocks"`
	DeliverySlotID    int64           `json:"delivery_slot_id"`
}

func withIdempotency(c *gin.Context) (uuid.UUID, bool) {
	requestIDStr := c.GetHeader("X-Request-Id")
	if requestIDStr == "" {
		return uuid.UUID{}, false
	}
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		logger.Log.Warn("passed not valid UUID into x-request-id", requestIDStr)
		return uuid.UUID{}, false
	}

	return requestID, true
}

func main() {
	ctx := context.Background()

	appRouter := gin.Default()

	postgresClient, err := postgres.NewPostgresRepository(ctx, config.Cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("init postgres client: %v", err)
	}

	ordersRepository := repo2.NewPostgresOrdersRepository(postgresClient)

	createOrderSagaProducer := kafka.NewKafkaCreateOrderSagaProducer()

	orchestrator := sagas.NewOrchestrator(ctx, ordersRepository, createOrderSagaProducer)
	orchestrator.Start(ctx)
	defer orchestrator.Stop()

	appRouter.POST("/orders", func(c *gin.Context) {
		body := CreateOrderBody{}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		requestID, withIdempotence := withIdempotency(c)

		if withIdempotence {
			logger.Log.Info("try to get early created order by request id", requestID.String())
			ord, err := ordersRepository.GetCreateOrderResponse(ctx, requestID)
			if err == nil {
				c.JSON(http.StatusOK, ord)
				return
			}
			if errors.Is(err, repo2.ErrCreateOrderResponseNotFound) {
				logger.Log.Info("that's is the first run for request ID:", requestID.String())
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		var ord *order.Order
		if err = postgresClient.BeginTxFunc(ctx, func(ctx context.Context) error {
			var innerErr error
			ord, innerErr = ordersRepository.CreateOrder(ctx, order.NewOrder(body.UserID, body.Price, body.ItemIDsWithStocks, body.DeliverySlotID))
			if innerErr != nil {
				return fmt.Errorf("create order: %w", innerErr)
			}
			if !withIdempotence {
				return nil
			}
			logger.Log.Info("save response for request ID", requestID.String())
			innerErr = ordersRepository.CreateCreateOrderResponse(ctx, requestID, ord)
			if innerErr != nil {
				return fmt.Errorf("create create order response: %w", innerErr)
			}
			return nil
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		command, err := create_order.InitCreateOrderSaga(ord.Status).GetNextCommand()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if err = createOrderSagaProducer.WriteOrderEvent(ctx, command, ord); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, ord)
	})

	appRouter.GET("/orders/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("wrong id format it must be int: %s", err.Error())})
			return
		}

		ord, err := ordersRepository.GetOrderByID(ctx, order.OrderID(id))
		if err != nil {
			if errors.Is(err, order.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, ord)
	})

	logger.Log.Info("starting service on address 0.0.0.0:9000...")
	appRouter.Run(fmt.Sprintf(":%s", "9000"))
}
