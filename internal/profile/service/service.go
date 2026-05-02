package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/profile/dto"
	"worksphere-api/internal/profile/repository"
	"worksphere-api/internal/storage"
	"worksphere-api/internal/user"
	apperrors "worksphere-api/pkg/errors"
	"worksphere-api/pkg/utils"
)

const MaxAvatarSize = 5 * 1024 * 1024 // 5MB

type ProfileService interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (dto.ProfileResponse, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) (dto.ProfileResponse, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordRequest) error
	GetAvatarUploadURL(ctx context.Context, userID uuid.UUID, req dto.AvatarUploadURLRequest) (dto.AvatarUploadURLResponse, error)
	ConfirmAvatarUpload(ctx context.Context, userID uuid.UUID, req dto.AvatarConfirmRequest) (dto.ProfileResponse, error)
	GetAvatarViewURL(ctx context.Context, userID uuid.UUID) (dto.AvatarViewURLResponse, error)
}

type profileService struct {
	repo           repository.ProfileRepository
	storage        storage.StorageProvider
	uploadTTL      time.Duration
	viewTTL        time.Duration
}

func NewProfileService(repo repository.ProfileRepository, storage storage.StorageProvider, uploadTTL, viewTTL time.Duration) ProfileService {
	return &profileService{
		repo:      repo,
		storage:   storage,
		uploadTTL: uploadTTL,
		viewTTL:   viewTTL,
	}
}

func (s *profileService) GetProfile(ctx context.Context, userID uuid.UUID) (dto.ProfileResponse, error) {
	u, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return dto.ProfileResponse{}, mapRepositoryError(err)
	}

	return toProfileResponse(u), nil
}

func (s *profileService) UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) (dto.ProfileResponse, error) {
	params := sqlc.UpdateProfileParams{
		ID: userID,
	}

	if req.FullName != nil {
		params.UpdateFullName = true
		params.FullName = pgtype.Text{String: *req.FullName, Valid: true}
	}
	if req.Phone != nil {
		params.UpdatePhone = true
		params.Phone = pgtype.Text{String: *req.Phone, Valid: true}
	}
	if req.JobTitle != nil {
		params.UpdateJobTitle = true
		params.JobTitle = pgtype.Text{String: *req.JobTitle, Valid: true}
	}

	u, err := s.repo.UpdateProfile(ctx, params)
	if err != nil {
		return dto.ProfileResponse{}, mapRepositoryError(err)
	}

	return toProfileResponse(u), nil
}

func (s *profileService) ChangePassword(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordRequest) error {
	currentHash, err := s.repo.GetUserPasswordHash(ctx, userID)
	if err != nil {
		return mapRepositoryError(err)
	}

	if err := utils.ComparePassword(currentHash, req.CurrentPassword); err != nil {
		return apperrors.New(http.StatusBadRequest, "INVALID_PASSWORD", "current password is incorrect")
	}

	if req.NewPassword == req.CurrentPassword {
		return apperrors.New(http.StatusBadRequest, "SAME_PASSWORD", "new password cannot be the same as current password")
	}

	newHash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
	}

	if err := s.repo.ChangePassword(ctx, userID, newHash); err != nil {
		return mapRepositoryError(err)
	}

	return nil
}

