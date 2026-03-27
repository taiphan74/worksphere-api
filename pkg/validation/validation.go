package validation

import (
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// IsValidEmail checks if the string is a valid email address.
func IsValidEmail(email string) bool {
	if email == "" {
		return false
	}
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

// NormalizeEmail trims whitespace and converts email to lowercase.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// IsValidStatus checks if the status is one of the allowed values.
func IsValidStatus(status string) bool {
	s := strings.ToUpper(strings.TrimSpace(status))
	switch s {
	case "ACTIVE", "INACTIVE", "SUSPENDED":
		return true
	default:
		return false
	}
}

// NormalizeStatus trims whitespace and converts status to uppercase.
func NormalizeStatus(status string) string {
	return strings.ToUpper(strings.TrimSpace(status))
}

// IsValidFullName checks if the full name is provided and not empty after trimming.
// Returns true if fullName is nil (optional behavior).
// Returns false if fullName is non-nil but empty after trimming.
func IsValidFullName(fullName *string) bool {
	if fullName == nil {
		return true
	}
	trimmed := strings.TrimSpace(*fullName)
	return trimmed != ""
}

var fieldMessages = map[string]map[string]string{
	"email": {
		"required": "Email is required",
		"email":    "Please provide a valid email address",
	},
	"password": {
		"required": "Password is required",
		"min":      "Password must be at least %s characters long",
	},
	"full_name": {
		"max":      "Full name cannot exceed %s characters",
		"required": "Full name is required",
	},
	"phone": {
		"max": "Phone number cannot exceed %s characters",
	},
	"job_title": {
		"max": "Job title cannot exceed %s characters",
	},
	"name": {
		"required": "Name is required",
		"max":      "Name cannot exceed %s characters",
	},
	"token": {
		"required": "Token is required",
	},
	"user_id": {
		"required": "User ID is required",
		"uuid":     "Please provide a valid user ID",
	},
	"role": {
		"required": "Role is required",
		"oneof":    "Please provide a valid role",
	},
	"file_name": {
		"required": "File name is required",
	},
	"content_type": {
		"required": "Content type is required",
	},
	"size": {
		"required": "File size is required",
		"gt":       "File size must be greater than 0",
	},
	"object_key": {
		"required": "Object key is required",
	},
	"current_password": {
		"required": "Current password is required",
	},
	"new_password": {
		"required": "New password is required",
		"min":      "New password must be at least %s characters long",
	},
	"confirm_new_password": {
		"required": "Please confirm your new password",
		"eqfield":  "Passwords do not match",
	},
}

var tagMessages = map[string]string{
	"required":  "This field is required",
	"email":     "Please provide a valid email address",
	"min":       "Please provide a value with minimum %s characters",
	"max":       "Please provide a value with maximum %s characters",
	"uuid":      "Please provide a valid UUID",
	"oneof":     "Please provide a valid value",
	"eqfield":   "Values do not match",
	"gt":        "Value must be greater than %s",
	"omitempty": "Please provide a valid value",
}

// HandleValidationError handles Gin binding validation errors and returns custom error messages.
func HandleValidationError(c *gin.Context, err error) {
	var ve validator.ValidationErrors
	if ok := asValidationErrors(err, &ve); !ok {
		responseError(c, "INVALID_REQUEST", "invalid request body")
		return
	}

	messages := make([]string, 0, len(ve))
	for _, e := range ve {
		msg := getErrorMessage(e)
		messages = append(messages, msg)
	}

	responseError(c, "VALIDATION_ERROR", strings.Join(messages, "; "))
}

func getErrorMessage(e validator.FieldError) string {
	field := e.Field()
	tag := e.Tag()
	param := e.Param()

	if fieldMsg, ok := fieldMessages[strings.ToLower(field)]; ok {
		if msg, ok := fieldMsg[tag]; ok {
			return formatMessage(msg, param)
		}
	}

	if msg, ok := tagMessages[tag]; ok {
		return formatMessage(msg, param)
	}

	return fmt.Sprintf("%s is invalid", strings.ToLower(field))
}

func formatMessage(msg, param string) string {
	if param == "" {
		return msg
	}
	return fmt.Sprintf(msg, param)
}

func asValidationErrors(err error, target *validator.ValidationErrors) bool {
	if err == nil {
		return false
	}

	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return false
	}

	*target = ve
	return true
}

func responseError(c *gin.Context, code, message string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}
