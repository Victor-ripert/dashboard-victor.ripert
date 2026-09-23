package main

import (
	"log"

	"gateway/internal/config"
	"gateway/internal/server"
)

func main() {
	cfg := config.Load()
	router := server.NewRouter(cfg)

	log.Printf("gateway listening on :%s", cfg.HTTPPort)

	if err := router.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatal(err)
	}
}
