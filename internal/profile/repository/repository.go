package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/user"
)

type ProfileRepository interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (user.User, error)
	UpdateProfile(ctx context.Context, params db.UpdateProfileParams) (user.User, error)
	UpdateAvatarKey(ctx context.Context, userID uuid.UUID, avatarKey string) error
	GetUserPasswordHash(ctx context.Context, userID uuid.UUID) (string, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, hashedPassword string) error
}

type profileRepository struct {
	queries *db.Queries
}

func NewProfileRepository(queries *db.Queries) ProfileRepository {
	return &profileRepository{queries: queries}
}

func (r *profileRepository) GetProfile(ctx context.Context, userID uuid.UUID) (user.User, error) {
	row, err := r.queries.GetProfile(ctx, userID)
	if err != nil {
		return user.User{}, err
	}

	return user.User{
		ID:         row.ID.String(),
		Email:      row.Email,
		FullName:   textPtr(row.FullName),
		AvatarKey:  textPtr(row.AvatarKey),
		Phone:      textPtr(row.Phone),
		JobTitle:   textPtr(row.JobTitle),
		IsVerified: row.IsVerified,
		Status:     row.Status,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}, nil
}

func (r *profileRepository) UpdateProfile(ctx context.Context, arg db.UpdateProfileParams) (user.User, error) {
	row, err := r.queries.UpdateProfile(ctx, arg)
	if err != nil {
		return user.User{}, err
	}

	return user.User{
		ID:         row.ID.String(),
		Email:      row.Email,
		FullName:   textPtr(row.FullName),
		AvatarKey:  textPtr(row.AvatarKey),
		Phone:      textPtr(row.Phone),
		JobTitle:   textPtr(row.JobTitle),
		IsVerified: row.IsVerified,
		Status:     row.Status,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}, nil
}

func (r *profileRepository) UpdateAvatarKey(ctx context.Context, userID uuid.UUID, avatarKey string) error {
	return r.queries.UpdateAvatarKey(ctx, db.UpdateAvatarKeyParams{
		ID:        userID,
		AvatarKey: pgtype.Text{String: avatarKey, Valid: avatarKey != ""},
	})
}

func (r *profileRepository) GetUserPasswordHash(ctx context.Context, userID uuid.UUID) (string, error) {
	return r.queries.GetUserPasswordHash(ctx, userID)
}

func (r *profileRepository) ChangePassword(ctx context.Context, userID uuid.UUID, hashedPassword string) error {
	return r.queries.ChangePassword(ctx, db.ChangePasswordParams{
		ID:           userID,
		PasswordHash: hashedPassword,
	})
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}
