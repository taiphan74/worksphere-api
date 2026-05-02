package authmodule

import (
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	authhandler "worksphere-api/internal/auth/handler"
	authjwt "worksphere-api/internal/auth/jwt"
	authrepository "worksphere-api/internal/auth/repository"
	authservice "worksphere-api/internal/auth/service"
	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/email"
	"worksphere-api/internal/ratelimit"
	"worksphere-api/internal/verification"
)

// AuthConfig chứa config riêng cho auth module.
type AuthConfig struct {
	EmailVerifyURL   string
	PasswordResetURL string
	GoogleClientID   string
	AppEnv           string
	AccessTTL        time.Duration
	RefreshTTL       time.Duration
}

// AuthDeps chứa dependency cho auth module setup.
type AuthDeps struct {
	DBPool               *pgxpool.Pool
	RedisClient          *redis.Client
	TokenManager         *authjwt.Manager
	EmailService         email.Service
	RateLimitService     ratelimit.Service
	VerificationService  verification.Service
	PasswordResetService verification.PasswordResetService
	Logger               *slog.Logger
	Config               AuthConfig
}

// Setup khởi tạo auth repository, service, handler.
func Setup(deps AuthDeps) *authhandler.AuthHandler {
	queries := db.New(deps.DBPool)

	authRepo := authrepository.NewAuthRepository(queries)
	systemRoleRepo := authrepository.NewSystemRoleRepository(queries)
	userSystemRoleRepo := authrepository.NewUserSystemRoleRepository(queries)
	refreshTokenRepo := authrepository.NewRefreshTokenRepository(deps.RedisClient)

	authService := authservice.NewAuthService(
		authRepo,
		systemRoleRepo,
		userSystemRoleRepo,
		refreshTokenRepo,
		deps.TokenManager,
		deps.RateLimitService,
		deps.VerificationService,
		deps.PasswordResetService,
		deps.EmailService,
		deps.Logger,
		deps.Config.EmailVerifyURL,
		deps.Config.PasswordResetURL,
		deps.Config.GoogleClientID,
		deps.Config.RefreshTTL,
	)

	return authhandler.NewAuthHandler(authService, deps.RateLimitService, deps.Config.AppEnv, deps.Config.AccessTTL, deps.Config.RefreshTTL)
}