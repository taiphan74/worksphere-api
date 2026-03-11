package verification

import "strings"

const (
	resetPasswordPrefix     = "reset:password:"
	resetPasswordUserPrefix = "reset:password:user:"
)

func ResetPasswordTokenKey(tokenHash string) string {
	return resetPasswordPrefix + strings.TrimSpace(tokenHash)
}

func ResetPasswordUserKey(userID string) string {
	return resetPasswordUserPrefix + strings.TrimSpace(userID)
}
