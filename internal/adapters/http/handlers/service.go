package handlers

import (
	"github.com/hihoak/otus-microservices-architect/internal/adapters/kafka"
	"github.com/hihoak/otus-microservices-architect/internal/service"
)

type Service struct {
	usersService *service.UserService
	kafkaClient  *kafka.ClientUsersEvents
}

func NewService(usersService *service.UserService, kafkaClient *kafka.ClientUsersEvents) *Service {
	return &Service{
		usersService: usersService,
		kafkaClient:  kafkaClient,
	}
}
