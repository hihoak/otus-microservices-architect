package handlers

import (
	"github.com/hihoak/otus-microservices-architect/internal/service"
)

type Service struct {
	usersService *service.UserService
}

func NewService(usersService *service.UserService) *Service {
	return &Service{
		usersService: usersService,
	}
}
