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
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/api/idtoken"

	"worksphere-api/internal/auth"
	"worksphere-api/internal/auth/dto"
	"worksphere-api/internal/auth/jwt"
	"worksphere-api/internal/auth/repository"
	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/user"
	"worksphere-api/internal/verification"
	apperrors "worksphere-api/pkg/errors"
	"worksphere-api/pkg/utils"
	"worksphere-api/pkg/validation"
)

type TokenManager interface {
	GenerateAccessToken(userID uuid.UUID, email string, roles []string) (string, error)
	GenerateRefreshToken(userID uuid.UUID, email string) (string, error)
	ParseAccessToken(tokenString string) (*jwt.Claims, error)
	ParseRefreshToken(tokenString string) (*jwt.Claims, error)
}

type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (RegisterResult, error)
	Login(ctx context.Context, req dto.LoginRequest) (user.User, string, string, error)
	LoginWithGoogle(ctx context.Context, req dto.GoogleLoginRequest) (user.User, string, string, error)
	RefreshToken(ctx context.Context, refreshToken string) (string, string, error)
	GetCurrentUser(ctx context.Context, userID uuid.UUID) (user.User, error)
	VerifyEmail(ctx context.Context, token string) (user.User, string, string, error)
	ResendVerification(ctx context.Context, email string, verificationUrl string) (ResendVerificationResult, error)
	ForgotPassword(ctx context.Context, email string, resetUrl string) error
	ResetPassword(ctx context.Context, token string, newPassword string) (user.User, string, string, error)
}

type RateLimiter interface {
	IsEmailLocked(ctx context.Context, email string) bool
	AllowResendVerificationEmail(ctx context.Context, email string) (bool, int, error)
	IncrementFailedLogin(ctx context.Context, email string)
	ClearFailedLogin(ctx context.Context, email string)
}

type authService struct {
	repo                 repository.AuthRepository
	systemRoleRepo       repository.SystemRoleRepository
	userSystemRoleRepo   repository.UserSystemRoleRepository
	refreshTokenRepo     repository.RefreshTokenRepository
	tokenManager         TokenManager
	rateLimiter          RateLimiter
	verificationService  verification.Service
	passwordResetService verification.PasswordResetService
	emailSender          EmailSender
	logger               *slog.Logger
	emailVerifyURL       string
	passwordResetURL     string
	googleClientID       string
	refreshTTL           time.Duration
}

