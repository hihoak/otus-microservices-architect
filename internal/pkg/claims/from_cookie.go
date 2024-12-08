package claims

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	authorizationCookieName = "Authorization"
)

type UserClaims struct {
	jwt.RegisteredClaims
	PreferredUsername string `json:"preferred_username"`
}

func UsernameFromGinContext(c *gin.Context) (string, error) {
	authCookie, err := c.Cookie(authorizationCookieName)
	if err != nil {
		return "", fmt.Errorf("failed to get authorization cookie: %w", err)
	}
	jwtToken, _ := jwt.ParseWithClaims(authCookie, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return nil, nil
	})
	if jwtToken == nil {
		return "", fmt.Errorf("failed to parse JWT token")
	}
	claims, ok := jwtToken.Claims.(*UserClaims)
	if !ok {
		return "", fmt.Errorf("failed to parse claims")
	}
	return claims.PreferredUsername, nil
}
