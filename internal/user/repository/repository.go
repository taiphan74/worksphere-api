package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/user"
)

type UserRepository interface {
	CreateUser(ctx context.Context, params db.CreateUserParams) (user.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (user.User, error)
	ListUsers(ctx context.Context) ([]user.User, error)
	UpdateUser(ctx context.Context, params db.UpdateUserParams) (user.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) (uuid.UUID, error)
}

type userRepository struct {
	queries *db.Queries
}

func NewUserRepository(queries *db.Queries) UserRepository {
	return &userRepository{queries: queries}
}

func (r *userRepository) CreateUser(ctx context.Context, params db.CreateUserParams) (user.User, error) {
	row, err := r.queries.CreateUser(ctx, params)
	if err != nil {
		return user.User{}, err
	}

	return toUser(row.ID, row.Email, row.FullName, row.CreatedAt.Time, row.UpdatedAt.Time), nil
}

func (r *userRepository) GetUserByID(ctx context.Context, id uuid.UUID) (user.User, error) {
	row, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		return user.User{}, err
	}

	return toUser(row.ID, row.Email, row.FullName, row.CreatedAt.Time, row.UpdatedAt.Time), nil
}

func (r *userRepository) ListUsers(ctx context.Context) ([]user.User, error) {
	rows, err := r.queries.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	users := make([]user.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, toUser(row.ID, row.Email, row.FullName, row.CreatedAt.Time, row.UpdatedAt.Time))
	}

	return users, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, params db.UpdateUserParams) (user.User, error) {
	row, err := r.queries.UpdateUser(ctx, params)
	if err != nil {
		return user.User{}, err
	}

	return toUser(row.ID, row.Email, row.FullName, row.CreatedAt.Time, row.UpdatedAt.Time), nil
}

func (r *userRepository) DeleteUser(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	return r.queries.DeleteUser(ctx, id)
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
