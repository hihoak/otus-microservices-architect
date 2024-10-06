package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/http/handlers"
	"github.com/hihoak/otus-microservices-architect/internal/adapters/repository/postgres"
	"github.com/hihoak/otus-microservices-architect/internal/domain/user/repo"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/config"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/logger"
	_ "github.com/hihoak/otus-microservices-architect/internal/pkg/logger"
	"github.com/hihoak/otus-microservices-architect/internal/service"
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

	app := handlers.NewService(userService)

	r := gin.Default()
	r.GET("/health", app.HealthHandler)
	r.POST("/users", app.CreateUserHandler)
	r.PUT("/users/:id", app.UpdateUserHandler)
	r.GET("/users/:id", app.GetUserHandler)
	r.DELETE("/users/:id", app.DeleteUserHandler)
	r.GET("/users", app.ListUsersHandler)

	logger.Log.Info("starting service on address 0.0.0.0:8000...")
	r.Run(fmt.Sprintf(":%s", config.Cfg.Port))
}
