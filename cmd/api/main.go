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

	authhandler "worksphere-api/internal/auth/handler"
	authjwt "worksphere-api/internal/auth/jwt"
	authrepository "worksphere-api/internal/auth/repository"
	authservice "worksphere-api/internal/auth/service"
	"worksphere-api/internal/config"
	"worksphere-api/internal/database"
	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/email"
	"worksphere-api/internal/middleware"
	"worksphere-api/internal/ratelimit"
	redisclient "worksphere-api/internal/redis"
	"worksphere-api/internal/router"
	userhandler "worksphere-api/internal/user/handler"
	userrepository "worksphere-api/internal/user/repository"
	userservice "worksphere-api/internal/user/service"
	"worksphere-api/internal/verification"
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
		logger.Warn("redis unavailable, continuing without redis", "error", err, "addr", cfg.Redis.Addr, "db", cfg.Redis.DB)
	} else {
		defer func() {
			if closeErr := redisClient.Close(); closeErr != nil {
				logger.Error("failed to close redis client", "error", closeErr)
			}
		}()
	}

	rateLimitService := ratelimit.NewService(redisClient, logger)
	registerIPMiddleware := ratelimit.RegisterIPMiddleware(rateLimitService)
	loginIPMiddleware := ratelimit.LoginIPMiddleware(rateLimitService)
	emailService := email.NewSMTPService(cfg.SMTP)
	verificationService := verification.NewService(redisClient, time.Duration(cfg.Verification.TokenTTLHours)*time.Hour)
	passwordResetService := verification.NewPasswordResetService(
		verification.NewService(redisClient, time.Duration(cfg.PasswordReset.TokenTTLMinutes)*time.Minute),
	)

	queries := db.New(dbPool)
	tokenManager := authjwt.NewManager(cfg.JWT)
	authRepo := authrepository.NewAuthRepository(queries)
	authService := authservice.NewAuthService(
		authRepo,
		tokenManager,
		rateLimitService,
		verificationService,
		passwordResetService,
		emailService,
		logger,
		cfg.Verification.EmailVerifyURL,
		cfg.PasswordReset.ResetURL,
		cfg.GoogleClientID,
	)
	authHandler := authhandler.NewAuthHandler(authService, rateLimitService)
	userRepo := userrepository.NewUserRepository(queries)
	userService := userservice.NewUserService(userRepo)
	userHandler := userhandler.NewUserHandler(userService)

	engine := router.New(cfg, logger, redisClient)
	groups := router.NewGroups(engine, middleware.JWTAuth(tokenManager))
	router.RegisterAuthRoutes(groups, authHandler, registerIPMiddleware, loginIPMiddleware)
	router.RegisterUserRoutes(groups, userHandler)

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
		"redis_enabled", redisClient != nil,
		"smtp_host", cfg.SMTP.Host,
		"smtp_port", cfg.SMTP.Port,
		"smtp_from", cfg.SMTP.From,
		"email_verify_url", cfg.Verification.EmailVerifyURL,
		"email_verification_ttl_hours", cfg.Verification.TokenTTLHours,
		"password_reset_url", cfg.PasswordReset.ResetURL,
		"password_reset_ttl_minutes", cfg.PasswordReset.TokenTTLMinutes,
		"email_enabled", emailService != nil,
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
