package validation

import (
	"net/mail"
	"strings"
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
