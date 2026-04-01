package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"worksphere-api/internal/auth/dto"
	"worksphere-api/internal/auth/service"
	"worksphere-api/internal/user"
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(ctx context.Context, req dto.RegisterRequest) (service.RegisterResult, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(service.RegisterResult), args.Error(1)
}

func (m *MockAuthService) Login(ctx context.Context, req dto.LoginRequest) (user.User, string, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(user.User), args.String(1), args.Error(2)
}

func (m *MockAuthService) LoginWithGoogle(ctx context.Context, req dto.GoogleLoginRequest) (user.User, string, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(user.User), args.String(1), args.Error(2)
}

func (m *MockAuthService) GetCurrentUser(ctx context.Context, userID uuid.UUID) (user.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(user.User), args.Error(1)
}

func (m *MockAuthService) VerifyEmail(ctx context.Context, token string) (user.User, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(user.User), args.Error(1)
}

func (m *MockAuthService) ResendVerification(ctx context.Context, email string) (service.ResendVerificationResult, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(service.ResendVerificationResult), args.Error(1)
}

func (m *MockAuthService) ForgotPassword(ctx context.Context, email string) error {
	args := m.Called(ctx, email)
	return args.Error(0)
}

func (m *MockAuthService) ResetPassword(ctx context.Context, token string, newPassword string) error {
	args := m.Called(ctx, token, newPassword)
	return args.Error(0)
}
