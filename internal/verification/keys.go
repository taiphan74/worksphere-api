package verification

import "strings"

const (
	emailVerifyPrefix     = "verify:email:"
	emailVerifyUserPrefix = "verify:email:user:"
)

func EmailTokenKey(tokenHash string) string {
	return emailVerifyPrefix + strings.TrimSpace(tokenHash)
}

func EmailUserKey(userID string) string {
	return emailVerifyUserPrefix + strings.TrimSpace(userID)
}
