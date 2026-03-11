package response

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "worksphere-api/pkg/errors"
)

type successResponse struct {
	Data    any    `json:"data"`
	Message string `json:"message"`
}

type errorEnvelope struct {
	Error             errorResponse `json:"error"`
	RetryAfterSeconds *int          `json:"retry_after_seconds,omitempty"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Success(c *gin.Context, status int, data any, message string) {
	if message == "" {
		message = "success"
	}

	c.JSON(status, successResponse{
		Data:    data,
		Message: message,
	})
}

func Error(c *gin.Context, err error) {
	var appErr *apperrors.AppError
	if ok := apperrors.As(err, &appErr); ok {
		c.JSON(appErr.StatusCode, errorEnvelope{
			Error: errorResponse{
				Code:    appErr.Code,
				Message: appErr.Message,
			},
		})
		return
	}

	c.JSON(http.StatusInternalServerError, errorEnvelope{
		Error: errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "something went wrong",
		},
	})
}

func ErrorWithRetryAfter(c *gin.Context, status int, code, message string, retryAfterSeconds int) {
	c.Header("Retry-After", itoa(retryAfterSeconds))
	c.JSON(status, errorEnvelope{
		Error: errorResponse{
			Code:    code,
			Message: message,
		},
		RetryAfterSeconds: &retryAfterSeconds,
	})
}

func itoa(value int) string {
	return fmt.Sprintf("%d", value)
}
