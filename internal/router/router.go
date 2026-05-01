package router

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"worksphere-api/internal/config"
	"worksphere-api/internal/middleware"
	"worksphere-api/pkg/response"
)

// RouteRegistrar là interface chung cho các handler tự đăng ký route.
type RouteRegistrar interface {
	RegisterRoutes(groups Groups, middlewares ...gin.HandlerFunc)
}

type Groups struct {
	Public    *gin.RouterGroup
	Protected *gin.RouterGroup
}

func New(cfg *config.Config, logger *slog.Logger, redisClient *redis.Client, middlewares ...gin.HandlerFunc) *gin.Engine {
	engine := gin.New()

	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendOrigin},
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	engine.Use(middleware.RequestID())
	engine.Use(middleware.Logger(logger))
	engine.Use(middleware.Recovery(logger))

	api := engine.Group("/api")
	for _, handler := range middlewares {
		if handler != nil {
			api.Use(handler)
		}
	}
	api.GET("/health", func(c *gin.Context) {
		appStatus := "ok"
		redisStatus := "ok"
		statusCode := http.StatusOK
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()

		if err := redisClient.Ping(ctx).Err(); err != nil {
			appStatus = "degraded"
			redisStatus = "unavailable"
			statusCode = http.StatusServiceUnavailable
		}

		response.Success(c, statusCode, gin.H{
			"status": appStatus,
			"env":    cfg.AppEnv,
			"redis":  redisStatus,
		}, "success")
	})

	return engine
}

func NewGroups(engine *gin.Engine, authMiddleware gin.HandlerFunc) Groups {
	api := engine.Group("/api")
	public := api.Group("")
	protected := api.Group("")
	protected.Use(authMiddleware)

	return Groups{
		Public:    public,
		Protected: protected,
	}
}