type RegisterResult struct {
	AccessToken           string
	RefreshToken          string
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
	systemRoleRepo repository.SystemRoleRepository,
	userSystemRoleRepo repository.UserSystemRoleRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	tokenManager TokenManager,
	rateLimiter RateLimiter,
	verificationService verification.Service,
	passwordResetService verification.PasswordResetService,
	emailSender EmailSender,
	logger *slog.Logger,
	emailVerifyURL string,
	passwordResetURL string,
	googleClientID string,
	refreshTTL time.Duration,
) AuthService {
	if logger == nil {
		logger = slog.Default()
	}

	return &authService{
		repo:                 repo,
		systemRoleRepo:       systemRoleRepo,
		userSystemRoleRepo:   userSystemRoleRepo,
		refreshTokenRepo:     refreshTokenRepo,
		tokenManager:         tokenManager,
		rateLimiter:          rateLimiter,
		verificationService:  verificationService,
		passwordResetService: passwordResetService,
		emailSender:          emailSender,
		logger:               logger,
		emailVerifyURL:       strings.TrimSpace(emailVerifyURL),
		passwordResetURL:     strings.TrimSpace(passwordResetURL),
		googleClientID:       strings.TrimSpace(googleClientID),
		refreshTTL:           refreshTTL,
	}
}

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (RegisterResult, error) {
	input, err := normalizeRegisterInput(req)
	if err != nil {
		return RegisterResult{}, err
	}

	passwordHash, err := utils.HashPassword(input.Password)
	if err != nil {
		return RegisterResult{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
	}

	record, err := s.createUserWithDefaultRole(ctx, db.CreateUserWithPasswordParams{
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

	roles, err := s.getUserRoles(ctx, parseUUID(record.ID))
	if err != nil {
		s.logger.Warn("user registered but failed to get roles", "user_id", record.ID, "email", record.Email, "error", err)
		roles = []string{"USER"} // Fallback to default role
	}

	accessToken, refreshToken, err := s.generateTokenPair(ctx, parseUUID(record.ID), record.Email, roles)
	if err != nil {
		return RegisterResult{}, err
	}

	if err := s.sendVerificationEmail(ctx, record, req.VerificationUrl); err != nil {
		s.logger.Warn("user registered but verification email was not sent", "user_id", record.ID, "email", record.Email, "error", err)
		return RegisterResult{
			AccessToken:           accessToken,
			RefreshToken:          refreshToken,
			User:                  record,
			VerificationEmailSent: false,
		}, nil
	}

	return RegisterResult{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		User:                  record,
		VerificationEmailSent: true,
	}, nil
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (user.User, string, string, error) {
	email := validation.NormalizeEmail(req.Email)
	password := strings.TrimSpace(req.Password)

	if email == "" || password == "" {
		return user.User{}, "", "", apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "email and password are required")
	}

	if !validation.IsValidEmail(email) {
		return user.User{}, "", "", apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "invalid email")
	}

	if s.rateLimiter != nil && s.rateLimiter.IsEmailLocked(ctx, email) {
		return user.User{}, "", "", apperrors.New(http.StatusTooManyRequests, "ACCOUNT_TEMPORARILY_LOCKED", "too many failed login attempts")
	}

	authUser, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return s.failLogin(ctx, email, auth.ErrInvalidCredentials)
		}

		return user.User{}, "", "", mapAuthRepositoryError(err, "failed to login")
	}

	// Check user status before verifying password
	switch authUser.User.Status {
	case "SUSPENDED":
		return s.failLogin(ctx, email, auth.ErrUserSuspended)
	case "INACTIVE":
		return s.failLogin(ctx, email, auth.ErrUserInactive)
	}

	if !authUser.User.IsVerified {
		// Tự động gửi lại email verification (đã có sẵn rate limit bên trong ResendVerification)
		_, _ = s.ResendVerification(ctx, email, "")

		return s.failLogin(ctx, email, auth.ErrEmailNotVerified)
	}

	if authUser.PasswordHash == "" {
		return s.failLogin(ctx, email, auth.ErrInvalidCredentials)
	}

	if err := utils.ComparePassword(authUser.PasswordHash, password); err != nil {
		return s.failLogin(ctx, email, auth.ErrInvalidCredentials)
	}

	roles, err := s.getUserRoles(ctx, parseUUID(authUser.User.ID))
	if err != nil {
		s.logger.Warn("login successful but failed to get roles", "user_id", authUser.User.ID, "email", authUser.User.Email, "error", err)
		roles = []string{"USER"} // Fallback to default role
	}

	accessToken, refreshToken, err := s.generateTokenPair(ctx, parseUUID(authUser.User.ID), authUser.User.Email, roles)
	if err != nil {
		return user.User{}, "", "", err
	}

	if s.rateLimiter != nil {
		s.rateLimiter.ClearFailedLogin(ctx, email)
	}

	return authUser.User, accessToken, refreshToken, nil
}

func (s *authService) LoginWithGoogle(ctx context.Context, req dto.GoogleLoginRequest) (user.User, string, string, error) {
	payload, err := idtoken.Validate(ctx, req.IDToken, s.googleClientID)
	if err != nil {
		return user.User{}, "", "", apperrors.New(http.StatusUnauthorized, "INVALID_TOKEN", "invalid google token")
	}

	emailVerified, ok := payload.Claims["email_verified"].(bool)
	if !ok || !emailVerified {
		return user.User{}, "", "", auth.ErrEmailNotVerified
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
			hashedPassword, _ := utils.HashPassword(password)

			record, err := s.createUserWithDefaultRole(ctx, db.CreateUserWithPasswordParams{
				ID:           uuid.New(),
				Email:        email,
				PasswordHash: hashedPassword,
				FullName:     pgtype.Text{String: fullName, Valid: fullName != ""},
				IsVerified:   true,
				Status:       "ACTIVE",
			})
			if err != nil {
				return user.User{}, "", "", mapAuthRepositoryError(err, "failed to create google user")
			}
			authUser.User = record
		} else {
			return user.User{}, "", "", mapAuthRepositoryError(err, "failed to find user")
		}
	}

	if authUser.User.Status != "ACTIVE" {
		return user.User{}, "", "", auth.ErrUserInactive
	}

	roles, err := s.getUserRoles(ctx, parseUUID(authUser.User.ID))
	if err != nil {
		s.logger.Warn("google login successful but failed to get roles", "user_id", authUser.User.ID, "email", authUser.User.Email, "error", err)
		roles = []string{"USER"} // Fallback to default role
	}

	accessToken, refreshToken, err := s.generateTokenPair(ctx, parseUUID(authUser.User.ID), authUser.User.Email, roles)
	if err != nil {
		return user.User{}, "", "", err
	}

	return authUser.User, accessToken, refreshToken, nil
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

