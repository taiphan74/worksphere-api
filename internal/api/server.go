package api

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	authhandler "worksphere-api/internal/auth/handler"
	authjwt "worksphere-api/internal/auth/jwt"
	authrepository "worksphere-api/internal/auth/repository"
	authservice "worksphere-api/internal/auth/service"
	"worksphere-api/internal/config"
	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/email"
	"worksphere-api/internal/middleware"
	profilehandler "worksphere-api/internal/profile/handler"
	profilerepository "worksphere-api/internal/profile/repository"
	profileservice "worksphere-api/internal/profile/service"
	"worksphere-api/internal/ratelimit"
	"worksphere-api/internal/router"
	"worksphere-api/internal/storage"
	userhandler "worksphere-api/internal/user/handler"
	userrepository "worksphere-api/internal/user/repository"
	userservice "worksphere-api/internal/user/service"
	"worksphere-api/internal/verification"
	workspacehandler "worksphere-api/internal/workspace/handler"
	workspacerepository "worksphere-api/internal/workspace/repository"
	workspaceservice "worksphere-api/internal/workspace/service"
)

// SetupRouter initializes all repositories, services, handlers and registers the routes.
func SetupRouter(cfg *config.Config, logger *slog.Logger, dbPool *pgxpool.Pool, redisClient *redis.Client) (*gin.Engine, error) {
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

	r2Storage, err := storage.NewR2Storage(cfg.R2)
	if err != nil {
		return nil, err
	}

	// Auth Domain
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

	// User Domain
	userRepo := userrepository.NewUserRepository(queries)
	userService := userservice.NewUserService(userRepo)
	userHandler := userhandler.NewUserHandler(userService)

	// Profile Domain
	profileRepo := profilerepository.NewProfileRepository(queries)
	profileService := profileservice.NewProfileService(profileRepo, r2Storage)
	profileHandler := profilehandler.NewProfileHandler(profileService)

	// Workspace Domain
	workspaceRepo := workspacerepository.NewWorkspaceRepository(queries)
	workspaceService := workspaceservice.NewWorkspaceService(workspaceRepo)
	workspaceHandler := workspacehandler.NewWorkspaceHandler(workspaceService)

	// Router Setup
	engine := router.New(cfg, logger, redisClient)
	groups := router.NewGroups(engine, middleware.JWTAuth(tokenManager))

	// Route Registration
	router.RegisterAuthRoutes(groups, authHandler, registerIPMiddleware, loginIPMiddleware)
	router.RegisterUserRoutes(groups, userHandler)
	router.RegisterProfileRoutes(groups, profileHandler)
	router.RegisterWorkspaceRoutes(groups, workspaceHandler)

	return engine, nil
}
