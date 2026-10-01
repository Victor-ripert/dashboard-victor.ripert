package config

import (
	"os"
)

type Config struct {
	HTTPPort      string
	TradingAPIURL string
}

func Load() Config {
	port := getEnv("HTTP_PORT", "")
	if port == "" {
		port = getEnv("PORT", "8080")
	}

	return Config{
		HTTPPort:      port,
		TradingAPIURL: getEnv("TRADING_API_URL", "http://localhost:8000"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
