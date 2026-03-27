package user

import (
	"net/http"
	apperrors "worksphere-api/pkg/errors"
)

var (
	ErrUserNotFound       = apperrors.New(http.StatusNotFound, "USER_NOT_FOUND", "user not found")
	ErrEmailAlreadyExists  = apperrors.New(http.StatusConflict, "EMAIL_ALREADY_EXISTS", "email already exists")
	ErrInvalidCredentials = apperrors.New(http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid credentials")
	ErrUserSuspended      = apperrors.New(http.StatusForbidden, "USER_SUSPENDED", "user is suspended")
	ErrUserInactive       = apperrors.New(http.StatusForbidden, "USER_INACTIVE", "user is inactive")
	ErrEmailNotVerified   = apperrors.New(http.StatusForbidden, "EMAIL_NOT_VERIFIED", "email is not verified")
	ErrInvalidInput       = apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "invalid input")
	ErrInvalidRequest     = apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
)
