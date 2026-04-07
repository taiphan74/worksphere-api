package repository

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type RefreshTokenRepository interface {
	Set(ctx context.Context, userID string, jtiHash string, ttl time.Duration) error
	Get(ctx context.Context, userID string) (string, error)
	Delete(ctx context.Context, userID string) error
}

type refreshTokenRepository struct {
	client *goredis.Client
}

func NewRefreshTokenRepository(client *goredis.Client) RefreshTokenRepository {
	return &refreshTokenRepository{client: client}
}

func (r *refreshTokenRepository) Set(ctx context.Context, userID string, jtiHash string, ttl time.Duration) error {
	key := fmt.Sprintf("refresh:%s", userID)
	return r.client.Set(ctx, key, jtiHash, ttl).Err()
}

func (r *refreshTokenRepository) Get(ctx context.Context, userID string) (string, error) {
	key := fmt.Sprintf("refresh:%s", userID)
	return r.client.Get(ctx, key).Result()
}

func (r *refreshTokenRepository) Delete(ctx context.Context, userID string) error {
	key := fmt.Sprintf("refresh:%s", userID)
	return r.client.Del(ctx, key).Err()
}
