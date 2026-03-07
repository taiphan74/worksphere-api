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

type Groups struct {
	Public    *gin.RouterGroup
	Protected *gin.RouterGroup
}

func New(cfg *config.Config, logger *slog.Logger) *gin.Engine {
	engine := gin.New()

	engine.Use(middleware.RequestID())
	engine.Use(middleware.Logger(logger))
	engine.Use(middleware.Recovery(logger))

	api := engine.Group("/api")
	api.GET("/health", func(c *gin.Context) {
		response.Success(c, http.StatusOK, gin.H{
			"status": "ok",
			"env":    cfg.AppEnv,
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

func RegisterAuthRoutes(groups Groups, handler AuthRouteRegistrar) {
	authPublic := groups.Public.Group("/auth")
	handler.RegisterPublicRoutes(authPublic)

	authProtected := groups.Protected.Group("/auth")
	handler.RegisterProtectedRoutes(authProtected)
}

func RegisterUserRoutes(group *gin.RouterGroup, handler RouteRegistrar) {
	users := group.Group("/users")
	handler.RegisterRoutes(users)
}
