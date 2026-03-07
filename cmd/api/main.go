package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"worksphere-api/internal/config"
	"worksphere-api/internal/database"
	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/router"
	userhandler "worksphere-api/internal/user/handler"
	userrepository "worksphere-api/internal/user/repository"
	userservice "worksphere-api/internal/user/service"
	applogger "worksphere-api/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	gin.SetMode(cfg.GinMode)

	logger := applogger.New(cfg.AppEnv)
	dbPool, err := database.NewPostgres(*cfg)
	if err != nil {
		logger.Error("failed to connect database", "error", err)
		os.Exit(1)
	}

	queries := db.New(dbPool)
	userRepo := userrepository.NewUserRepository(queries)
	userService := userservice.NewUserService(userRepo)
	userHandler := userhandler.NewUserHandler(userService)

	engine := router.New(cfg, logger)
	router.RegisterUserRoutes(engine, userHandler)

	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("starting server", "env", cfg.AppEnv, "port", cfg.AppPort, "database", cfg.DB.Name)

	go func() {
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", serveErr)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	dbPool.Close()
	logger.Info("server stopped")
}