func (s *authService) VerifyEmail(ctx context.Context, token string) (user.User, string, string, error) {
	if strings.TrimSpace(token) == "" {
		return user.User{}, "", "", apperrors.New(http.StatusBadRequest, "INVALID_TOKEN", "token is required")
	}

	userID, err := s.verificationService.GetEmailVerificationUserID(ctx, token)
	if err != nil {
		return user.User{}, "", "", mapVerificationError(err)
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return user.User{}, "", "", apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to parse verification user ID")
	}

	verifiedUser, err := s.repo.MarkUserEmailVerified(ctx, parsedUserID)
	if err != nil {
		return user.User{}, "", "", mapAuthRepositoryError(err, "failed to verify email")
	}

	if err := s.verificationService.DeleteEmailVerificationToken(ctx, token, userID); err != nil {
		return user.User{}, "", "", apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete verification token")
	}

	roles, err := s.getUserRoles(ctx, parseUUID(verifiedUser.ID))
	if err != nil {
		roles = []string{"USER"}
	}

	accessToken, refreshToken, err := s.generateTokenPair(ctx, parseUUID(verifiedUser.ID), verifiedUser.Email, roles)
	if err != nil {
		return user.User{}, "", "", err
	}

	return verifiedUser, accessToken, refreshToken, nil
}

func (s *authService) ResendVerification(ctx context.Context, email string, verificationUrl string) (ResendVerificationResult, error) {
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

	if err := s.sendVerificationEmail(ctx, authUser.User, verificationUrl); err != nil {
		s.logger.Warn("failed to resend verification email", "user_id", authUser.User.ID, "email", authUser.User.Email, "error", err)
	}

	return ResendVerificationResult{}, nil
}

func (s *authService) ForgotPassword(ctx context.Context, email string, resetUrl string) error {
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

	return s.sendPasswordResetEmail(ctx, authUser.User, resetUrl)
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	claims, err := s.tokenManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return "", "", apperrors.New(http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired refresh token")
	}

	userID := claims.Subject

	storedHash, err := s.refreshTokenRepo.Get(ctx, userID)
	if err != nil {
		return "", "", apperrors.New(http.StatusUnauthorized, "INVALID_TOKEN", "invalid refresh token session")
	}

	if storedHash != hashString(refreshToken) {
		// Potential reuse attack or invalid session
		_ = s.refreshTokenRepo.Delete(ctx, userID)
		return "", "", apperrors.New(http.StatusUnauthorized, "INVALID_TOKEN", "token has been reused or invalid")
	}

	roles, err := s.getUserRoles(ctx, parseUUID(userID))
	if err != nil {
		roles = []string{"USER"}
	}

	// Rotate token: delete old, generate new
	_ = s.refreshTokenRepo.Delete(ctx, userID)

	return s.generateTokenPair(ctx, parseUUID(userID), claims.Email, roles)
}

func (s *authService) generateTokenPair(ctx context.Context, userID uuid.UUID, email string, roles []string) (string, string, error) {
	refreshToken, err := s.tokenManager.GenerateRefreshToken(userID, email)
	if err != nil {
		return "", "", apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate refresh token")
	}

	accessToken, err := s.tokenManager.GenerateAccessToken(userID, email, roles)
	if err != nil {
		return "", "", apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate access token")
	}

	// Store hashed refresh token in Redis
	if err := s.refreshTokenRepo.Set(ctx, userID.String(), hashString(refreshToken), s.refreshTTL); err != nil {
		return "", "", apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to store refresh session")
	}

	return accessToken, refreshToken, nil
}

