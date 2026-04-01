package utils

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestPasswordUtils(t *testing.T) {
	t.Run("HashPassword success", func(t *testing.T) {
		password := "mysecurepassword123"
		hash, err := HashPassword(password)

		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, password, hash)
		
		// Verify using bcrypt directly to confirm it's standard
		err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		assert.NoError(t, err)
	})

	t.Run("HashPassword should produce different salts for same password", func(t *testing.T) {
		password := "samepassword"
		hash1, _ := HashPassword(password)
		hash2, _ := HashPassword(password)

		assert.NotEqual(t, hash1, hash2, "Bcrypt hashes should have different salts")
	})

	t.Run("HashPassword fails if password > 72 bytes", func(t *testing.T) {
		longPassword := "a123456789b123456789c123456789d123456789e123456789f123456789g123456789h123" // 73 chars
		hash, err := HashPassword(longPassword)

		assert.Error(t, err)
		assert.Empty(t, hash)
		assert.Contains(t, err.Error(), "password too long")
	})

	t.Run("ComparePassword success", func(t *testing.T) {
		password := "correct_pass"
		hash, _ := HashPassword(password)

		err := ComparePassword(hash, password)
		assert.NoError(t, err)
	})

	t.Run("ComparePassword failure", func(t *testing.T) {
		password := "correct_pass"
		wrongPassword := "wrong_pass"
		hash, _ := HashPassword(password)

		err := ComparePassword(hash, wrongPassword)
		assert.Error(t, err, "Should return error for incorrect password")
		assert.Equal(t, bcrypt.ErrMismatchedHashAndPassword, err)
	})
}
