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
	GetEmailVerificationUserID(ctx context.Context, token string) (string, error)
	DeleteEmailVerificationToken(ctx context.Context, token string, userID string) error
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

	opCtx, cancel := withRedisTimeout(ctx)
	defer cancel()

	pipe := s.client.TxPipeline()
	pipe.Set(opCtx, EmailTokenKey(tokenHash), normalizedUserID, s.ttl)
	pipe.Set(opCtx, EmailUserKey(normalizedUserID), tokenHash, s.ttl)
	if _, err := pipe.Exec(opCtx); err != nil {
		return "", fmt.Errorf("generate verification token: store redis keys: %w", err)
	}

	return rawToken, nil
}

func (s *service) GetEmailVerificationUserID(ctx context.Context, token string) (string, error) {
	if s.client == nil {
		return "", ErrServiceUnavailable
	}

	tokenHash := hashToken(token)
	if tokenHash == "" {
		return "", ErrInvalidToken
	}

	opCtx, cancel := withRedisTimeout(ctx)
	defer cancel()

	userID, err := s.client.Get(opCtx, EmailTokenKey(tokenHash)).Result()
	return handleTokenLookupResult(userID, err, "get email verification user id: lookup token")
}

func (s *service) DeleteEmailVerificationToken(ctx context.Context, token string, userID string) error {
	if s.client == nil {
		return ErrServiceUnavailable
	}

	tokenHash := hashToken(token)
	normalizedUserID := strings.TrimSpace(userID)
	if tokenHash == "" || normalizedUserID == "" {
		return ErrInvalidToken
	}

	opCtx, cancel := withRedisTimeout(ctx)
	defer cancel()

	if err := s.client.Del(opCtx, EmailTokenKey(tokenHash), EmailUserKey(normalizedUserID)).Err(); err != nil {
		return fmt.Errorf("delete email verification token: %w", err)
	}

	return nil
}

func (s *service) ClearEmailVerification(ctx context.Context, userID string) error {
	if s.client == nil {
		return ErrServiceUnavailable
	}

	normalizedUserID := strings.TrimSpace(userID)
	if normalizedUserID == "" {
		return nil
	}

	opCtx, cancel := withRedisTimeout(ctx)
	defer cancel()

	tokenHash, err := s.client.Get(opCtx, EmailUserKey(normalizedUserID)).Result()
	if err := handleClearLookupErr(err, "clear email verification: lookup user token"); err != nil {
		return err
	}

	keys := []string{EmailUserKey(normalizedUserID)}
	if tokenHash != "" {
		keys = append(keys, EmailTokenKey(tokenHash))
	}

	if delErr := s.client.Del(opCtx, keys...).Err(); delErr != nil {
		return fmt.Errorf("clear email verification: delete keys: %w", delErr)
	}

	return nil
}

func withRedisTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, redisOpTimeout)
}

func handleTokenLookupResult(userID string, err error, operation string) (string, error) {
	if err == goredis.Nil {
		return "", ErrInvalidToken
	}
	if err != nil {
		return "", fmt.Errorf("%s: %w", operation, err)
	}

	return userID, nil
}

func handleClearLookupErr(err error, operation string) error {
	if err != nil && err != goredis.Nil {
		return fmt.Errorf("%s: %w", operation, err)
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
