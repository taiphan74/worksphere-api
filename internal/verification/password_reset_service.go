package verification

import (
	"context"
	"fmt"
	"strings"
)

type PasswordResetService interface {
	GenerateResetToken(ctx context.Context, userID string) (string, error)
	GetPasswordResetUserID(ctx context.Context, token string) (string, error)
	DeletePasswordResetToken(ctx context.Context, token string, userID string) error
	ClearResetToken(ctx context.Context, userID string) error
}

type passwordResetService struct {
	client *service
}

func NewPasswordResetService(client Service) PasswordResetService {
	svc, ok := client.(*service)
	if !ok {
		return &passwordResetService{}
	}

	return &passwordResetService{client: svc}
}

func (s *passwordResetService) GenerateResetToken(ctx context.Context, userID string) (string, error) {
	if s.client == nil || s.client.client == nil {
		return "", ErrServiceUnavailable
	}

	normalizedUserID := strings.TrimSpace(userID)
	if normalizedUserID == "" {
		return "", fmt.Errorf("generate reset token: user id is required")
	}

	if err := s.ClearResetToken(ctx, normalizedUserID); err != nil {
		return "", err
	}

	rawToken, tokenHash, err := generateTokenPair()
	if err != nil {
		return "", err
	}

	opCtx, cancel := withRedisTimeout(ctx)
	defer cancel()

	pipe := s.client.client.TxPipeline()
	pipe.Set(opCtx, ResetPasswordTokenKey(tokenHash), normalizedUserID, s.client.ttl)
	pipe.Set(opCtx, ResetPasswordUserKey(normalizedUserID), tokenHash, s.client.ttl)
	if _, err := pipe.Exec(opCtx); err != nil {
		return "", fmt.Errorf("generate reset token: store redis keys: %w", err)
	}

	return rawToken, nil
}

func (s *passwordResetService) GetPasswordResetUserID(ctx context.Context, token string) (string, error) {
	if s.client == nil || s.client.client == nil {
		return "", ErrServiceUnavailable
	}

	tokenHash := hashToken(token)
	if tokenHash == "" {
		return "", ErrInvalidToken
	}

	opCtx, cancel := withRedisTimeout(ctx)
	defer cancel()

	userID, err := s.client.client.Get(opCtx, ResetPasswordTokenKey(tokenHash)).Result()
	return handleTokenLookupResult(userID, err, "get password reset user id: lookup token")
}

func (s *passwordResetService) DeletePasswordResetToken(ctx context.Context, token string, userID string) error {
	if s.client == nil || s.client.client == nil {
		return ErrServiceUnavailable
	}

	tokenHash := hashToken(token)
	normalizedUserID := strings.TrimSpace(userID)
	if tokenHash == "" || normalizedUserID == "" {
		return ErrInvalidToken
	}

	opCtx, cancel := withRedisTimeout(ctx)
	defer cancel()

	if err := s.client.client.Del(opCtx, ResetPasswordTokenKey(tokenHash), ResetPasswordUserKey(normalizedUserID)).Err(); err != nil {
		return fmt.Errorf("delete password reset token: %w", err)
	}

	return nil
}

func (s *passwordResetService) ClearResetToken(ctx context.Context, userID string) error {
	if s.client == nil || s.client.client == nil {
		return ErrServiceUnavailable
	}

	normalizedUserID := strings.TrimSpace(userID)
	if normalizedUserID == "" {
		return nil
	}

	opCtx, cancel := withRedisTimeout(ctx)
	defer cancel()

	tokenHash, err := s.client.client.Get(opCtx, ResetPasswordUserKey(normalizedUserID)).Result()
	if err != nil {
		return handleClearLookupErr(err, "clear reset token: lookup user token")
	}

	keys := []string{ResetPasswordUserKey(normalizedUserID)}
	if tokenHash != "" {
		keys = append(keys, ResetPasswordTokenKey(tokenHash))
	}

	if delErr := s.client.client.Del(opCtx, keys...).Err(); delErr != nil {
		return fmt.Errorf("clear reset token: delete keys: %w", delErr)
	}

	return nil
}
