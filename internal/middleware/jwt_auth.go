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
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if header == "" {
			abortUnauthorized(c, "UNAUTHORIZED", "authorization header is required")
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			abortUnauthorized(c, "INVALID_TOKEN", "invalid authorization header")
			return
		}

		claims, err := tokenParser.ParseAccessToken(strings.TrimSpace(parts[1]))
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
