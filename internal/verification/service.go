package verification

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var (
	ErrServiceUnavailable = errors.New("verification service unavailable")
	ErrInvalidToken       = errors.New("verification token invalid or expired")
)

const redisOpTimeout = 2 * time.Second

type Service interface {
	GenerateEmailVerificationToken(ctx context.Context, userID string) (string, error)
	VerifyEmailToken(ctx context.Context, token string) (string, error)
	ClearEmailVerification(ctx context.Context, userID string) error
}

type service struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewService(client *goredis.Client, ttl time.Duration) Service {
	return &service{
		client: client,
		ttl:    ttl,
	}
}

func (s *service) GenerateEmailVerificationToken(ctx context.Context, userID string) (string, error) {
	if s.client == nil {
		return "", ErrServiceUnavailable
	}

	normalizedUserID := strings.TrimSpace(userID)
	if normalizedUserID == "" {
		return "", fmt.Errorf("generate verification token: user id is required")
	}

	if err := s.ClearEmailVerification(ctx, normalizedUserID); err != nil {
		return "", err
	}

	rawToken, tokenHash, err := generateTokenPair()
	if err != nil {
		return "", err
	}

	opCtx, cancel := context.WithTimeout(ctx, redisOpTimeout)
	defer cancel()

	pipe := s.client.TxPipeline()
	pipe.Set(opCtx, EmailTokenKey(tokenHash), normalizedUserID, s.ttl)
	pipe.Set(opCtx, EmailUserKey(normalizedUserID), tokenHash, s.ttl)
	if _, err := pipe.Exec(opCtx); err != nil {
		return "", fmt.Errorf("generate verification token: store redis keys: %w", err)
	}

	return rawToken, nil
}

func (s *service) VerifyEmailToken(ctx context.Context, token string) (string, error) {
	if s.client == nil {
		return "", ErrServiceUnavailable
	}

	tokenHash := hashToken(token)
	if tokenHash == "" {
		return "", ErrInvalidToken
	}

	opCtx, cancel := context.WithTimeout(ctx, redisOpTimeout)
	defer cancel()

	userID, err := s.client.GetDel(opCtx, EmailTokenKey(tokenHash)).Result()
	if err == goredis.Nil {
		return "", ErrInvalidToken
	}
	if err != nil {
		return "", fmt.Errorf("verify email token: lookup token: %w", err)
	}

	if delErr := s.client.Del(opCtx, EmailUserKey(userID)).Err(); delErr != nil {
		return "", fmt.Errorf("verify email token: delete user key: %w", delErr)
	}

	return userID, nil
}

func (s *service) ClearEmailVerification(ctx context.Context, userID string) error {
	if s.client == nil {
		return ErrServiceUnavailable
	}

	normalizedUserID := strings.TrimSpace(userID)
	if normalizedUserID == "" {
		return nil
	}

	opCtx, cancel := context.WithTimeout(ctx, redisOpTimeout)
	defer cancel()

	tokenHash, err := s.client.Get(opCtx, EmailUserKey(normalizedUserID)).Result()
	if err != nil && err != goredis.Nil {
		return fmt.Errorf("clear email verification: lookup user token: %w", err)
	}

	keys := []string{EmailUserKey(normalizedUserID)}
	if err == nil && tokenHash != "" {
		keys = append(keys, EmailTokenKey(tokenHash))
	}

	if delErr := s.client.Del(opCtx, keys...).Err(); delErr != nil {
		return fmt.Errorf("clear email verification: delete keys: %w", delErr)
	}

	return nil
}

func generateTokenPair() (string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate verification token: %w", err)
	}

	rawToken := base64.RawURLEncoding.EncodeToString(buf)
	return rawToken, hashToken(rawToken), nil
}

func hashToken(token string) string {
	normalized := strings.TrimSpace(token)
	if normalized == "" {
		return ""
	}

	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
