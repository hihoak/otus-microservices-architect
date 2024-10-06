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
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	err := s.usersService.CreateUser(context.Background(), body.FirstName, body.Surname, body.Age)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	c.JSON(http.StatusOK, gin.H{})
}
