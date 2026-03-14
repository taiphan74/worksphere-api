package service

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"crypto/rand"
	"encoding/base64"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"

	"worksphere-api/internal/auth/dto"
	"worksphere-api/internal/auth/jwt"
	"worksphere-api/internal/auth/repository"
	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/user"
	"worksphere-api/internal/verification"
	apperrors "worksphere-api/pkg/errors"
	"worksphere-api/pkg/validation"
)

type TokenManager interface {
	GenerateAccessToken(userID uuid.UUID, email string) (string, error)
	ParseAccessToken(tokenString string) (*jwt.Claims, error)
}

type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (RegisterResult, error)
	Login(ctx context.Context, req dto.LoginRequest) (user.User, string, error)
	LoginWithGoogle(ctx context.Context, req dto.GoogleLoginRequest) (user.User, string, error)
	GetCurrentUser(ctx context.Context, userID uuid.UUID) (user.User, error)
	VerifyEmail(ctx context.Context, token string) (user.User, error)
	ResendVerification(ctx context.Context, email string) (ResendVerificationResult, error)
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token string, newPassword string) error
}

type RateLimiter interface {
	IsEmailLocked(ctx context.Context, email string) bool
	AllowResendVerificationEmail(ctx context.Context, email string) (bool, int, error)
	IncrementFailedLogin(ctx context.Context, email string)
	ClearFailedLogin(ctx context.Context, email string)
}

type authService struct {
	repo                 repository.AuthRepository
	tokenManager         TokenManager
	rateLimiter          RateLimiter
	verificationService  verification.Service
	passwordResetService verification.PasswordResetService
	emailSender          EmailSender
	logger               *slog.Logger
	emailVerifyURL       string
	passwordResetURL     string
	googleClientID       string
}

type RegisterResult struct {
	AccessToken           string
	User                  user.User
	VerificationEmailSent bool
}

type ResendVerificationResult struct {
	RateLimited       bool
	RetryAfterSeconds int
}

type EmailSender interface {
	SendHTML(ctx context.Context, to string, subject string, html string) error
}

func NewAuthService(
	repo repository.AuthRepository,
	tokenManager TokenManager,
	rateLimiter RateLimiter,
	verificationService verification.Service,
	passwordResetService verification.PasswordResetService,
	emailSender EmailSender,
	logger *slog.Logger,
	emailVerifyURL string,
	passwordResetURL string,
	googleClientID string,
) AuthService {
	if logger == nil {
		logger = slog.Default()
	}

	return &authService{
		repo:                 repo,
		tokenManager:         tokenManager,
		rateLimiter:          rateLimiter,
		verificationService:  verificationService,
		passwordResetService: passwordResetService,
		emailSender:          emailSender,
		logger:               logger,
		emailVerifyURL:       strings.TrimSpace(emailVerifyURL),
		passwordResetURL:     strings.TrimSpace(passwordResetURL),
		googleClientID:       strings.TrimSpace(googleClientID),
	}
}

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (RegisterResult, error) {
	input, err := normalizeRegisterInput(req)
	if err != nil {
		return RegisterResult{}, err
	}

	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		return RegisterResult{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
	}

	record, err := s.repo.CreateUserWithPassword(ctx, db.CreateUserWithPasswordParams{
		ID:           uuid.New(),
		Email:        input.Email,
		PasswordHash: passwordHash,
		FullName:     pgtype.Text{Valid: false},
		IsVerified:   false,
		Status:       "ACTIVE",
	})
	if err != nil {
		return RegisterResult{}, mapAuthRepositoryError(err, "failed to register user")
	}

	accessToken, err := s.tokenManager.GenerateAccessToken(parseUUID(record.ID), record.Email)
	if err != nil {
		return RegisterResult{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate access token")
	}

	if err := s.sendVerificationEmail(ctx, record); err != nil {
		s.logger.Warn("user registered but verification email was not sent", "user_id", record.ID, "email", record.Email, "error", err)
		return RegisterResult{
			AccessToken:           accessToken,
			User:                  record,
			VerificationEmailSent: false,
		}, nil
	}

	return RegisterResult{
		AccessToken:           accessToken,
		User:                  record,
		VerificationEmailSent: true,
	}, nil
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

	if !authUser.User.IsVerified {
		return s.failLogin(ctx, email, apperrors.New(http.StatusForbidden, "EMAIL_NOT_VERIFIED", "email is not verified"))
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

func (s *authService) LoginWithGoogle(ctx context.Context, req dto.GoogleLoginRequest) (user.User, string, error) {
	payload, err := idtoken.Validate(ctx, req.IDToken, s.googleClientID)
	if err != nil {
		return user.User{}, "", apperrors.New(http.StatusUnauthorized, "INVALID_TOKEN", "invalid google token")
	}

	emailVerified, ok := payload.Claims["email_verified"].(bool)
	if !ok || !emailVerified {
		return user.User{}, "", apperrors.New(http.StatusUnauthorized, "EMAIL_NOT_VERIFIED", "google email not verified")
	}

	email := validation.NormalizeEmail(payload.Claims["email"].(string))
	fullName := strings.TrimSpace(payload.Claims["name"].(string))
	if req.FullName != "" {
		fullName = req.FullName
	}

	authUser, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			password, _ := generateRandomString(32)
			hashedPassword, _ := hashPassword(password)

			record, err := s.repo.CreateUserWithPassword(ctx, db.CreateUserWithPasswordParams{
				ID:           uuid.New(),
				Email:        email,
				PasswordHash: hashedPassword,
				FullName:     pgtype.Text{String: fullName, Valid: fullName != ""},
				IsVerified:   true,
				Status:       "ACTIVE",
			})
			if err != nil {
				return user.User{}, "", mapAuthRepositoryError(err, "failed to create google user")
			}
			authUser.User = record
		} else {
			return user.User{}, "", mapAuthRepositoryError(err, "failed to find user")
		}
	}

	if authUser.User.Status != "ACTIVE" {
		return user.User{}, "", apperrors.New(http.StatusForbidden, "USER_INACTIVE", "user is not active")
	}

	token, err := s.tokenManager.GenerateAccessToken(parseUUID(authUser.User.ID), authUser.User.Email)
	if err != nil {
		return user.User{}, "", apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate access token")
	}

	return authUser.User, token, nil
}

func generateRandomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *authService) GetCurrentUser(ctx context.Context, userID uuid.UUID) (user.User, error) {
	record, err := s.repo.GetUserByIDForAuthProfile(ctx, userID)
	if err != nil {
		return user.User{}, mapAuthRepositoryError(err, "failed to get current user")
	}

	return record, nil
}

