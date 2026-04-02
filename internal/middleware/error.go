package middleware

import apperrors "worksphere-api/pkg/errors"

// Middleware error codes
const (
	ErrUnauthorized            = "UNAUTHORIZED"
	ErrInvalidToken            = "INVALID_TOKEN"
	ErrInsufficientPermissions = "INSUFFICIENT_PERMISSIONS"
	ErrInternalError           = "INTERNAL_ERROR"
)

// Error messages
const (
	msgAuthRequired          = "authentication required"
	msgInvalidToken          = "invalid token"
	msgInsufficientPerms     = "you do not have permission to access this resource"
	msgInvalidRolesInContext = "invalid user roles in context"
)

// Pre-defined errors for middleware
var (
	ErrAuthRequired          = apperrors.New(403, ErrUnauthorized, msgAuthRequired)
	ErrInvalidRolesInContext = apperrors.New(500, ErrInternalError, msgInvalidRolesInContext)
	ErrInsufficientPerms     = apperrors.New(403, ErrInsufficientPermissions, msgInsufficientPerms)
)
