package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"dashboard/gateway/auth"
	"dashboard/gateway/internal/proxy"
	"dashboard/gateway/mailer"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var client = &http.Client{Timeout: 10 * time.Second}
var errMetadataUnavailable = errors.New("service metadata unavailable")

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
	githubService := proxy.NewGitHubServiceProxy(getenv("GITHUB_URL", "http://github:4000"))

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

		services := []any{weather}
		if githubMetadata, githubErr := fetchServiceMetadata(c, githubService); githubErr == nil {
			services = append(services, githubMetadata)
		}
		c.JSON(http.StatusOK, gin.H{
			"client": gin.H{"host": c.ClientIP()},
			"server": gin.H{
				"current_time": time.Now().Unix(),
				"services":     services,
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

	proxy.RegisterGitHubRoutes(api, authService.RequireAuth(), githubService)

	r.Run(":" + getenv("PORT", "8080"))
}

func fetchServiceMetadata(c *gin.Context, service *proxy.GitHubServiceProxy) (any, error) {
	request, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, service.BaseURL()+"/about.json", nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errMetadataUnavailable
	}
	var metadata any
	if err := json.NewDecoder(response.Body).Decode(&metadata); err != nil {
		return nil, err
	}
	return metadata, nil
}