func hashString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *authService) ResetPassword(ctx context.Context, token string, newPassword string) (user.User, string, string, error) {
	rawToken := strings.TrimSpace(token)
	password := strings.TrimSpace(newPassword)
	if rawToken == "" {
		return user.User{}, "", "", apperrors.New(http.StatusBadRequest, "INVALID_TOKEN", "token is required")
	}
	if len(password) < 8 {
		return user.User{}, "", "", apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "new_password must be at least 8 characters")
	}

	userID, err := s.passwordResetService.GetPasswordResetUserID(ctx, rawToken)
	if err != nil {
		return user.User{}, "", "", mapPasswordResetError(err)
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return user.User{}, "", "", apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to parse reset password user")
	}

	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return user.User{}, "", "", apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
	}

	u, err := s.repo.ResetUserPassword(ctx, parsedUserID, passwordHash)
	if err != nil {
		return user.User{}, "", "", mapAuthRepositoryError(err, "failed to reset password")
	}

	if err := s.passwordResetService.DeletePasswordResetToken(ctx, rawToken, userID); err != nil {
		return user.User{}, "", "", apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete reset token")
	}

	roles, err := s.getUserRoles(ctx, parsedUserID)
	if err != nil {
		s.logger.Warn("password reset successful but failed to get roles", "user_id", userID, "email", u.Email, "error", err)
		roles = []string{"USER"}
	}

	accessToken, refreshToken, err := s.generateTokenPair(ctx, parsedUserID, u.Email, roles)
	if err != nil {
		return user.User{}, "", "", err
	}

	return u, accessToken, refreshToken, nil
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

func mapAuthRepositoryError(err error, fallbackMessage string) error {
	if stderrors.Is(err, pgx.ErrNoRows) {
		return auth.ErrUserNotFound
	}

	var pgErr *pgconn.PgError
	if apperrors.As(err, &pgErr) && pgErr.Code == "23505" {
		return auth.ErrEmailAlreadyRegistered
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

func (s *authService) failLogin(ctx context.Context, email string, err error) (user.User, string, string, error) {
	if s.rateLimiter != nil {
		s.rateLimiter.IncrementFailedLogin(ctx, email)
	}

	return user.User{}, "", "", err
}

func (s *authService) sendVerificationEmail(ctx context.Context, record user.User, customVerificationUrl string) error {
	if s.verificationService == nil || s.emailSender == nil {
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "email verification is not configured")
	}

	rawToken, err := s.verificationService.GenerateEmailVerificationToken(ctx, record.ID)
	if err != nil {
		return mapVerificationError(err)
	}

	baseUrl := s.emailVerifyURL
	if customVerificationUrl != "" {
		baseUrl = customVerificationUrl
	}

	verifyLink, err := buildVerificationLink(baseUrl, rawToken)
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

func (s *authService) sendPasswordResetEmail(ctx context.Context, record user.User, customResetUrl string) error {
	if s.passwordResetService == nil || s.emailSender == nil {
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "password reset is not configured")
	}

	rawToken, err := s.passwordResetService.GenerateResetToken(ctx, record.ID)
	if err != nil {
		return mapPasswordResetError(err)
	}

	baseUrl := s.passwordResetURL
	if customResetUrl != "" {
		baseUrl = customResetUrl
	}

	resetLink, err := buildVerificationLink(baseUrl, rawToken)
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

func (s *authService) createUserWithDefaultRole(ctx context.Context, params db.CreateUserWithPasswordParams) (user.User, error) {
	createdUser, err := s.repo.CreateUserWithPassword(ctx, params)
	if err != nil {
		return user.User{}, err
	}

	if s.systemRoleRepo == nil || s.userSystemRoleRepo == nil {
		return createdUser, nil
	}

	defaultRole, err := s.systemRoleRepo.GetByCode(ctx, "USER")
	if err != nil {
		return user.User{}, err
	}

	userID, err := uuid.Parse(createdUser.ID)
	if err != nil {
		return user.User{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to parse created user")
	}

	if err := s.userSystemRoleRepo.AssignRole(ctx, userID, defaultRole.ID, nil); err != nil {
		return user.User{}, err
	}

	return createdUser, nil
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

func (s *authService) getUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	if s.userSystemRoleRepo == nil {
		return []string{"USER"}, nil
	}

	roles, err := s.userSystemRoleRepo.GetUserRoleCodes(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(roles) == 0 {
		return []string{"USER"}, nil
	}

	return roles, nil
}
