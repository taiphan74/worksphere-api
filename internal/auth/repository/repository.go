package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "worksphere-api/internal/database/sqlc"
)

type AuthRepository interface {
	CreateUserWithPassword(ctx context.Context, params db.CreateUserWithPasswordParams) (db.User, error)
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	GetUserByIDForAuthProfile(ctx context.Context, id uuid.UUID) (db.User, error)
}

type authRepository struct {
	queries *db.Queries
}

func NewAuthRepository(queries *db.Queries) AuthRepository {
	return &authRepository{queries: queries}
}

func (r *authRepository) CreateUserWithPassword(ctx context.Context, params db.CreateUserWithPasswordParams) (db.User, error) {
	row, err := r.queries.CreateUserWithPassword(ctx, params)
	if err != nil {
		return db.User{}, err
	}

	return toUser(row.ID, row.Email, row.FullName, row.PasswordHash, row.CreatedAt, row.UpdatedAt), nil
}

func (r *authRepository) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return db.User{}, err
	}

	return toUser(row.ID, row.Email, row.FullName, row.PasswordHash, row.CreatedAt, row.UpdatedAt), nil
}

func (r *authRepository) GetUserByIDForAuthProfile(ctx context.Context, id uuid.UUID) (db.User, error) {
	row, err := r.queries.GetUserByIDForAuthProfile(ctx, id)
	if err != nil {
		return db.User{}, err
	}

	return toUser(row.ID, row.Email, row.FullName, row.PasswordHash, row.CreatedAt, row.UpdatedAt), nil
}

func toUser(
	id uuid.UUID,
	email string,
	fullName string,
	passwordHash string,
	createdAt pgtype.Timestamp,
	updatedAt pgtype.Timestamp,
) db.User {
	return db.User{
		ID:           id,
		Email:        email,
		FullName:     fullName,
		PasswordHash: passwordHash,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}
