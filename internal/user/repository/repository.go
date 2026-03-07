package repository

import (
	"context"

	"github.com/google/uuid"

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
	return r.queries.CreateUser(ctx, params)
}

func (r *userRepository) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	return r.queries.GetUserByID(ctx, id)
}

func (r *userRepository) ListUsers(ctx context.Context) ([]db.User, error) {
	return r.queries.ListUsers(ctx)
}

func (r *userRepository) UpdateUser(ctx context.Context, params db.UpdateUserParams) (db.User, error) {
	return r.queries.UpdateUser(ctx, params)
}

func (r *userRepository) DeleteUser(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	return r.queries.DeleteUser(ctx, id)
}
