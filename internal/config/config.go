package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const DefaultDevelopmentJWTSecret = "dev-secret-change-me"

type Config struct {
	AppEnv         string
	AppPort        string
	GinMode        string
	FrontendOrigin string
	DB             DatabaseConfig
	Redis          RedisConfig
	SMTP           SMTPConfig
	JWT            JWTConfig
	Verification   VerificationConfig
	PasswordReset  PasswordResetConfig
	Invitation     InvitationConfig
	GoogleClientID string
	R2             R2Config
}

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Endpoint        string
	PublicBaseURL   string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type JWTConfig struct {
	Secret           string
	ExpiresInMinutes int
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type SMTPConfig struct {
	Host string
	Port int
	User string
	Pass string
	From string
}

type VerificationConfig struct {
	EmailVerifyURL string
	TokenTTLHours  int
}

type PasswordResetConfig struct {
	ResetURL        string
	TokenTTLMinutes int
}

type InvitationConfig struct {
	TokenTTLHours int
	FrontendAcceptURL string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:         getEnv("APP_ENV", "development"),
		AppPort:        getEnv("APP_PORT", "8080"),
		GinMode:        getEnv("GIN_MODE", "debug"),
		FrontendOrigin: getEnv("FRONTEND_ORIGIN", "http://localhost:3000"),
		DB: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "worksphere"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		SMTP: SMTPConfig{
			Host: getEnv("SMTP_HOST", "smtp.gmail.com"),
			Port: getEnvAsInt("SMTP_PORT", 587),
			User: getEnv("SMTP_USER", ""),
			Pass: getEnv("SMTP_PASS", ""),
			From: getEnv("SMTP_FROM", ""),
		},
		JWT: JWTConfig{
			Secret:           getEnv("JWT_SECRET", ""),
			ExpiresInMinutes: getEnvAsInt("JWT_EXPIRES_IN_MINUTES", 60),
		},
		Verification: VerificationConfig{
			EmailVerifyURL: getEnv("EMAIL_VERIFY_URL", "http://localhost:3000/verify-email"),
			TokenTTLHours:  getEnvAsInt("EMAIL_VERIFICATION_TTL_HOURS", 24),
		},
		PasswordReset: PasswordResetConfig{
			ResetURL:        getEnv("PASSWORD_RESET_URL", "http://localhost:3000/reset-password"),
			TokenTTLMinutes: getEnvAsInt("PASSWORD_RESET_TTL_MINUTES", 15),
		},
		Invitation: InvitationConfig{
			TokenTTLHours:     getEnvAsInt("INVITATION_TOKEN_TTL_HOURS", 72),
			FrontendAcceptURL: getEnv("INVITATION_ACCEPT_URL", "http://localhost:3000/invitations/accept"),
		},
		GoogleClientID: getEnv("GOOGLE_CLIENT_ID", ""),
		R2: R2Config{
			AccountID:       getEnv("R2_ACCOUNT_ID", ""),
			AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
			BucketName:      getEnv("R2_BUCKET_NAME", ""),
			Endpoint:        getEnv("R2_ENDPOINT", ""),
			PublicBaseURL:   getEnv("R2_PUBLIC_BASE_URL", ""),
		},
	}

	if cfg.AppPort == "" {
		return nil, fmt.Errorf("APP_PORT must not be empty")
	}

	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET must not be empty")
	}

	if strings.ToLower(cfg.AppEnv) != "development" && cfg.JWT.Secret == DefaultDevelopmentJWTSecret {
		return nil, fmt.Errorf("JWT_SECRET must not use the development default outside development")
	}

	if cfg.JWT.ExpiresInMinutes <= 0 {
		return nil, fmt.Errorf("JWT_EXPIRES_IN_MINUTES must be greater than 0")
	}

	if cfg.Verification.EmailVerifyURL == "" {
		return nil, fmt.Errorf("EMAIL_VERIFY_URL must not be empty")
	}

	if cfg.Verification.TokenTTLHours <= 0 {
		return nil, fmt.Errorf("EMAIL_VERIFICATION_TTL_HOURS must be greater than 0")
	}

	if cfg.PasswordReset.ResetURL == "" {
		return nil, fmt.Errorf("PASSWORD_RESET_URL must not be empty")
	}

	if cfg.PasswordReset.TokenTTLMinutes <= 0 {
		return nil, fmt.Errorf("PASSWORD_RESET_TTL_MINUTES must be greater than 0")
	}

	if cfg.Redis.DB < 0 {
		return nil, fmt.Errorf("REDIS_DB must be greater than or equal to 0")
	}

	if cfg.SMTP.Host == "" {
		return nil, fmt.Errorf("SMTP_HOST must not be empty")
	}

	if cfg.SMTP.Port <= 0 {
		return nil, fmt.Errorf("SMTP_PORT must be greater than 0")
	}

	if cfg.SMTP.User == "" {
		return nil, fmt.Errorf("SMTP_USER must not be empty")
	}

	if cfg.SMTP.Pass == "" {
		return nil, fmt.Errorf("SMTP_PASS must not be empty")
	}

	if cfg.SMTP.From == "" {
		return nil, fmt.Errorf("SMTP_FROM must not be empty")
	}

	if cfg.Invitation.TokenTTLHours <= 0 {
		return nil, fmt.Errorf("INVITATION_TOKEN_TTL_HOURS must be greater than 0")
	}

	if cfg.Invitation.FrontendAcceptURL == "" {
		return nil, fmt.Errorf("INVITATION_ACCEPT_URL must not be empty")
	}

	// Build R2 Endpoint if not provided
	if cfg.R2.Endpoint == "" && cfg.R2.AccountID != "" {
		cfg.R2.Endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.R2.AccountID)
	}

	// Validate R2 Configuration
	if cfg.R2.AccessKeyID == "" {
		return nil, fmt.Errorf("R2_ACCESS_KEY_ID must not be empty")
	}
	if cfg.R2.SecretAccessKey == "" {
		return nil, fmt.Errorf("R2_SECRET_ACCESS_KEY must not be empty")
	}
	if cfg.R2.BucketName == "" {
		return nil, fmt.Errorf("R2_BUCKET_NAME must not be empty")
	}
	if cfg.R2.Endpoint == "" {
		return nil, fmt.Errorf("R2_ENDPOINT (or R2_ACCOUNT_ID to build it) must not be empty")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}

	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
