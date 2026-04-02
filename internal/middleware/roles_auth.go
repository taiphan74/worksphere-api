package middleware

import (
	"slices"

	"github.com/gin-gonic/gin"

	"worksphere-api/pkg/response"
)

// RequireRoles creates a middleware that checks if the user has at least one of the required roles.
// The roles are extracted from the JWT claims that were set during authentication.
//
// Usage:
//
//	// Only ADMIN can access
//	router.Use(RequireRoles("ADMIN"))
//
//	// ADMIN or SUPER_ADMIN can access
//	router.Use(RequireRoles("ADMIN", "SUPER_ADMIN"))
func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get roles from JWT claims (set by JWTAuth middleware)
		rolesValue, exists := c.Get("user_roles")
		if !exists {
			abortWithError(c, ErrAuthRequired)
			return
		}

		userRoles, ok := rolesValue.([]string)
		if !ok {
			abortWithError(c, ErrInvalidRolesInContext)
			return
		}

		// Check if user has at least one of the required roles
		hasRole := false
		for _, role := range userRoles {
			if slices.Contains(roles, role) {
				hasRole = true
				break
			}
		}

		if !hasRole {
			abortWithError(c, ErrInsufficientPerms)
			return
		}

		c.Next()
	}
}

// RequireRole creates a middleware that checks if the user has a specific role.
// This is a convenience wrapper around RequireRoles for single role checks.
//
// Usage:
//
//	// Only ADMIN can access
//	router.Use(RequireRole("ADMIN"))
func RequireRole(role string) gin.HandlerFunc {
	return RequireRoles(role)
}

func abortWithError(c *gin.Context, err error) {
	response.Error(c, err)
	c.Abort()
}
