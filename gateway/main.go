package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"time"

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

	r := gin.Default()
	// Let the front-end (served from another origin) call the gateway
	r.Use(cors.Default())

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

	// GET /weather/widgets/current_weather?city=Paris -> weather service
	r.GET("/weather/widgets/:widget", func(c *gin.Context) {
		target := weatherURL + "/widgets/" + url.PathEscape(c.Param("widget")) + "?" + c.Request.URL.RawQuery
		resp, err := client.Get(target)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Weather service unavailable"})
			return
		}
		defer resp.Body.Close()

		c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
	})

	r.Run(":" + getenv("PORT", "8080"))
}
