package router

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"worksphere-api/internal/config"
	"worksphere-api/internal/middleware"
	"worksphere-api/pkg/response"
)

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
