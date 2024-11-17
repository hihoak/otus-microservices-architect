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
)

var accountRepository *repo.PostgresAccountsRepository

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

	appRouter.PUT("/accounts/:id/withdraw", func(c *gin.Context) {
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

		if err := acc.Withdraw(body.Amount); err != nil {
			if errors.Is(err, account.ErrUnsufficientFunds) {
				c.JSON(http.StatusNotAcceptable, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if err := accountRepository.UpdateAccount(ctx, acc); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{})
	})

	// make a new reader that consumes from topic-A
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"kafka.kafka.svc.cluster.local:9092"},
		GroupID:  "billing-service",
		Topic:    "users-events",
		MaxBytes: 10e6, // 10MB
	})

	go func() {
		defer func() {
			if err := r.Close(); err != nil {
				logger.Log.Error("failed to close reader:", err)
			}
		}()

		for {
			m, err := r.FetchMessage(context.Background())
			if err != nil {
				logger.Log.Error("failed to fetch message:", err)
				break
			}
			var unmarshalledEvent kafka2.UserEvent
			err = json.Unmarshal(m.Value, &unmarshalledEvent)
			if err != nil {
				logger.Log.Error("error unmarshalling event: %v", err)
				if err := r.CommitMessages(ctx, m); err != nil {
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
				if err := r.CommitMessages(ctx, m); err != nil {
					logger.Log.Error("error committing messages: %v", err)
				}
				continue
			default:
				log.Printf("unknown user event type: %v", unmarshalledEvent.EventType)
			}
			if err := r.CommitMessages(ctx, m); err != nil {
				logger.Log.Error("error committing messages: %v", err)
			}
		}

		log.Fatalf("restart because of kafka error")
	}()

	logger.Log.Info("starting service on address 0.0.0.0:9000...")
	appRouter.Run(fmt.Sprintf(":%s", "9000"))
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
