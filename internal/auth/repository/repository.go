package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

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

	return toUser(row.ID, row.Email, row.FullName, row.CreatedAt.Time, row.UpdatedAt.Time), nil
}

func (r *authRepository) GetUserByEmail(ctx context.Context, email string) (AuthUser, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return AuthUser{}, err
	}

	return AuthUser{
		User:         toUser(row.ID, row.Email, row.FullName, row.CreatedAt.Time, row.UpdatedAt.Time),
		PasswordHash: row.PasswordHash,
	}, nil
}

func (r *authRepository) GetUserByIDForAuthProfile(ctx context.Context, id uuid.UUID) (user.User, error) {
	row, err := r.queries.GetUserByIDForAuthProfile(ctx, id)
	if err != nil {
		return user.User{}, err
	}

	return toUser(row.ID, row.Email, row.FullName, row.CreatedAt.Time, row.UpdatedAt.Time), nil
}

func toUser(
	id uuid.UUID,
	email string,
	fullName string,
	createdAt time.Time,
	updatedAt time.Time,
) user.User {
	return user.User{
		ID:        id.String(),
		Email:     email,
		FullName:  fullName,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}
