package router

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"worksphere-api/internal/config"
	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/middleware"
	userhandler "worksphere-api/internal/user/handler"
	userrepository "worksphere-api/internal/user/repository"
	userservice "worksphere-api/internal/user/service"
	"worksphere-api/pkg/response"
)

func New(cfg *config.Config, logger *slog.Logger, queries *db.Queries) *gin.Engine {
	engine := gin.New()

	engine.Use(middleware.RequestID())
	engine.Use(middleware.Logger(logger))
	engine.Use(middleware.Recovery(logger))

	userRepo := userrepository.NewUserRepository(queries)
	userService := userservice.NewUserService(userRepo)
	userHandler := userhandler.NewUserHandler(userService)

	api := engine.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			response.Success(c, http.StatusOK, gin.H{
				"status": "ok",
				"env":    cfg.AppEnv,
			}, "success")
		})

		users := api.Group("/users")
		userHandler.RegisterRoutes(users)
	}

	return engine
}
