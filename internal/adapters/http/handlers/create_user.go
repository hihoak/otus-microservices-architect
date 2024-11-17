package handlers

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/hihoak/otus-microservices-architect/internal/domain/user"
	"net/http"
)

type CreateUserBody struct {
	FirstName string `json:"first_name" binding:"required"`
	Surname   string `json:"sur_name" binding:"required"`
	Age       uint8  `json:"age" binding:"required"`
}

func (s Service) CreateUserHandler(c *gin.Context) {
	body := CreateUserBody{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	username, _, _ := c.Request.BasicAuth()
	usr, err := s.usersService.CreateUser(context.Background(), body.FirstName, body.Surname, body.Age, username)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err = s.kafkaClient.WriteUserCreatedEvent(context.Background(), usr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}
