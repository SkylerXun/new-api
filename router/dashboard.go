package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetDashboardRouter(router *gin.Engine) {
	// CCSwitch performs a lightweight balance refresh using the API key. Keep
	// this endpoint on read-only token auth so an exhausted/expired key can
	// still report its wallet balance without being able to relay requests.
	readonlyRouter := router.Group("/")
	readonlyRouter.Use(middleware.RouteTag("old_api"))
	readonlyRouter.Use(gzip.Gzip(gzip.DefaultCompression))
	readonlyRouter.Use(middleware.GlobalAPIRateLimit())
	readonlyRouter.Use(middleware.CORS())
	readonlyRouter.Use(middleware.TokenAuthReadOnly())
	readonlyRouter.GET("/v1/usage", controller.GetWalletUsage)

	apiRouter := router.Group("/")
	apiRouter.Use(middleware.RouteTag("old_api"))
	apiRouter.Use(gzip.Gzip(gzip.DefaultCompression))
	apiRouter.Use(middleware.GlobalAPIRateLimit())
	apiRouter.Use(middleware.CORS())
	apiRouter.Use(middleware.TokenAuth())
	{
		apiRouter.GET("/dashboard/billing/subscription", controller.GetSubscription)
		apiRouter.GET("/v1/dashboard/billing/subscription", controller.GetSubscription)
		apiRouter.GET("/dashboard/billing/usage", controller.GetUsage)
		apiRouter.GET("/v1/dashboard/billing/usage", controller.GetUsage)
	}
}
