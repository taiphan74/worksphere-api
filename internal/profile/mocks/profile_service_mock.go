package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"worksphere-api/internal/profile/dto"
)

type MockProfileService struct {
	mock.Mock
}

func (m *MockProfileService) GetProfile(ctx context.Context, userID uuid.UUID) (dto.ProfileResponse, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(dto.ProfileResponse), args.Error(1)
}

func (m *MockProfileService) UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) (dto.ProfileResponse, error) {
	args := m.Called(ctx, userID, req)
	return args.Get(0).(dto.ProfileResponse), args.Error(1)
}

func (m *MockProfileService) ChangePassword(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordRequest) error {
	args := m.Called(ctx, userID, req)
	return args.Error(0)
}

func (m *MockProfileService) GetAvatarUploadURL(ctx context.Context, userID uuid.UUID, req dto.AvatarUploadURLRequest) (dto.AvatarUploadURLResponse, error) {
	args := m.Called(ctx, userID, req)
	return args.Get(0).(dto.AvatarUploadURLResponse), args.Error(1)
}

func (m *MockProfileService) ConfirmAvatarUpload(ctx context.Context, userID uuid.UUID, req dto.AvatarConfirmRequest) (dto.ProfileResponse, error) {
	args := m.Called(ctx, userID, req)
	return args.Get(0).(dto.ProfileResponse), args.Error(1)
}

func (m *MockProfileService) GetAvatarViewURL(ctx context.Context, userID uuid.UUID) (dto.AvatarViewURLResponse, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(dto.AvatarViewURLResponse), args.Error(1)
}
