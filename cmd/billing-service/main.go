package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hihoak/otus-microservices-architect/cmd/billing-service/domain/account"
	"github.com/hihoak/otus-microservices-architect/cmd/billing-service/domain/account/repo"
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

var accountRepository *repo.PostgresAccountsRepository
var kafkaBilling *kafka2.ClientBillingEvents

type CreateAccountBody struct {
	UserID int64 `json:"user_id"`
}

type TopUpBody struct {
	Amount int64 `json:"amount"`
}

type WithDrawBody struct {
	Amount int64 `json:"amount"`
}

func main() {
	ctx := context.Background()

	appRouter := gin.Default()

	postgresClient, err := postgres.NewPostgresRepository(ctx, config.Cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("init postgres client: %v", err)
	}

	accountRepository = repo.NewPostgresAccountsRepository(postgresClient)
	kafkaBilling = kafka2.NewKafkaBillingEvents()

	appRouter.POST("/accounts", createAccountGin)

	appRouter.GET("/accounts", func(c *gin.Context) {
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

		acc, err := accountRepository.GetAccountByUserID(ctx, int64(userID))
		if err != nil {
			if errors.Is(err, account.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, acc)
	})

	appRouter.PUT("/accounts/:id/top-up", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("wrong id format it must be int: %s", err.Error())})
			return
		}

		body := TopUpBody{}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		acc, err := accountRepository.GetAccountByID(ctx, int64(id))
		if err != nil {
			if errors.Is(err, account.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		acc.TopUp(body.Amount)

		if err := accountRepository.UpdateAccount(ctx, acc); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{})
	})

	appRouter.PUT("/accounts/:id/withdraw", withdrawMoneyGin)

	listenUsersEvents(ctx)
	listenOrdersEvents(ctx)

	logger.Log.Info("starting service on address 0.0.0.0:9000...")
	appRouter.Run(fmt.Sprintf(":%s", "9000"))
}

func withdrawMoney(ctx context.Context, id int64, amount int64) error {
	acc, err := accountRepository.GetAccountByID(ctx, id)
	if err != nil {
		return err
	}

	if err := acc.Withdraw(amount); err != nil {
		return err
	}

	if err := accountRepository.UpdateAccount(ctx, acc); err != nil {
		return err
	}

	return nil
}

