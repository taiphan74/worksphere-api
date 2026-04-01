package ratelimit

import (
	"context"
	"log/slog"
	"math"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	globalBurstLimit      = 10
	globalBurstWindow     = time.Second
	globalSustainedLimit  = 60
	globalSustainedWindow = time.Minute
	loginIPLimit          = 5
	loginIPWindow         = time.Minute
	loginEmailFailLimit   = 5
	loginEmailFailWindow  = 10 * time.Minute
	registerMinuteLimit   = 3
	registerMinuteWindow  = time.Minute
	registerHourLimit     = 10
	registerHourWindow    = time.Hour
	resendEmailLimit      = 1
	resendEmailWindow     = 5 * time.Minute
	resendIPLimit         = 5
	resendIPWindow        = time.Hour
	redisOpTimeout        = time.Second
)

type Service interface {
	AllowGlobalIP(ctx context.Context, ip string) (bool, int, error)
	AllowLoginIP(ctx context.Context, ip string) bool
	AllowRegisterIP(ctx context.Context, ip string) bool
	AllowResendVerificationIP(ctx context.Context, ip string) (bool, int, error)
	AllowResendVerificationEmail(ctx context.Context, email string) (bool, int, error)
	IsEmailLocked(ctx context.Context, email string) bool
	IncrementFailedLogin(ctx context.Context, email string)
	ClearFailedLogin(ctx context.Context, email string)
}

type service struct {
	client *goredis.Client
	logger *slog.Logger
}

func NewService(client *goredis.Client, logger *slog.Logger) Service {
	return &service{
		client: client,
		logger: logger,
	}
}

func (s *service) AllowLoginIP(ctx context.Context, ip string) bool {
	if s.client == nil {
		return true
	}

	count, err := s.incrementWithTTL(ctx, LoginIPKey(ip), loginIPWindow)
	if err != nil {
		s.warn("login IP rate limit check failed, allowing request", "ip", ip, "error", err)
		return true
	}

	return count <= loginIPLimit
}

func (s *service) AllowGlobalIP(ctx context.Context, ip string) (bool, int, error) {
	if s.client == nil {
		return true, 0, nil
	}

	allowed, retryAfterSeconds, err := s.allowWithTTL(ctx, GlobalBurstKey(ip), globalBurstLimit, globalBurstWindow)
	if err != nil {
		s.warn("global burst rate limit check failed, allowing request", "ip", ip, "error", err)
		return true, 0, nil
	}
	if !allowed {
		return false, retryAfterSeconds, nil
	}

	allowed, retryAfterSeconds, err = s.allowWithTTL(ctx, GlobalSustainedKey(ip), globalSustainedLimit, globalSustainedWindow)
	if err != nil {
		s.warn("global sustained rate limit check failed, allowing request", "ip", ip, "error", err)
		return true, 0, nil
	}

	return allowed, retryAfterSeconds, nil
}

func (s *service) AllowRegisterIP(ctx context.Context, ip string) bool {
	if s.client == nil {
		return true
	}

	minuteCount, err := s.incrementWithTTL(ctx, RegisterMinuteKey(ip), registerMinuteWindow)
	if err != nil {
		s.warn("register minute rate limit check failed, allowing request", "ip", ip, "error", err)
		return true
	}

	hourCount, err := s.incrementWithTTL(ctx, RegisterHourKey(ip), registerHourWindow)
	if err != nil {
		s.warn("register hour rate limit check failed, allowing request", "ip", ip, "error", err)
		return true
	}

	return minuteCount <= registerMinuteLimit && hourCount <= registerHourLimit
}

func (s *service) AllowResendVerificationIP(ctx context.Context, ip string) (bool, int, error) {
	if s.client == nil {
		return true, 0, nil
	}

	allowed, retryAfterSeconds, err := s.allowWithTTL(ctx, ResendVerificationIPKey(ip), resendIPLimit, resendIPWindow)
	if err != nil {
		s.warn("resend verification IP rate limit check failed, allowing request", "ip", ip, "error", err)
		return true, 0, nil
	}

	return allowed, retryAfterSeconds, nil
}

func (s *service) AllowResendVerificationEmail(ctx context.Context, email string) (bool, int, error) {
	if s.client == nil {
		return true, 0, nil
	}

	allowed, retryAfterSeconds, err := s.allowWithTTL(ctx, ResendVerificationEmailKey(email), resendEmailLimit, resendEmailWindow)
	if err != nil {
		s.warn("resend verification email rate limit check failed, allowing request", "email", email, "error", err)
		return true, 0, nil
	}

	return allowed, retryAfterSeconds, nil
}

func (s *service) IsEmailLocked(ctx context.Context, email string) bool {
	if s.client == nil {
		return false
	}

	opCtx, cancel := context.WithTimeout(ctx, redisOpTimeout)
	defer cancel()

	count, err := s.client.Get(opCtx, LoginEmailKey(email)).Int()
	if err == nil {
		return count >= loginEmailFailLimit
	}

	if err == goredis.Nil {
		return false
	}

	s.warn("email lock check failed, allowing request", "email", email, "error", err)
	return false
}

func (s *service) IncrementFailedLogin(ctx context.Context, email string) {
	if s.client == nil {
		return
	}

	if _, err := s.incrementWithTTL(ctx, LoginEmailKey(email), loginEmailFailWindow); err != nil {
		s.warn("failed to increment email login counter", "email", email, "error", err)
	}
}

func (s *service) ClearFailedLogin(ctx context.Context, email string) {
	if s.client == nil {
		return
	}

	opCtx, cancel := context.WithTimeout(ctx, redisOpTimeout)
	defer cancel()

	if err := s.client.Del(opCtx, LoginEmailKey(email)).Err(); err != nil {
		s.warn("failed to clear email login counter", "email", email, "error", err)
	}
}

func (s *service) incrementWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	opCtx, cancel := context.WithTimeout(ctx, redisOpTimeout)
	defer cancel()

	count, err := s.client.Incr(opCtx, key).Result()
	if err != nil {
		return 0, err
	}

	if count == 1 {
		if err := s.client.Expire(opCtx, key, ttl).Err(); err != nil {
			return 0, err
		}
	}

	return count, nil
}

func (s *service) allowWithTTL(ctx context.Context, key string, limit int64, window time.Duration) (bool, int, error) {
	count, err := s.incrementWithTTL(ctx, key, window)
	if err != nil {
		return false, 0, err
	}

	if count <= limit {
		return true, 0, nil
	}

	retryAfter, err := s.retryAfterSeconds(ctx, key, window)
	if err != nil {
		return false, 0, err
	}

	return false, retryAfter, nil
}

func (s *service) retryAfterSeconds(ctx context.Context, key string, fallback time.Duration) (int, error) {
	opCtx, cancel := context.WithTimeout(ctx, redisOpTimeout)
	defer cancel()

	ttl, err := s.client.TTL(opCtx, key).Result()
	if err != nil {
		return 0, err
	}

	if ttl <= 0 {
		ttl = fallback
	}

	return int(math.Ceil(ttl.Seconds())), nil
}

func (s *service) warn(msg string, args ...any) {
	if s.logger != nil {
		s.logger.Warn(msg, args...)
	}
}
