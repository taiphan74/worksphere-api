package repository

import (
	"context"

	"github.com/google/uuid"

	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/user"
)

type UserRepository interface {
	CreateUser(ctx context.Context, params db.CreateUserParams) (user.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (user.User, error)
	ListUsers(ctx context.Context, params db.ListUsersParams) ([]user.User, error)
	UpdateUser(ctx context.Context, params db.UpdateUserParams) (user.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) (uuid.UUID, error)
	RestoreUser(ctx context.Context, id uuid.UUID) (user.User, error)
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

	return toUser(
		row.ID,
		row.Email,
		row.FullName,
		row.IsVerified,
		row.Status,
		row.PasswordChangedAt,
		row.CreatedAt,
		row.UpdatedAt,
		row.DeletedAt,
	), nil
}

func (r *userRepository) GetUserByID(ctx context.Context, id uuid.UUID) (user.User, error) {
	row, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		return user.User{}, err
	}

	return toUser(
		row.ID,
		row.Email,
		row.FullName,
		row.IsVerified,
		row.Status,
		row.PasswordChangedAt,
		row.CreatedAt,
		row.UpdatedAt,
		row.DeletedAt,
	), nil
}

func (r *userRepository) ListUsers(ctx context.Context, params db.ListUsersParams) ([]user.User, error) {
	rows, err := r.queries.ListUsers(ctx, params)
	if err != nil {
		return nil, err
	}

	users := make([]user.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, toUser(
			row.ID,
			row.Email,
			row.FullName,
			row.IsVerified,
			row.Status,
			row.PasswordChangedAt,
			row.CreatedAt,
			row.UpdatedAt,
			row.DeletedAt,
		))
	}

	return users, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, params db.UpdateUserParams) (user.User, error) {
	row, err := r.queries.UpdateUser(ctx, params)
	if err != nil {
		return user.User{}, err
	}

	return toUser(
		row.ID,
		row.Email,
		row.FullName,
		row.IsVerified,
		row.Status,
		row.PasswordChangedAt,
		row.CreatedAt,
		row.UpdatedAt,
		row.DeletedAt,
	), nil
}

func (r *userRepository) DeleteUser(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	return r.queries.DeleteUser(ctx, id)
}

func (r *userRepository) RestoreUser(ctx context.Context, id uuid.UUID) (user.User, error) {
	row, err := r.queries.RestoreUser(ctx, id)
	if err != nil {
		return user.User{}, err
	}

	return toUser(
		row.ID,
		row.Email,
		row.FullName,
		row.IsVerified,
		row.Status,
		row.PasswordChangedAt,
		row.CreatedAt,
		row.UpdatedAt,
		row.DeletedAt,
	), nil
}

