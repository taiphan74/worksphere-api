package utils

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plain text password using bcrypt with default cost.
// It returns an error if the password is longer than 72 bytes.
func HashPassword(password string) (string, error) {
	if len(password) > 72 {
		return "", errors.New("password too long: bcrypt only supports up to 72 bytes")
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ComparePassword compares a hashed password with a plain text password.
func ComparePassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
