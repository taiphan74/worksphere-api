package ratelimit

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "worksphere-api/pkg/errors"
	"worksphere-api/pkg/response"
)

func LoginIPMiddleware(service Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			c.Next()
			return
		}

		if service.AllowLoginIP(c.Request.Context(), c.ClientIP()) {
			c.Next()
			return
		}

		response.Error(c, apperrors.New(http.StatusTooManyRequests, "RATE_LIMITED", "too many login attempts"))
		c.Abort()
	}
}

func RegisterIPMiddleware(service Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			c.Next()
			return
		}

		if service.AllowRegisterIP(c.Request.Context(), c.ClientIP()) {
			c.Next()
			return
		}

		response.Error(c, apperrors.New(http.StatusTooManyRequests, "RATE_LIMITED", "too many registration attempts"))
		c.Abort()
	}
}
