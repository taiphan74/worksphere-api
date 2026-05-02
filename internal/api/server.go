package api

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	authmodule "worksphere-api/internal/authmodule"
	authjwt "worksphere-api/internal/auth/jwt"
	"worksphere-api/internal/config"
	"worksphere-api/internal/email"
	"worksphere-api/internal/middleware"
	profilemodule "worksphere-api/internal/profilemodule"
	"worksphere-api/internal/ratelimit"
	"worksphere-api/internal/router"
	"worksphere-api/internal/storage"
	taskmodule "worksphere-api/internal/taskmodule"
	usermodule "worksphere-api/internal/usermodule"
	"worksphere-api/internal/verification"
	workspacemodule "worksphere-api/internal/workspacemodule"
)

// SetupRouter initializes all repositories, services, handlers and registers the routes.
func SetupRouter(cfg *config.Config, logger *slog.Logger, dbPool *pgxpool.Pool, redisClient *redis.Client) (*gin.Engine, error) {
	rateLimitService := ratelimit.NewService(redisClient, logger)
	globalIPMiddleware := ratelimit.GlobalIPMiddleware(rateLimitService)
	registerIPMiddleware := ratelimit.RegisterIPMiddleware(rateLimitService)
	loginIPMiddleware := ratelimit.LoginIPMiddleware(rateLimitService)
	emailService := email.NewSMTPService(cfg.SMTP)

	verificationService := verification.NewService(redisClient, cfg.Verification.TokenTTL)
	passwordResetService := verification.NewPasswordResetService(
		verification.NewService(redisClient, cfg.PasswordReset.TokenTTL),
	)

	tokenManager := authjwt.NewManager(cfg.JWT)

	r2Storage, err := storage.NewR2Storage(cfg.R2)
	if err != nil {
		return nil, err
	}

	// Auth Domain
	authHandler := authmodule.Setup(authmodule.AuthDeps{
		DBPool:               dbPool,
		RedisClient:          redisClient,
		TokenManager:         tokenManager,
		EmailService:         emailService,
		RateLimitService:     rateLimitService,
		VerificationService:  verificationService,
		PasswordResetService: passwordResetService,
		Logger:               logger,
		Config: authmodule.AuthConfig{
			EmailVerifyURL:    cfg.Verification.EmailVerifyURL,
			PasswordResetURL:  cfg.PasswordReset.ResetURL,
			GoogleClientID:    cfg.GoogleClientID,
			AppEnv:            cfg.AppEnv,
			AccessTTL:  cfg.JWT.AccessTTL,
			RefreshTTL: cfg.JWT.RefreshTTL,
		},
	})

	// User Domain
	userHandler := usermodule.Setup(usermodule.UserDeps{DBPool: dbPool})

	// Profile Domain
	profileHandler := profilemodule.Setup(profilemodule.ProfileDeps{
		DBPool:         dbPool,
		Storage:        r2Storage,
		AvatarUploadTTL: cfg.Profile.AvatarUploadURLTTL,
		AvatarViewTTL:   cfg.Profile.AvatarViewURLTTL,
	})

	// Workspace Domain
	workspaceHandler := workspacemodule.Setup(workspacemodule.WorkspaceDeps{
		DBPool: dbPool,
	})

	memberHandler := workspacemodule.SetupMember(workspacemodule.MemberDeps{
		DBPool: dbPool,
	})

	invitationHandler := workspacemodule.SetupInvitation(workspacemodule.InvitationDeps{
		DBPool:       dbPool,
		EmailService: emailService,
		Config: workspacemodule.WorkspaceConfig{
			InvitationFrontendURL: cfg.Invitation.FrontendAcceptURL,
			SMTPFrom:              cfg.SMTP.From,
		},
	})

	// Task Domain
	taskHandler := taskmodule.Setup(taskmodule.TaskDeps{
		DBPool: dbPool,
	})

	// Router Setup
	engine := router.New(cfg, logger, redisClient, globalIPMiddleware)
	groups := router.NewGroups(engine, middleware.JWTAuth(tokenManager))

	// Route Registration
	authHandler.RegisterRoutes(groups, registerIPMiddleware, loginIPMiddleware)
	userHandler.RegisterRoutes(groups)
	profileHandler.RegisterRoutes(groups)
	workspaceHandler.RegisterRoutes(groups)
	memberHandler.RegisterRoutes(groups)
	invitationHandler.RegisterRoutes(groups)
	taskHandler.RegisterRoutes(groups)

	return engine, nil
}