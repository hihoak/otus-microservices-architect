package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/logger"
	"golang.org/x/oauth2"
	"log"
	"net"
	"net/http"
	"time"
)

const (
	clientID       = "auth-service"
	accessClientID = "account"

	archHomework80 = "arch.homework:80"
	keycloak80     = "keycloak.keycloak.svc.cluster.local:8888"

	keycloakURL = "http://keycloak.keycloak.svc.cluster.local:8888/keycloak/realms/master"

	externalKeycloakURL                 = "http://arch.homework/keycloak/realms/master"
	externalLoginAuthServiceRedirectURL = "http://arch.homework/auth/artem/login"
)

const (
	accessTokenCookieName = "Authorization"
)

var (
	idTokenVerifier     *oidc.IDTokenVerifier
	accessTokenVerifier *oidc.IDTokenVerifier
)

func main() {
	ctx := context.Background()

	ctx = oidc.InsecureIssuerURLContext(ctx, externalKeycloakURL)

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	// or create your own transport, there's an example on godoc.
	http.DefaultTransport.(*http.Transport).DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		logger.Log.Info("send request to ", addr)
		if addr == archHomework80 {
			addr = keycloak80
		}
		return dialer.DialContext(ctx, network, addr)
	}

	oidcProvider, err := oidc.NewProvider(ctx, keycloakURL)
	if err != nil {
		log.Fatalf("could not create oidc provider: %v", err)
	}

	oauth2Config := oauth2.Config{
		ClientID:    clientID,
		RedirectURL: externalLoginAuthServiceRedirectURL,
		Endpoint:    oidcProvider.Endpoint(),
		Scopes:      []string{oidc.ScopeOpenID, "profile", "email"},
	}

	idTokenVerifier = oidcProvider.Verifier(&oidc.Config{ClientID: clientID})
	accessTokenVerifier = oidcProvider.Verifier(&oidc.Config{ClientID: accessClientID})

	appRouter := gin.Default()

	appRouter.GET("/sign-up", func(c *gin.Context) {
		reason, err := checkTokensFromCookie(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "check cookie failed" + err.Error()})
			return
		}
		if reason == "" {
			c.JSON(http.StatusOK, gin.H{"status": "already signed up"})
			return
		}

		http.Redirect(c.Writer, c.Request, oauth2Config.AuthCodeURL("some-state"), http.StatusFound)
	})

	appRouter.GET("/verify", func(c *gin.Context) {
		status, err := checkTokensFromCookie(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "check cookie failed" + err.Error()})
			return
		}
		if status != "" {
			c.JSON(http.StatusUnauthorized, gin.H{"status": "you need to pass Authorization cookie " + status})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "already signed up"})
		return
	})

	appRouter.GET("/login/basic-auth", func(c *gin.Context) {
		username, password, ok := c.Request.BasicAuth()
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"status": "you need to pass basic auth username, password"})
			return
		}
		token, err := oauth2Config.PasswordCredentialsToken(ctx, username, password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"status": "wrong username or password " + err.Error()})
			return
		}
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("Authorization", token.AccessToken, int((time.Minute * 30).Seconds()), "", "", false, true)

		c.JSON(http.StatusOK, gin.H{})
	})

	appRouter.GET("/login", func(c *gin.Context) {
		status, err := checkTokensFromCookie(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "check cookie failed" + err.Error()})
			return
		}
		if status == "" {
			c.JSON(http.StatusOK, gin.H{"status": "already signed up"})
			return
		}

		oauth2Token, err := oauth2Config.Exchange(ctx, c.Query("code"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no id_token field in oauth2 token"})
			return
		}

		// Parse and verify ID Token payload.
		_, err = idTokenVerifier.Verify(ctx, rawIDToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "can't verify id_token: " + err.Error()})
			return
		}

		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("Authorization", oauth2Token.AccessToken, int((time.Minute * 30).Seconds()), "", "", false, true)

		c.JSON(http.StatusOK, gin.H{})
	})

	logger.Log.Info("starting service on address 0.0.0.0:9000...")
	appRouter.Run(fmt.Sprintf(":%s", "9000"))
}

func checkTokensFromCookie(c *gin.Context) (string, error) {
	logger.Log.Info("check tokens from cookie")
	accessToken, err := c.Cookie(accessTokenCookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return "not found token from cookie", nil
		} else {
			return "", fmt.Errorf("get access token from cookie: %w", err)
		}
	}

	_, err = accessTokenVerifier.Verify(context.Background(), accessToken)
	if err != nil {
		var expiredErr *oidc.TokenExpiredError
		if errors.As(err, &expiredErr) {
			logger.Log.Info("token expired")
			return fmt.Sprintf("token expired: %s", expiredErr.Expiry), nil
		} else {
			return "", fmt.Errorf("verify access token: %w", err)
		}
	}
	logger.Log.Info("token found")
	return "", nil
}
