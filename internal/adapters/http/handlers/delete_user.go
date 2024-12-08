package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hihoak/otus-microservices-architect/internal/domain/user"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/claims"
	"net/http"
	"strconv"
)

func (s Service) DeleteUserHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("wrong id format it must be int: %s", err.Error())})
		return
	}
	username, err := claims.UsernameFromGinContext(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	err = s.usersService.DeleteUser(context.Background(), uint64(id), username)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}
