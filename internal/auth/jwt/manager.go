package jwt

import (
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"worksphere-api/internal/config"
)

const (
	AccessTokenType  = "access"
	RefreshTokenType = "refresh"
)

type Claims struct {
	Email string   `json:"email"`
	Type  string   `json:"type"`
	Roles []string `json:"roles,omitempty"`
	jwtlib.RegisteredClaims
}

type Manager struct {
	secret           []byte
	expiresIn        time.Duration
	refreshExpiresIn time.Duration
	signMethod       jwtlib.SigningMethod
}

func NewManager(cfg config.JWTConfig) *Manager {
	return &Manager{
		secret:           []byte(cfg.Secret),
		expiresIn:        time.Duration(cfg.ExpiresInMinutes) * time.Minute,
		refreshExpiresIn: time.Duration(cfg.RefreshExpiresInDays) * 24 * time.Hour,
		signMethod:       jwtlib.SigningMethodHS256,
	}
}

func (m *Manager) GenerateAccessToken(userID uuid.UUID, email string, roles []string) (string, error) {
	now := time.Now()

	claims := Claims{
		Email: email,
		Type:  AccessTokenType,
		Roles: roles,
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwtlib.NewNumericDate(now),
			ExpiresAt: jwtlib.NewNumericDate(now.Add(m.expiresIn)),
		},
	}

	token := jwtlib.NewWithClaims(m.signMethod, claims)

	signedToken, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign jwt token: %w", err)
	}

	return signedToken, nil
}

func (m *Manager) GenerateRefreshToken(userID uuid.UUID, email string) (string, error) {
	now := time.Now()

	claims := Claims{
		Email: email,
		Type:  RefreshTokenType,
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwtlib.NewNumericDate(now),
			ExpiresAt: jwtlib.NewNumericDate(now.Add(m.refreshExpiresIn)),
		},
	}

	token := jwtlib.NewWithClaims(m.signMethod, claims)

	signedToken, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign refresh jwt token: %w", err)
	}

	return signedToken, nil
}

func (m *Manager) ParseAccessToken(tokenString string) (*Claims, error) {
	claims, err := m.ParseToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.Type != AccessTokenType {
		return nil, fmt.Errorf("invalid token type")
	}

	return claims, nil
}

func (m *Manager) ParseRefreshToken(tokenString string) (*Claims, error) {
	claims, err := m.ParseToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.Type != RefreshTokenType {
		return nil, fmt.Errorf("invalid token type")
	}

	return claims, nil
}

func (m *Manager) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwtlib.ParseWithClaims(tokenString, &Claims{}, func(token *jwtlib.Token) (any, error) {
		if token.Method.Alg() != m.signMethod.Alg() {
			return nil, fmt.Errorf("unexpected signing method")
		}

		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	if claims.Subject == "" || claims.Email == "" {
		return nil, fmt.Errorf("missing required token claims")
	}

	return claims, nil
}
