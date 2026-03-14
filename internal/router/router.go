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

type AuthHandler interface {
	Register(*gin.Context)
	Login(*gin.Context)
	VerifyEmail(*gin.Context)
	ResendVerification(*gin.Context)
	ForgotPassword(*gin.Context)
	ResetPassword(*gin.Context)
	GoogleLogin(*gin.Context)
	Me(*gin.Context)
}

type UserRouteRegistrar interface {
	RegisterRoutes(*gin.RouterGroup)
}

type Groups struct {
	Public    *gin.RouterGroup
	Protected *gin.RouterGroup
}

func New(cfg *config.Config, logger *slog.Logger, redisClient *redis.Client) *gin.Engine {
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
	api.GET("/health", func(c *gin.Context) {
		appStatus := "ok"
		redisStatus := "disabled"
		statusCode := http.StatusOK

		if redisClient != nil {
			redisStatus = "ok"
			ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
			defer cancel()

			if err := redisClient.Ping(ctx).Err(); err != nil {
				appStatus = "degraded"
				redisStatus = "unavailable"
				statusCode = http.StatusServiceUnavailable
			}
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

func RegisterAuthRoutes(
	groups Groups,
	handler AuthHandler,
	registerIPMiddleware gin.HandlerFunc,
	loginIPMiddleware gin.HandlerFunc,
) {
	authPublic := groups.Public.Group("/auth")
	if registerIPMiddleware != nil {
		authPublic.POST("/register", registerIPMiddleware, handler.Register)
	} else {
		authPublic.POST("/register", handler.Register)
	}

	if loginIPMiddleware != nil {
		authPublic.POST("/login", loginIPMiddleware, handler.Login)
	} else {
		authPublic.POST("/login", handler.Login)
	}
	authPublic.GET("/verify-email", handler.VerifyEmail)
	authPublic.POST("/resend-verification", handler.ResendVerification)
	authPublic.POST("/forgot-password", handler.ForgotPassword)
	authPublic.POST("/reset-password", handler.ResetPassword)
	authPublic.POST("/google", handler.GoogleLogin)

	authProtected := groups.Protected.Group("/auth")
	authProtected.GET("/me", handler.Me)
}

func RegisterUserRoutes(groups Groups, handler UserRouteRegistrar) {
	protectedGroup := groups.Protected.Group("/users")
	handler.RegisterRoutes(protectedGroup)
}
