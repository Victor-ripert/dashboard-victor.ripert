package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"dashboard/gateway/auth"
	"dashboard/gateway/internal/github"
	"dashboard/gateway/mailer"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var client = &http.Client{Timeout: 10 * time.Second}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	weatherURL := getenv("WEATHER_URL", "http://localhost:4000")
	databaseURL := getenv("DATABASE_URL", "postgres://dashboard:dashboard@localhost:5432/dashboard?sslmode=disable")
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET must be set")
	}

	store, err := auth.NewPostgresStore(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("Database unavailable: %v", err)
	}
	emailSender, err := mailer.NewSMTPMailer(mailer.SMTPConfig{
		Host:     getenv("SMTP_HOST", "localhost"),
		Port:     getenv("SMTP_PORT", "1025"),
		From:     getenv("SMTP_FROM", "Dashboard <noreply@dashboard.local>"),
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Emails are sent through SMTP server %s", emailSender.Addr())

	authService := auth.New(store, emailSender, jwtSecret)
	githubClient := github.NewClient()
	githubActivityClient := github.NewCachedClient(githubClient, time.Minute)

	r := gin.Default()
	// Let the front-end (served from another origin) call the gateway
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AddAllowHeaders("Authorization")
	r.Use(cors.New(corsConfig))

	// Routes used by the front-end live under /api
	api := r.Group("/api")
	authService.Routes(api)

	// Aggregated metadata for the front-end, built from each service's /about.json
	r.GET("/about.json", func(c *gin.Context) {
		resp, err := client.Get(weatherURL + "/about.json")
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Weather service unavailable"})
			return
		}
		defer resp.Body.Close()

		var weather any
		if err := json.NewDecoder(resp.Body).Decode(&weather); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Weather service unavailable"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"client": gin.H{"host": c.ClientIP()},
			"server": gin.H{
				"current_time": time.Now().Unix(),
				"services":     []any{weather},
			},
		})
	})

	// GET /api/weather/widgets/current_weather?city=Paris -> weather service (logged-in users only)
	api.GET("/weather/widgets/:widget", authService.RequireAuth(), func(c *gin.Context) {
		target := weatherURL + "/widgets/" + url.PathEscape(c.Param("widget")) + "?" + c.Request.URL.RawQuery
		resp, err := client.Get(target)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Weather service unavailable"})
			return
		}
		defer resp.Body.Close()

		c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
	})

	registerGithubRoutes(api, authService.RequireAuth(), githubActivityClient)

	r.Run(":" + getenv("PORT", "8080"))
}
