package mocks

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"worksphere-api/internal/user"
	"worksphere-api/internal/user/dto"
)

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(ctx context.Context, req dto.CreateUserRequest) (user.User, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(user.User), args.Error(1)
}

func (m *MockUserService) GetUserByID(ctx context.Context, id uuid.UUID) (user.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(user.User), args.Error(1)
}

func (m *MockUserService) ListUsers(ctx context.Context, status *string, search *string) ([]user.User, error) {
	args := m.Called(ctx, status, search)
	return args.Get(0).([]user.User), args.Error(1)
}

func (m *MockUserService) UpdateUser(ctx context.Context, id uuid.UUID, req dto.UpdateUserRequest) (user.User, error) {
	args := m.Called(ctx, id, req)
	return args.Get(0).(user.User), args.Error(1)
}

func (m *MockUserService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserService) RestoreUser(ctx context.Context, id uuid.UUID) (user.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(user.User), args.Error(1)
}
