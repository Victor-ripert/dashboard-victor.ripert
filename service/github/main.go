package main

import (
	"context"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET must be set")
	}
	var store *ConnectionStore
	var err error
	if rawKey := os.Getenv("GITHUB_TOKEN_ENCRYPTION_KEY"); rawKey != "" {
		key, decodeErr := hex.DecodeString(rawKey)
		if decodeErr != nil || len(key) != 32 {
			log.Fatal("GITHUB_TOKEN_ENCRYPTION_KEY must be 64 hexadecimal characters")
		}
		store, err = NewConnectionStore(context.Background(), env("DATABASE_URL", "postgres://dashboard:dashboard@localhost:5432/dashboard?sslmode=disable"), key)
		if err != nil {
			log.Fatalf("GitHub store unavailable: %v", err)
		}
	}
	service := NewGitHubService(GitHubConfig{
		ClientID: env("GITHUB_CLIENT_ID", ""), ClientSecret: env("GITHUB_CLIENT_SECRET", ""),
		RedirectURL: env("GITHUB_REDIRECT_URL", "http://localhost:8080/api/github/callback"),
		FrontendURL: env("GITHUB_FRONTEND_URL", "http://localhost:5173/"),
		JWTSecret:   jwtSecret,
	}, store)

	router := gin.Default()
	registerRoutes(router, service)
	server := &http.Server{Addr: ":" + env("PORT", "4000"), Handler: router, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("GitHub service listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
