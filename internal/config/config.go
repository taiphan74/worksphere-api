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
	AppEnv  string
	AppPort string
	GinMode string
	DB      DatabaseConfig
	JWT     JWTConfig
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

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: getEnv("APP_PORT", "8080"),
		GinMode: getEnv("GIN_MODE", "debug"),
		DB: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "worksphere"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		JWT: JWTConfig{
			Secret:           getEnv("JWT_SECRET", ""),
			ExpiresInMinutes: getEnvAsInt("JWT_EXPIRES_IN_MINUTES", 60),
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
