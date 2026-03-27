package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/user"
	"worksphere-api/pkg/mapper"
)

type AuthUser struct {
	User         user.User
	PasswordHash string
}

type AuthRepository interface {
	CreateUserWithPassword(ctx context.Context, params db.CreateUserWithPasswordParams) (user.User, error)
	GetUserByEmail(ctx context.Context, email string) (AuthUser, error)
	GetUserByIDForAuthProfile(ctx context.Context, id uuid.UUID) (user.User, error)
	MarkUserEmailVerified(ctx context.Context, id uuid.UUID) (user.User, error)
	ResetUserPassword(ctx context.Context, id uuid.UUID, passwordHash string) (user.User, error)
}

type authRepository struct {
	queries *db.Queries
}

func NewAuthRepository(queries *db.Queries) AuthRepository {
	return &authRepository{queries: queries}
}

func (r *authRepository) CreateUserWithPassword(ctx context.Context, params db.CreateUserWithPasswordParams) (user.User, error) {
	row, err := r.queries.CreateUserWithPassword(ctx, params)
	if err != nil {
		return user.User{}, err
	}

	return toUser(
		row.ID,
		row.Email,
		row.FullName,
		row.AvatarKey,
		row.Phone,
		row.JobTitle,
		row.IsVerified,
		row.Status,
		row.PasswordChangedAt,
		row.CreatedAt,
		row.UpdatedAt,
		row.DeletedAt,
	), nil
}

func (r *authRepository) GetUserByEmail(ctx context.Context, email string) (AuthUser, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return AuthUser{}, err
	}

	return AuthUser{
		User: toUser(
			row.ID,
			row.Email,
			row.FullName,
			row.AvatarKey,
			row.Phone,
			row.JobTitle,
			row.IsVerified,
			row.Status,
			row.PasswordChangedAt,
			row.CreatedAt,
			row.UpdatedAt,
			row.DeletedAt,
		),
		PasswordHash: row.PasswordHash,
	}, nil
}

func (r *authRepository) GetUserByIDForAuthProfile(ctx context.Context, id uuid.UUID) (user.User, error) {
	row, err := r.queries.GetUserByIDForAuthProfile(ctx, id)
	if err != nil {
		return user.User{}, err
	}

	return toUser(
		row.ID,
		row.Email,
		row.FullName,
		row.AvatarKey,
		row.Phone,
		row.JobTitle,
		row.IsVerified,
		row.Status,
		row.PasswordChangedAt,
		row.CreatedAt,
		row.UpdatedAt,
		row.DeletedAt,
	), nil
}

func (r *authRepository) MarkUserEmailVerified(ctx context.Context, id uuid.UUID) (user.User, error) {
	row, err := r.queries.MarkUserEmailVerified(ctx, id)
	if err != nil {
		return user.User{}, err
	}

	return toUser(
		row.ID,
		row.Email,
		row.FullName,
		row.AvatarKey,
		row.Phone,
		row.JobTitle,
		row.IsVerified,
		row.Status,
		row.PasswordChangedAt,
		row.CreatedAt,
		row.UpdatedAt,
		row.DeletedAt,
	), nil
}

func (r *authRepository) ResetUserPassword(ctx context.Context, id uuid.UUID, passwordHash string) (user.User, error) {
	row, err := r.queries.ResetUserPassword(ctx, db.ResetUserPasswordParams{
		ID:           id,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return user.User{}, err
	}

	return toUser(
		row.ID,
		row.Email,
		row.FullName,
		row.AvatarKey,
		row.Phone,
		row.JobTitle,
		row.IsVerified,
		row.Status,
		row.PasswordChangedAt,
		row.CreatedAt,
		row.UpdatedAt,
		row.DeletedAt,
	), nil
}

func toUser(
	id uuid.UUID,
	email string,
	fullName pgtype.Text,
	avatarKey pgtype.Text,
	phone pgtype.Text,
	jobTitle pgtype.Text,
	isVerified bool,
	status string,
	passwordChangedAt pgtype.Timestamptz,
	createdAt pgtype.Timestamptz,
	updatedAt pgtype.Timestamptz,
	deletedAt pgtype.Timestamptz,
) user.User {
	return user.User{
		ID:                id.String(),
		Email:             email,
		FullName:          mapper.TextPtr(fullName),
		AvatarKey:         mapper.TextPtr(avatarKey),
		Phone:             mapper.TextPtr(phone),
		JobTitle:          mapper.TextPtr(jobTitle),
		IsVerified:        isVerified,
		Status:            status,
		PasswordChangedAt: mapper.TimestamptzPtr(passwordChangedAt),
		CreatedAt:         createdAt.Time,
		UpdatedAt:         updatedAt.Time,
		DeletedAt:         mapper.TimestamptzPtr(deletedAt),
	}
}