func (s *authService) VerifyEmail(ctx context.Context, token string) (user.User, error) {
	if strings.TrimSpace(token) == "" {
		return user.User{}, apperrors.New(http.StatusBadRequest, "INVALID_TOKEN", "token is required")
	}

	userID, err := s.verificationService.GetEmailVerificationUserID(ctx, token)
	if err != nil {
		return user.User{}, mapVerificationError(err)
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return user.User{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to parse verification user")
	}

	verifiedUser, err := s.repo.MarkUserEmailVerified(ctx, parsedUserID)
	if err != nil {
		return user.User{}, mapAuthRepositoryError(err, "failed to verify email")
	}

	if err := s.verificationService.DeleteEmailVerificationToken(ctx, token, userID); err != nil {
		return user.User{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete verification token")
	}

	return verifiedUser, nil
}

func (s *authService) ResendVerification(ctx context.Context, email string) (ResendVerificationResult, error) {
	normalizedEmail := validation.NormalizeEmail(email)
	if normalizedEmail == "" {
		return ResendVerificationResult{}, apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "email is required")
	}

	if !validation.IsValidEmail(normalizedEmail) {
		return ResendVerificationResult{}, apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "invalid email")
	}

	if s.rateLimiter != nil {
		allowed, retryAfterSeconds, err := s.rateLimiter.AllowResendVerificationEmail(ctx, normalizedEmail)
		if err != nil {
			s.logger.Warn("resend verification email rate limit check failed, allowing request", "email", normalizedEmail, "error", err)
		} else if !allowed {
			return ResendVerificationResult{
				RateLimited:       true,
				RetryAfterSeconds: retryAfterSeconds,
			}, nil
		}
	}

	authUser, err := s.repo.GetUserByEmail(ctx, normalizedEmail)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return ResendVerificationResult{}, nil
		}

		return ResendVerificationResult{}, mapAuthRepositoryError(err, "failed to resend verification email")
	}

	if authUser.User.IsVerified {
		return ResendVerificationResult{}, nil
	}

	if err := s.sendVerificationEmail(ctx, authUser.User); err != nil {
		s.logger.Warn("failed to resend verification email", "user_id", authUser.User.ID, "email", authUser.User.Email, "error", err)
	}

	return ResendVerificationResult{}, nil
}

