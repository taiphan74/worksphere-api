package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/user"
)

type AuthUser struct {
	User         user.User
	PasswordHash string
}

type AuthRepository interface {
	CreateUserWithPassword(ctx context.Context, params db.CreateUserWithPasswordParams) (user.User, error)
	GetUserByEmail(ctx context.Context, email string) (AuthUser, error)
	GetUserByIDForAuthProfile(ctx context.Context, id uuid.UUID) (user.User, error)
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
		row.Username,
		row.AvatarUrl,
		row.Phone,
		row.JobTitle,
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
			row.Username,
			row.AvatarUrl,
			row.Phone,
			row.JobTitle,
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
		row.Username,
		row.AvatarUrl,
		row.Phone,
		row.JobTitle,
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
	username pgtype.Text,
	avatarURL pgtype.Text,
	phone pgtype.Text,
	jobTitle pgtype.Text,
	status string,
	passwordChangedAt pgtype.Timestamptz,
	createdAt pgtype.Timestamptz,
	updatedAt pgtype.Timestamptz,
	deletedAt pgtype.Timestamptz,
) user.User {
	return user.User{
		ID:                id.String(),
		Email:             email,
		FullName:          textPtr(fullName),
		Username:          textPtr(username),
		AvatarURL:         textPtr(avatarURL),
		Phone:             textPtr(phone),
		JobTitle:          textPtr(jobTitle),
		Status:            status,
		PasswordChangedAt: timestamptzPtr(passwordChangedAt),
		CreatedAt:         createdAt.Time,
		UpdatedAt:         updatedAt.Time,
		DeletedAt:         timestamptzPtr(deletedAt),
	}
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}

	result := value.String
	return &result
}

func timestamptzPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	result := value.Time
	return &result
}
