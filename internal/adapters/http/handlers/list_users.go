package handlers

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (s Service) ListUsersHandler(c *gin.Context) {
	username, _, _ := c.Request.BasicAuth()
	u, err := s.usersService.ListUser(context.Background(), username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": u})
}
