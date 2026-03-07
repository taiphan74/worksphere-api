package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "worksphere-api/internal/database/sqlc"
)

type UserRepository interface {
	CreateUser(ctx context.Context, params db.CreateUserParams) (db.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	ListUsers(ctx context.Context) ([]db.User, error)
	UpdateUser(ctx context.Context, params db.UpdateUserParams) (db.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) (uuid.UUID, error)
}

type userRepository struct {
	queries *db.Queries
}

func NewUserRepository(queries *db.Queries) UserRepository {
	return &userRepository{queries: queries}
}

func (r *userRepository) CreateUser(ctx context.Context, params db.CreateUserParams) (db.User, error) {
	row, err := r.queries.CreateUser(ctx, params)
	if err != nil {
		return db.User{}, err
	}

	return toUser(row.ID, row.Email, row.FullName, row.PasswordHash, row.CreatedAt, row.UpdatedAt), nil
}

func (r *userRepository) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	row, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		return db.User{}, err
	}

	return toUser(row.ID, row.Email, row.FullName, row.PasswordHash, row.CreatedAt, row.UpdatedAt), nil
}

func (r *userRepository) ListUsers(ctx context.Context) ([]db.User, error) {
	rows, err := r.queries.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	users := make([]db.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, toUser(row.ID, row.Email, row.FullName, row.PasswordHash, row.CreatedAt, row.UpdatedAt))
	}

	return users, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, params db.UpdateUserParams) (db.User, error) {
	row, err := r.queries.UpdateUser(ctx, params)
	if err != nil {
		return db.User{}, err
	}

	return toUser(row.ID, row.Email, row.FullName, row.PasswordHash, row.CreatedAt, row.UpdatedAt), nil
}

func (r *userRepository) DeleteUser(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	return r.queries.DeleteUser(ctx, id)
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
