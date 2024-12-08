package handlers

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/claims"
	"net/http"
)

func (s Service) ListUsersHandler(c *gin.Context) {
	username, err := claims.UsernameFromGinContext(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	u, err := s.usersService.ListUser(context.Background(), username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": u})
}