func (s *authService) ForgotPassword(ctx context.Context, email string) error {
	normalizedEmail := validation.NormalizeEmail(email)
	if normalizedEmail == "" {
		return apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "email is required")
	}

	if !validation.IsValidEmail(normalizedEmail) {
		return apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "invalid email")
	}

	authUser, err := s.repo.GetUserByEmail(ctx, normalizedEmail)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil
		}

		return mapAuthRepositoryError(err, "failed to process forgot password")
	}

	return s.sendPasswordResetEmail(ctx, authUser.User)
}

func (s *authService) ResetPassword(ctx context.Context, token string, newPassword string) error {
	rawToken := strings.TrimSpace(token)
	password := strings.TrimSpace(newPassword)
	if rawToken == "" {
		return apperrors.New(http.StatusBadRequest, "INVALID_TOKEN", "token is required")
	}
	if len(password) < 8 {
		return apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "new_password must be at least 8 characters")
	}

	userID, err := s.passwordResetService.GetPasswordResetUserID(ctx, rawToken)
	if err != nil {
		return mapPasswordResetError(err)
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to parse reset password user")
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
	}

	if _, err := s.repo.ResetUserPassword(ctx, parsedUserID, passwordHash); err != nil {
		return mapAuthRepositoryError(err, "failed to reset password")
	}

	if err := s.passwordResetService.DeletePasswordResetToken(ctx, rawToken, userID); err != nil {
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete reset token")
	}

	return nil
}

type registerInput struct {
	Email    string
	Password string
}

func normalizeRegisterInput(req dto.RegisterRequest) (registerInput, error) {
	email := validation.NormalizeEmail(req.Email)
	password := strings.TrimSpace(req.Password)

	if email == "" || password == "" {
		return registerInput{}, apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "email and password are required")
	}

	if !validation.IsValidEmail(email) {
		return registerInput{}, apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "invalid email")
	}

	if len(password) < 8 {
		return registerInput{}, apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "password must be at least 8 characters")
	}

	return registerInput{
		Email:    email,
		Password: password,
	}, nil
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

func (s *authService) sendVerificationEmail(ctx context.Context, record user.User) error {
	if s.verificationService == nil || s.emailSender == nil {
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "email verification is not configured")
	}

	rawToken, err := s.verificationService.GenerateEmailVerificationToken(ctx, record.ID)
	if err != nil {
		return mapVerificationError(err)
	}

	verifyLink, err := buildVerificationLink(s.emailVerifyURL, rawToken)
	if err != nil {
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to build verification link")
	}

	body := fmt.Sprintf(
		"<p>Please verify your email address for WorkSphere.</p><p><a href=\"%s\">Verify email</a></p>",
		verifyLink,
	)
	if err := s.emailSender.SendHTML(ctx, record.Email, "Verify your WorkSphere email", body); err != nil {
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to send verification email")
	}

	return nil
}

func (s *authService) sendPasswordResetEmail(ctx context.Context, record user.User) error {
	if s.passwordResetService == nil || s.emailSender == nil {
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "password reset is not configured")
	}

	rawToken, err := s.passwordResetService.GenerateResetToken(ctx, record.ID)
	if err != nil {
		return mapPasswordResetError(err)
	}

	resetLink, err := buildVerificationLink(s.passwordResetURL, rawToken)
	if err != nil {
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to build reset password link")
	}

	body := fmt.Sprintf(
		"<p>You requested a password reset for WorkSphere.</p><p><a href=\"%s\">Reset password</a></p>",
		resetLink,
	)
	if err := s.emailSender.SendHTML(ctx, record.Email, "Reset your password", body); err != nil {
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to send password reset email")
	}

	return nil
}

func buildVerificationLink(baseURL, token string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", err
	}

	query := parsed.Query()
	query.Set("token", token)
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

func mapVerificationError(err error) error {
	switch {
	case err == nil:
		return nil
	case stderrors.Is(err, verification.ErrInvalidToken):
		return apperrors.New(http.StatusBadRequest, "INVALID_OR_EXPIRED_TOKEN", "token is invalid or expired")
	case stderrors.Is(err, verification.ErrServiceUnavailable):
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "verification service unavailable")
	default:
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "verification process failed")
	}
}

func mapPasswordResetError(err error) error {
	switch {
	case err == nil:
		return nil
	case stderrors.Is(err, verification.ErrInvalidToken):
		return apperrors.New(http.StatusBadRequest, "INVALID_TOKEN", "invalid or expired reset token")
	case stderrors.Is(err, verification.ErrServiceUnavailable):
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "password reset service unavailable")
	default:
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "password reset process failed")
	}
}
