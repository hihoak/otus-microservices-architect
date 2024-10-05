package handlers

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (s Service) ListUsersHandler(c *gin.Context) {
	u, err := s.usersService.ListUser(context.Background())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	c.JSON(http.StatusOK, gin.H{"users": u})
}
