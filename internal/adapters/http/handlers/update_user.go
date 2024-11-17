package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hihoak/otus-microservices-architect/internal/domain/user"
	"net/http"
	"strconv"
)

type UpdateUserBody struct {
	FirstName string `json:"first_name" binding:"required"`
	Surname   string `json:"sur_name" binding:"required"`
	Age       uint8  `json:"age" binding:"required"`
}

func (s Service) UpdateUserHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("wrong id format it must be int: %s", err.Error())})
	}

	body := UpdateUserBody{}
	if err := c.BindJSON(&body); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	err = s.usersService.UpdateUser(context.Background(), uint64(id), body.FirstName, body.Surname, body.Age)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}
