package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/logger"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"sync"
)

type UserInfo struct {
	Username       string
	HashedPassword []byte
}

type UsersRepo struct {
	repo map[string]*UserInfo
	mu   *sync.Mutex
}

func (u *UsersRepo) GetUserByUsername(ctx context.Context, username string) (*UserInfo, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	user, ok := u.repo[username]
	return user, ok
}

func (u *UsersRepo) SetUserInfo(ctx context.Context, user *UserInfo) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.repo[user.Username] = user
}

var (
	usersRepo = UsersRepo{
		repo: make(map[string]*UserInfo),
		mu:   &sync.Mutex{},
	}
)

type SignUpRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func main() {
	ctx := context.Background()

	appRouter := gin.Default()

	appRouter.POST("/sign-up", func(c *gin.Context) {
		req := SignUpRequest{}
		if err := c.BindJSON(&req); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("parse body: %s", err.Error())})
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 4)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("generate password hash: %s", err.Error())})
			return
		}

		usersRepo.SetUserInfo(context.Background(), &UserInfo{
			Username:       req.Username,
			HashedPassword: hash,
		})

		c.JSON(http.StatusOK, gin.H{})
	})

	appRouter.GET("/login", func(c *gin.Context) {
		user, password, ok := c.Request.BasicAuth()
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no username or password"})
			return
		}

		userInfo, ok := usersRepo.GetUserByUsername(ctx, user)
		if !ok {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("user not found: %s", user)})
			return
		}

		if err := bcrypt.CompareHashAndPassword(userInfo.HashedPassword, []byte(password)); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("wrong password for user %s", user)})
			return
		}

		c.JSON(http.StatusOK, gin.H{})
	})

	logger.Log.Info("starting service on address 0.0.0.0:9000...")
	appRouter.Run(fmt.Sprintf(":%s", "9000"))
}