func withdrawMoneyGin(c *gin.Context) {
	ctx := context.Background()
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("wrong id format it must be int: %s", err.Error())})
		return
	}

	body := TopUpBody{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := withdrawMoney(ctx, int64(id), body.Amount); err != nil {
		if errors.Is(err, account.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, account.ErrUnsufficientFunds) {
			c.JSON(http.StatusNotAcceptable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func createAccount(ctx context.Context, userID int64) (account.AccountID, error) {
	acc, err := accountRepository.GetAccountByUserID(ctx, userID)
	if acc != nil {
		return acc.ID, nil
	}
	if err != nil && !errors.Is(err, account.ErrNotFound) {
		return 0, fmt.Errorf("get account by user ID: %w", err)
	}

	if err := accountRepository.CreateAccount(ctx, account.NewAccount(userID)); err != nil {
		return 0, fmt.Errorf("create account: %w", err)
	}

	return 0, nil
}

func createAccountGin(c *gin.Context) {
	ctx := context.Background()
	body := CreateAccountBody{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	accountID, err := createAccount(ctx, body.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": accountID})
}

func listenUsersEvents(ctx context.Context) {
	// make a new reader that consumes from topic-A
	usersEventsReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:               []string{"kafka.kafka.svc.cluster.local:9092"},
		GroupID:               "billing-service-users",
		Topic:                 "users-events",
		MaxBytes:              10e6, // 10MB,
		JoinGroupBackoff:      time.Millisecond * 500,
		HeartbeatInterval:     time.Millisecond * 200,
		WatchPartitionChanges: true,
	})

	go func() {
		defer func() {
			if err := usersEventsReader.Close(); err != nil {
				logger.Log.Error("failed to close reader:", err)
			}
		}()

		for {
			m, err := usersEventsReader.FetchMessage(context.Background())
			if err != nil {
				logger.Log.Error("failed to fetch message:", err)
				break
			}
			var unmarshalledEvent kafka2.UserEvent
			err = json.Unmarshal(m.Value, &unmarshalledEvent)
			if err != nil {
				logger.Log.Error("error unmarshalling event: %v", err)
				if err := usersEventsReader.CommitMessages(ctx, m); err != nil {
					logger.Log.Error("error committing messages: %v", err)
				}
				continue
			}

			logger.Log.Info("consumed event: %v", unmarshalledEvent)
			switch unmarshalledEvent.EventType {
			case kafka2.UserCreatedEvent:
				_, err := createAccount(ctx, int64(unmarshalledEvent.User.ID))
				if err != nil {
					logger.Log.Error("error creating account: %v", err)
					continue
				}
				if err := usersEventsReader.CommitMessages(ctx, m); err != nil {
					logger.Log.Error("error committing messages: %v", err)
				}
				continue
			default:
				log.Printf("unknown user event type: %v", unmarshalledEvent.EventType)
			}
			if err := usersEventsReader.CommitMessages(ctx, m); err != nil {
				logger.Log.Error("error committing messages: %v", err)
			}
		}

		log.Fatalf("restart because of kafka error")
	}()
}

func listenOrdersEvents(ctx context.Context) {
	// make a new reader that consumes from topic-A
	ordersEventsReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:               []string{"kafka.kafka.svc.cluster.local:9092"},
		GroupID:               "billing-service-orders",
		Topic:                 "orders-events",
		MaxWait:               time.Millisecond * 200,
		JoinGroupBackoff:      time.Millisecond * 500,
		HeartbeatInterval:     time.Millisecond * 200,
		WatchPartitionChanges: true,
	})

	go func() {
		defer func() {
			if err := ordersEventsReader.Close(); err != nil {
				logger.Log.Error("failed to close reader:", err)
			}
		}()

		for {
			m, err := ordersEventsReader.FetchMessage(context.Background())
			if err != nil {
				logger.Log.Error("failed to fetch message:", err)
				break
			}
			var unmarshalledEvent kafka2.OrderEvent
			err = json.Unmarshal(m.Value, &unmarshalledEvent)
			if err != nil {
				logger.Log.Error("error unmarshalling event: %v", err)
				if err := ordersEventsReader.CommitMessages(ctx, m); err != nil {
					logger.Log.Error("error committing messages: %v", err)
				}
				continue
			}

			logger.Log.Info("consumed event: %v", unmarshalledEvent)
			switch unmarshalledEvent.EventType {
			case kafka2.OrderCreatedEvent:
				acc, err := accountRepository.GetAccountByUserID(ctx, unmarshalledEvent.Order.UserID)
				if err != nil {
					if errors.Is(err, account.ErrNotFound) {
						if err := ordersEventsReader.CommitMessages(ctx, m); err != nil {
							logger.Log.Error("error committing messages: %v", err)
						}
						continue
					}
					logger.Log.Error("error get account by user ID: %v", err)
					continue
				}

				if err := withdrawMoney(ctx, int64(acc.ID), unmarshalledEvent.Order.Price); err != nil {
					if errors.Is(err, account.ErrNotFound) {
						if err := ordersEventsReader.CommitMessages(ctx, m); err != nil {
							logger.Log.Error("error committing messages: %v", err)
						}
						continue
					}
					if errors.Is(err, account.ErrUnsufficientFunds) {
						if err := kafkaBilling.WriteBillingEvent(ctx, kafka2.MoneyWithdrawFailedEvent, unmarshalledEvent.Order.UserID); err != nil {
							logger.Log.Error("failed to write billing event: %v", err)
						}

						if err := ordersEventsReader.CommitMessages(ctx, m); err != nil {
							logger.Log.Error("error committing messages: %v", err)
						}
						continue
					}
					logger.Log.Error("error withdrawMoney: %v", err)
					continue
				}

				if err := kafkaBilling.WriteBillingEvent(ctx, kafka2.MoneyWithdrawSucceededEvent, unmarshalledEvent.Order.UserID); err != nil {
					logger.Log.Error("failed to write billing event: %v", err)
					continue
				}
				if err := ordersEventsReader.CommitMessages(ctx, m); err != nil {
					logger.Log.Error("error committing messages: %v", err)
				}
				continue
			default:
				log.Printf("unknown user event type: %v", unmarshalledEvent.EventType)
			}
			if err := ordersEventsReader.CommitMessages(ctx, m); err != nil {
				logger.Log.Error("error committing messages: %v", err)
			}
		}

		log.Fatalf("restart because of kafka error")
	}()
}
