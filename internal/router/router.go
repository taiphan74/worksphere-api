package router

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"worksphere-api/internal/config"
	"worksphere-api/internal/middleware"
	"worksphere-api/pkg/response"
)

type RouteRegistrar interface {
	RegisterRoutes(*gin.RouterGroup)
}

type AuthRouteRegistrar interface {
	RegisterPublicRoutes(*gin.RouterGroup)
	RegisterProtectedRoutes(*gin.RouterGroup)
}

func New(cfg *config.Config, logger *slog.Logger) *gin.Engine {
	engine := gin.New()

	engine.Use(middleware.RequestID())
	engine.Use(middleware.Logger(logger))
	engine.Use(middleware.Recovery(logger))

	api := engine.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			response.Success(c, http.StatusOK, gin.H{
				"status": "ok",
				"env":    cfg.AppEnv,
			}, "success")
		})
	}

	return engine
}

func RegisterUserRoutes(engine *gin.Engine, handler RouteRegistrar) {
	api := engine.Group("/api")
	users := api.Group("/users")
	handler.RegisterRoutes(users)
}

func RegisterAuthRoutes(engine *gin.Engine, handler AuthRouteRegistrar, authMiddleware gin.HandlerFunc) {
	api := engine.Group("/api")
	auth := api.Group("/auth")

	handler.RegisterPublicRoutes(auth)

	protected := auth.Group("")
	protected.Use(authMiddleware)
	handler.RegisterProtectedRoutes(protected)
}
