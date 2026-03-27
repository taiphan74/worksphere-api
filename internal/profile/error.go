package profile

import (
	"net/http"

	apperrors "worksphere-api/pkg/errors"
)

var (
	// Profile errors
	ErrProfileNotFound     = apperrors.New(http.StatusNotFound, "PROFILE_NOT_FOUND", "profile not found")
	ErrInvalidFullName     = apperrors.New(http.StatusBadRequest, "INVALID_FULL_NAME", "full name cannot be empty")
	ErrFullNameTooLong     = apperrors.New(http.StatusBadRequest, "FULL_NAME_TOO_LONG", "full name cannot exceed 150 characters")
	ErrInvalidPhone        = apperrors.New(http.StatusBadRequest, "INVALID_PHONE", "invalid phone number")
	ErrPhoneTooLong        = apperrors.New(http.StatusBadRequest, "PHONE_TOO_LONG", "phone number cannot exceed 20 characters")
	ErrInvalidJobTitle     = apperrors.New(http.StatusBadRequest, "INVALID_JOB_TITLE", "invalid job title")
	ErrJobTitleTooLong     = apperrors.New(http.StatusBadRequest, "JOB_TITLE_TOO_LONG", "job title cannot exceed 100 characters")

	// Password change errors
	ErrCurrentPasswordRequired = apperrors.New(http.StatusBadRequest, "CURRENT_PASSWORD_REQUIRED", "current password is required")
	ErrNewPasswordRequired     = apperrors.New(http.StatusBadRequest, "NEW_PASSWORD_REQUIRED", "new password is required")
	ErrNewPasswordTooShort     = apperrors.New(http.StatusBadRequest, "NEW_PASSWORD_TOO_SHORT", "new password must be at least 8 characters long")
	ErrPasswordMismatch        = apperrors.New(http.StatusBadRequest, "PASSWORD_MISMATCH", "new password and confirmation do not match")
	ErrCurrentPasswordWrong    = apperrors.New(http.StatusUnauthorized, "CURRENT_PASSWORD_WRONG", "current password is incorrect")
	ErrPasswordSameAsOld       = apperrors.New(http.StatusBadRequest, "PASSWORD_SAME_AS_OLD", "new password must be different from current password")

	// Avatar errors
	ErrInvalidAvatarFormat   = apperrors.New(http.StatusBadRequest, "INVALID_AVATAR_FORMAT", "invalid avatar format")
	ErrAvatarTooLarge        = apperrors.New(http.StatusBadRequest, "AVATAR_TOO_LARGE", "avatar file is too large")
	ErrInvalidContentType    = apperrors.New(http.StatusBadRequest, "INVALID_CONTENT_TYPE", "only image files are allowed")
	ErrAvatarUploadFailed    = apperrors.New(http.StatusInternalServerError, "AVATAR_UPLOAD_FAILED", "failed to upload avatar")
	ErrAvatarNotFound        = apperrors.New(http.StatusNotFound, "AVATAR_NOT_FOUND", "avatar not found")

	// Generic errors
	ErrInvalidInput          = apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "invalid input")
	ErrInvalidRequest        = apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	ErrUserNotFound          = apperrors.New(http.StatusNotFound, "USER_NOT_FOUND", "user not found")
	ErrInternalServer        = apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "something went wrong")
)
