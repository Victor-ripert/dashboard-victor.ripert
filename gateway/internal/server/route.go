package server

import (
	"github.com/gin-gonic/gin"

	"gateway/internal/api"
	"gateway/internal/config"
	"gateway/internal/middleware"
	"gateway/internal/pages"
	"gateway/internal/proxy"
)

func NewRouter(cfg config.Config) *gin.Engine {
	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())

	tradingAPI := proxy.NewIBAPIProxy(cfg.TradingAPIURL)

	registerPageRoutes(router)
	registerAPIRoutes(router, tradingAPI)

	return router
}

func registerPageRoutes(router *gin.Engine) {
	pages.RegisterRoutes(router)

	// TODO: Servir le frontend React compilé depuis ./frontend/dist.
	// TODO: Ajouter un fallback vers index.html pour les routes React.
}

func registerAPIRoutes(
	router *gin.Engine,
	tradingAPI *proxy.IBAPIProxy,
) {
	api.RegisterRoutes(router, tradingAPI)
}
