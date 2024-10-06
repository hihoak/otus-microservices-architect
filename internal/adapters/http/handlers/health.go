package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (s Service) HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "OK",
	})
}
