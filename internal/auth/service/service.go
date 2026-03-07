package service

import (
	"context"
	stderrors "errors"
	"net/http"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"worksphere-api/internal/auth/dto"
	"worksphere-api/internal/auth/jwt"
	"worksphere-api/internal/auth/repository"
	db "worksphere-api/internal/database/sqlc"
	apperrors "worksphere-api/pkg/errors"
)

type TokenManager interface {
	GenerateAccessToken(userID uuid.UUID, email string) (string, error)
	ParseAccessToken(tokenString string) (*jwt.Claims, error)
}

type AuthService struct {
	repo         repository.AuthRepository
	tokenManager TokenManager
}

func NewAuthService(repo repository.AuthRepository, tokenManager TokenManager) *AuthService {
	return &AuthService{
		repo:         repo,
		tokenManager: tokenManager,
	}
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (db.User, string, error) {
	email, fullName, password, err := normalizeRegisterInput(req)
	if err != nil {
		return db.User{}, "", err
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return db.User{}, "", apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
	}

	user, err := s.repo.CreateUserWithPassword(ctx, db.CreateUserWithPasswordParams{
		ID:           uuid.New(),
		Email:        email,
		FullName:     fullName,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return db.User{}, "", mapAuthRepositoryError(err, "failed to register user")
	}

	token, err := s.tokenManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return db.User{}, "", apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate access token")
	}

	return user, token, nil
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (db.User, string, error) {
	email := normalizeEmail(req.Email)
	password := strings.TrimSpace(req.Password)

	if email == "" || password == "" {
		return db.User{}, "", apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "email and password are required")
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return db.User{}, "", apperrors.New(http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid credentials")
		}

		return db.User{}, "", mapAuthRepositoryError(err, "failed to login")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return db.User{}, "", apperrors.New(http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid credentials")
	}

	token, err := s.tokenManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return db.User{}, "", apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate access token")
	}

	return user, token, nil
}

func (s *AuthService) GetCurrentUser(ctx context.Context, userID uuid.UUID) (db.User, error) {
	user, err := s.repo.GetUserByIDForAuthProfile(ctx, userID)
	if err != nil {
		return db.User{}, mapAuthRepositoryError(err, "failed to get current user")
	}

	return user, nil
}

func normalizeRegisterInput(req dto.RegisterRequest) (string, string, string, error) {
	email := normalizeEmail(req.Email)
	fullName := strings.TrimSpace(req.FullName)
	password := strings.TrimSpace(req.Password)

	if email == "" || fullName == "" || password == "" {
		return "", "", "", apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "email, full_name, and password are required")
	}

	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", "", "", apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "invalid email")
	}

	if len(password) < 8 {
		return "", "", "", apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "password must be at least 8 characters")
	}

	return email, fullName, password, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
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