func (s *profileService) GetAvatarUploadURL(ctx context.Context, userID uuid.UUID, req dto.AvatarUploadURLRequest) (dto.AvatarUploadURLResponse, error) {
	// Validate size
	if req.Size > MaxAvatarSize {
		return dto.AvatarUploadURLResponse{}, apperrors.New(http.StatusBadRequest, "FILE_TOO_LARGE", fmt.Sprintf("avatar size must not exceed %d bytes (5MB)", MaxAvatarSize))
	}

	// Validate MIME type
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
	}
	contentType := strings.ToLower(req.ContentType)
	if !allowedTypes[contentType] {
		return dto.AvatarUploadURLResponse{}, apperrors.New(http.StatusBadRequest, "INVALID_CONTENT_TYPE", "only JPEG, PNG and WEBP images are allowed")
	}

	// Generate safe object key
	// Convention: profiles/{userID}/avatar/{uuid}{ext}
	ext := filepath.Ext(req.FileName)
	if ext == "" {
		// Fallback extension based on content type
		switch strings.ToLower(req.ContentType) {
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/webp":
			ext = ".webp"
		}
	}
	
	objectKey := fmt.Sprintf("profiles/%s/avatar/%s%s", userID.String(), uuid.New().String(), ext)

	uploadURL, err := s.storage.GeneratePresignedUploadURL(ctx, objectKey, contentType, s.uploadTTL)
	if err != nil {
		return dto.AvatarUploadURLResponse{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate upload URL")
	}

	return dto.AvatarUploadURLResponse{
		ObjectKey: objectKey,
		UploadURL: uploadURL,
		Method:    "PUT",
		ExpiresIn: int(s.uploadTTL.Seconds()),
		RequiredHeaders: map[string]string{
			"Content-Type": contentType,
		},
	}, nil
}

func (s *profileService) ConfirmAvatarUpload(ctx context.Context, userID uuid.UUID, req dto.AvatarConfirmRequest) (dto.ProfileResponse, error) {
	// Normalize and security check: ensure the object key stays within the user's namespace
	cleanKey := path.Clean(req.ObjectKey)
	if cleanKey == "" || cleanKey == "." || strings.HasPrefix(cleanKey, "..") || strings.HasPrefix(cleanKey, "/") {
		return dto.ProfileResponse{}, apperrors.New(http.StatusBadRequest, "INVALID_KEY", "invalid object key format")
	}

	prefix := fmt.Sprintf("profiles/%s/avatar/", userID.String())
	if !strings.HasPrefix(cleanKey, prefix) || cleanKey == prefix {
		return dto.ProfileResponse{}, apperrors.New(http.StatusForbidden, "FORBIDDEN", "invalid object key for this user")
	}

	// Verify the object exists in R2
	exists, err := s.storage.FileExists(ctx, req.ObjectKey)
	if err != nil {
		return dto.ProfileResponse{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to verify upload")
	}
	if !exists {
		return dto.ProfileResponse{}, apperrors.New(http.StatusBadRequest, "FILE_NOT_FOUND", "file has not been uploaded yet")
	}

	// Update the avatar key in DB
	if err := s.repo.UpdateAvatarKey(ctx, userID, req.ObjectKey); err != nil {
		return dto.ProfileResponse{}, mapRepositoryError(err)
	}

	// Return updated profile
	return s.GetProfile(ctx, userID)
}

func (s *profileService) GetAvatarViewURL(ctx context.Context, userID uuid.UUID) (dto.AvatarViewURLResponse, error) {
	u, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return dto.AvatarViewURLResponse{}, mapRepositoryError(err)
	}

	if u.AvatarKey == nil || *u.AvatarKey == "" {
		return dto.AvatarViewURLResponse{}, apperrors.New(http.StatusNotFound, "AVATAR_NOT_FOUND", "user has no avatar")
	}

	viewURL, err := s.storage.GeneratePresignedDownloadURL(ctx, *u.AvatarKey, s.viewTTL)
	if err != nil {
		return dto.AvatarViewURLResponse{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate view URL")
	}

	return dto.AvatarViewURLResponse{
		ViewURL:   viewURL,
		ExpiresIn: int(s.viewTTL.Seconds()),
	}, nil
}

func toProfileResponse(u user.User) dto.ProfileResponse {
	return dto.ProfileResponse{
		ID:         u.ID,
		Email:      u.Email,
		FullName:   u.FullName,
		AvatarKey:  u.AvatarKey,
		Phone:      u.Phone,
		JobTitle:   u.JobTitle,
		IsVerified: u.IsVerified,
		Status:     u.Status,
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
	}
}

func mapRepositoryError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.New(http.StatusNotFound, "RESOURCE_NOT_FOUND", "the requested resource was not found")
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return apperrors.New(http.StatusRequestTimeout, "TIMEOUT", "request timed out")
	}
	return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "an internal database error occurred")
}
