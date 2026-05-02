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

	"worksphere-api/internal/api"
	"worksphere-api/internal/config"
	"worksphere-api/internal/database"
	redisclient "worksphere-api/internal/redis"
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
	defer dbPool.Close()

	redisClient, err := redisclient.NewClient(cfg.Redis)
	if err != nil {
		logger.Error("failed to connect to redis", "error", err, "addr", cfg.Redis.Addr, "db", cfg.Redis.DB)
		os.Exit(1)
	}
	defer func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			logger.Error("failed to close redis client", "error", closeErr)
		}
	}()

	engine, err := api.SetupRouter(cfg, logger, dbPool, redisClient)
	if err != nil {
		logger.Error("failed to setup router", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info(
		"starting server",
		"env", cfg.AppEnv,
		"port", cfg.AppPort,
		"database", cfg.DB.Name,
		"redis_addr", cfg.Redis.Addr,
		"redis_db", cfg.Redis.DB,
		"redis_enabled", true,
		"smtp_host", cfg.SMTP.Host,
		"smtp_port", cfg.SMTP.Port,
		"smtp_from", cfg.SMTP.From,
		"email_verify_url", cfg.Verification.EmailVerifyURL,
		"email_verification_ttl", cfg.Verification.TokenTTL.String(),
		"password_reset_url", cfg.PasswordReset.ResetURL,
		"password_reset_ttl", cfg.PasswordReset.TokenTTL.String(),
		"email_enabled", cfg.SMTP.Host != "",
		"r2_bucket", cfg.R2.BucketName,
		"r2_endpoint", cfg.R2.Endpoint,
	)

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
	logger.Info("server stopped")
}
