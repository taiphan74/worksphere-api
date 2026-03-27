package auth

import (
	"net/http"

	apperrors "worksphere-api/pkg/errors"
)

var (
	// Registration errors
	ErrEmailAlreadyRegistered = apperrors.New(http.StatusConflict, "EMAIL_ALREADY_REGISTERED", "email is already registered")
	ErrInvalidEmail          = apperrors.New(http.StatusBadRequest, "INVALID_EMAIL", "invalid email address")
	ErrWeakPassword          = apperrors.New(http.StatusBadRequest, "WEAK_PASSWORD", "password must be at least 8 characters long")

	// Login errors
	ErrInvalidCredentials = apperrors.New(http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
	ErrUserSuspended      = apperrors.New(http.StatusForbidden, "USER_SUSPENDED", "your account has been suspended")
	ErrUserInactive       = apperrors.New(http.StatusForbidden, "USER_INACTIVE", "your account is not active")

	// Email verification errors
	ErrEmailNotVerified   = apperrors.New(http.StatusForbidden, "EMAIL_NOT_VERIFIED", "please verify your email first")
	ErrInvalidToken       = apperrors.New(http.StatusBadRequest, "INVALID_TOKEN", "invalid or expired token")
	ErrTokenExpired       = apperrors.New(http.StatusBadRequest, "TOKEN_EXPIRED", "token has expired")
	ErrUserNotFound       = apperrors.New(http.StatusNotFound, "USER_NOT_FOUND", "user not found")

	// Password reset errors
	ErrInvalidResetToken   = apperrors.New(http.StatusBadRequest, "INVALID_RESET_TOKEN", "invalid or expired reset token")
	ErrResetTokenExpired   = apperrors.New(http.StatusBadRequest, "RESET_TOKEN_EXPIRED", "reset token has expired")
	ErrPasswordSameAsOld   = apperrors.New(http.StatusBadRequest, "PASSWORD_SAME_AS_OLD", "new password must be different from old password")

	// Google login errors
	ErrInvalidGoogleToken  = apperrors.New(http.StatusUnauthorized, "INVALID_GOOGLE_TOKEN", "invalid Google token")
	ErrGoogleEmailMismatch = apperrors.New(http.StatusBadRequest, "GOOGLE_EMAIL_MISMATCH", "email from Google token does not match")

	// Rate limit errors
	ErrTooManyRequests     = apperrors.New(http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "too many attempts, please try again later")

	// Generic errors
	ErrInvalidInput        = apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "invalid input")
	ErrInvalidRequest      = apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	ErrInternalServer      = apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "something went wrong")
)
