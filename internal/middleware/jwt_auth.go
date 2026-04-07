package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authjwt "worksphere-api/internal/auth/jwt"
	apperrors "worksphere-api/pkg/errors"
	"worksphere-api/pkg/response"
)

const CurrentUserIDKey = "current_user_id"

type AccessTokenParser interface {
	ParseAccessToken(tokenString string) (*authjwt.Claims, error)
}

func JWTAuth(tokenParser AccessTokenParser) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractAccessToken(c)
		if tokenString == "" {
			abortUnauthorized(c, "UNAUTHORIZED", "authentication token is required")
			return
		}

		claims, err := tokenParser.ParseAccessToken(tokenString)
		if err != nil {
			abortUnauthorized(c, "INVALID_TOKEN", "invalid token")
			return
		}

		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			abortUnauthorized(c, "INVALID_TOKEN", "invalid token")
			return
		}

		c.Set(CurrentUserIDKey, userID)
		c.Set("user_roles", claims.Roles)
		c.Next()
	}
}

// extractAccessToken extracts the access token from either the access_token cookie or the Authorization header
func extractAccessToken(c *gin.Context) string {
	// First, try to get token from the access_token cookie
	cookieToken, err := c.Cookie("access_token")
	if err == nil && cookieToken != "" {
		return cookieToken
	}

	// Fallback to Authorization header
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if header == "" {
		return ""
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

func GetCurrentUserID(c *gin.Context) (uuid.UUID, error) {
	value, exists := c.Get(CurrentUserIDKey)
	if !exists {
		return uuid.Nil, apperrors.New(http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	}

	userID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil, apperrors.New(http.StatusUnauthorized, "INVALID_TOKEN", "invalid token")
	}

	return userID, nil
}

func abortUnauthorized(c *gin.Context, code, message string) {
	response.Error(c, apperrors.New(http.StatusUnauthorized, code, message))
	c.Abort()
}
