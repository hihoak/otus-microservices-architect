package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hihoak/otus-microservices-architect/cmd/notification-service/domain/notification"
	"github.com/hihoak/otus-microservices-architect/cmd/notification-service/domain/notification/repo"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/repository/postgres"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/config"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/logger"
	"log"
	"net/http"
	"strconv"
	"time"
)

type CreateNotificationBody struct {
	UserID int64  `json:"user_id"`
	Text   string `json:"text"`
}

func main() {
	ctx := context.Background()

	appRouter := gin.Default()

	postgresClient, err := postgres.NewPostgresRepository(ctx, config.Cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("init postgres client: %v", err)
	}

	notificationsRepository := repo.NewPostgresNotificationsRepository(postgresClient)

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

	logger.Log.Info("starting service on address 0.0.0.0:9000...")
	appRouter.Run(fmt.Sprintf(":%s", "9000"))
}
