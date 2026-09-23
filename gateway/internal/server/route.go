package server

import (
	"github.com/gin-gonic/gin"

	"gateway/internal/api"
	"gateway/internal/config"
	"gateway/internal/proxy"
)

func NewRouter(cfg config.Config) *gin.Engine {
	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())

	tradingAPI := proxy.NewTradingAPIClient(cfg.TradingAPIURL)

	api.RegisterRoutes(router, tradingAPI)

	// TODO: Ajouter plus tard le service des fichiers statiques React.
	// TODO: Ajouter plus tard les middlewares d'authentification.
	// TODO: Ajouter plus tard la configuration CORS stricte.

	return router
}
