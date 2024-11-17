package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/http/handlers"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/kafka"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/repository/postgres"
	"github.com/hihoak/otus-microservices-architect/internal/domain/user/repo"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/config"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/logger"
	_ "github.com/hihoak/otus-microservices-architect/internal/pkg/logger"
	"github.com/hihoak/otus-microservices-architect/internal/service"
	"github.com/penglongli/gin-metrics/ginmetrics"
	"log"
)

func main() {
	ctx := context.Background()
	postgresClient, err := postgres.NewPostgresRepository(ctx, config.Cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("init postgres client: %v", err)
	}
	usersRepo := repo.NewPostgresUsersRepository(postgresClient)
	userService := service.NewUserService(usersRepo)
	kafkaClient := kafka.NewKafka()

	app := handlers.NewService(userService, kafkaClient)

	appRouter := gin.Default()
	metricsRouter := gin.Default()

	m := ginmetrics.GetMonitor()
	m.SetMetricPath("/metrics")
	m.SetSlowTime(10)
	m.SetDuration([]float64{0.01, 0.05, 0.1, 0.2, 0.5})
	m.UseWithoutExposingEndpoint(appRouter)
	m.Expose(metricsRouter)
	go func() {
		if err := metricsRouter.Run(":8001"); err != nil {
			log.Fatalf("metrics router start: %v", err)
		}
	}()

	appRouter.GET("/health", app.HealthHandler)
	appRouter.POST("/users", app.CreateUserHandler)
	appRouter.PUT("/users/:id", app.UpdateUserHandler)
	appRouter.GET("/users/:id", app.GetUserHandler)
	appRouter.DELETE("/users/:id", app.DeleteUserHandler)
	appRouter.GET("/users", app.ListUsersHandler)

	logger.Log.Info("starting service on address 0.0.0.0:8000...")
	appRouter.Run(fmt.Sprintf(":%s", config.Cfg.Port))
}
