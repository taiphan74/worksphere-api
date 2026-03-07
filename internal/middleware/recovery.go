package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	apperrors "worksphere-api/pkg/errors"
	"worksphere-api/pkg/response"
)

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		requestID, _ := c.Get(RequestIDKey)

		logger.Error("panic recovered",
			slog.Any("panic", recovered),
			slog.String("request_id", toString(requestID)),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.String("stack_trace", string(debug.Stack())),
		)

		appErr := apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "something went wrong")
		response.Error(c, appErr)
	})
}
