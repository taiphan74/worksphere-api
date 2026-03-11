package service

import (
	"context"
	stderrors "errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"worksphere-api/internal/auth/dto"
	"worksphere-api/internal/auth/jwt"
	"worksphere-api/internal/auth/repository"
	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/user"
	apperrors "worksphere-api/pkg/errors"
	"worksphere-api/pkg/validation"
)

type TokenManager interface {
	GenerateAccessToken(userID uuid.UUID, email string) (string, error)
	ParseAccessToken(tokenString string) (*jwt.Claims, error)
}

type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (user.User, string, error)
	Login(ctx context.Context, req dto.LoginRequest) (user.User, string, error)
	GetCurrentUser(ctx context.Context, userID uuid.UUID) (user.User, error)
}

type LoginRateLimiter interface {
	IsEmailLocked(ctx context.Context, email string) bool
	IncrementFailedLogin(ctx context.Context, email string)
	ClearFailedLogin(ctx context.Context, email string)
}

type authService struct {
	repo         repository.AuthRepository
	tokenManager TokenManager
	rateLimiter  LoginRateLimiter
}

func NewAuthService(repo repository.AuthRepository, tokenManager TokenManager, rateLimiter LoginRateLimiter) AuthService {
	return &authService{
		repo:         repo,
		tokenManager: tokenManager,
		rateLimiter:  rateLimiter,
	}
}

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (user.User, string, error) {
	email, password, err := normalizeRegisterInput(req)
	if err != nil {
		return user.User{}, "", err
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return user.User{}, "", apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
	}

	record, err := s.repo.CreateUserWithPassword(ctx, db.CreateUserWithPasswordParams{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return user.User{}, "", mapAuthRepositoryError(err, "failed to register user")
	}

	token, err := s.tokenManager.GenerateAccessToken(parseUUID(record.ID), record.Email)
	if err != nil {
		return user.User{}, "", apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate access token")
	}

	return record, token, nil
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (user.User, string, error) {
	email := validation.NormalizeEmail(req.Email)
	password := strings.TrimSpace(req.Password)

	if email == "" || password == "" {
		return user.User{}, "", apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "email and password are required")
	}

	if !validation.IsValidEmail(email) {
		return user.User{}, "", apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "invalid email")
	}

	if s.rateLimiter != nil && s.rateLimiter.IsEmailLocked(ctx, email) {
		return user.User{}, "", apperrors.New(http.StatusTooManyRequests, "ACCOUNT_TEMPORARILY_LOCKED", "too many failed login attempts")
	}

	authUser, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return s.failLogin(ctx, email, apperrors.New(http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid credentials"))
		}

		return user.User{}, "", mapAuthRepositoryError(err, "failed to login")
	}

	// Check user status before verifying password
	switch authUser.User.Status {
	case "SUSPENDED":
		return s.failLogin(ctx, email, apperrors.New(http.StatusForbidden, "USER_SUSPENDED", "user is suspended"))
	case "INACTIVE":
		return s.failLogin(ctx, email, apperrors.New(http.StatusForbidden, "USER_INACTIVE", "user is inactive"))
	}

	if authUser.PasswordHash == "" {
		return s.failLogin(ctx, email, apperrors.New(http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid credentials"))
	}

	if err := bcrypt.CompareHashAndPassword([]byte(authUser.PasswordHash), []byte(password)); err != nil {
		return s.failLogin(ctx, email, apperrors.New(http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid credentials"))
	}

	token, err := s.tokenManager.GenerateAccessToken(parseUUID(authUser.User.ID), authUser.User.Email)
	if err != nil {
		return user.User{}, "", apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate access token")
	}

	if s.rateLimiter != nil {
		s.rateLimiter.ClearFailedLogin(ctx, email)
	}

	return authUser.User, token, nil
}

func (s *authService) GetCurrentUser(ctx context.Context, userID uuid.UUID) (user.User, error) {
	record, err := s.repo.GetUserByIDForAuthProfile(ctx, userID)
	if err != nil {
		return user.User{}, mapAuthRepositoryError(err, "failed to get current user")
	}

	return record, nil
}

func normalizeRegisterInput(req dto.RegisterRequest) (string, string, error) {
	email := validation.NormalizeEmail(req.Email)
	password := strings.TrimSpace(req.Password)

	if email == "" || password == "" {
		return "", "", apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "email and password are required")
	}

	if !validation.IsValidEmail(email) {
		return "", "", apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "invalid email")
	}

	if len(password) < 8 {
		return "", "", apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "password must be at least 8 characters")
	}

	return email, password, nil
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashed), nil
}

func mapAuthRepositoryError(err error, fallbackMessage string) error {
	if stderrors.Is(err, pgx.ErrNoRows) {
		return apperrors.New(http.StatusNotFound, "USER_NOT_FOUND", "user not found")
	}

	var pgErr *pgconn.PgError
	if apperrors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperrors.New(http.StatusConflict, "EMAIL_ALREADY_EXISTS", "email already exists")
	}

	return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", fallbackMessage)
}

func parseUUID(id string) uuid.UUID {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil
	}

	return parsed
}

func (s *authService) failLogin(ctx context.Context, email string, err error) (user.User, string, error) {
	if s.rateLimiter != nil {
		s.rateLimiter.IncrementFailedLogin(ctx, email)
	}

	return user.User{}, "", err
}
