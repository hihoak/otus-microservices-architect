package service

import (
	"context"
	"fmt"
	"github.com/hihoak/otus-microservices-architect/internal/domain/user"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/logger"
)

type UserRepository interface {
	GetUser(ctx context.Context, id user.UserID) (*user.User, error)
	CreateUser(ctx context.Context, user user.User) error
	UpdateUser(ctx context.Context, user *user.User) error
	DeleteUser(ctx context.Context, id user.UserID) error
	ListUser(ctx context.Context) ([]user.User, error)
}

type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetUser(ctx context.Context, id uint64) (*user.User, error) {
	logger.Log.Info(fmt.Sprintf("[GetUser] id: %v", id))
	u, err := s.repo.GetUser(ctx, user.UserID(id))
	if err != nil {
		return nil, fmt.Errorf("GetUser: %w", err)
	}
	return u, nil
}

func (s *UserService) CreateUser(ctx context.Context, firstName, surName string, age uint8) error {
	logger.Log.Info(fmt.Sprintf("[CreateUser] firstName: %v, surName: %v. age: %v", firstName, surName, age))
	err := s.repo.CreateUser(ctx, user.NewUser(firstName, surName, age))
	if err != nil {
		return fmt.Errorf("CreateUser: %w", err)
	}
	return nil
}

func (s *UserService) UpdateUser(ctx context.Context, id uint64, firstName string, surName string, age uint8) error {
	logger.Log.Info(fmt.Sprintf("[UpdateUser] id: %v with firstName: %v, surName: %v, age: %v", id, firstName, surName, age))
	u, err := s.repo.GetUser(ctx, user.UserID(id))
	if err != nil {
		return fmt.Errorf("GetUser: %w", err)
	}
	u.SetFirstName(firstName)
	u.SetSurname(surName)
	u.SetAge(age)

	if err = s.repo.UpdateUser(ctx, u); err != nil {
		return fmt.Errorf("UpdateUser: %w", err)
	}
	return nil
}

func (s *UserService) DeleteUser(ctx context.Context, id uint64) error {
	logger.Log.Info(fmt.Sprintf("[DeleteUser] id: %v", id))
	if err := s.repo.DeleteUser(ctx, user.UserID(id)); err != nil {
		return fmt.Errorf("DeleteUser: %w", err)
	}
	return nil
}

func (s *UserService) ListUser(ctx context.Context) ([]user.User, error) {
	logger.Log.Info(fmt.Sprintf("[ListUser] ListUser"))
	return s.repo.ListUser(ctx)
}