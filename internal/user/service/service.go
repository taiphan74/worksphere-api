package service

import (
	"context"
	stderrors "errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/user/dto"
	"worksphere-api/internal/user/repository"
	apperrors "worksphere-api/pkg/errors"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) CreateUser(ctx context.Context, req dto.CreateUserRequest) (db.User, error) {
	user, err := s.repo.CreateUser(ctx, db.CreateUserParams{
		ID:       uuid.New(),
		Email:    req.Email,
		FullName: req.FullName,
	})
	if err != nil {
		return db.User{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create user")
	}

	return user, nil
}

func (s *UserService) GetUserByID(ctx context.Context, id string) (db.User, error) {
	userID, err := parseUserID(id)
	if err != nil {
		return db.User{}, err
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return db.User{}, apperrors.New(http.StatusNotFound, "USER_NOT_FOUND", "user not found")
		}

		return db.User{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get user")
	}

	return user, nil
}

func (s *UserService) ListUsers(ctx context.Context) ([]db.User, error) {
	users, err := s.repo.ListUsers(ctx)
	if err != nil {
		return nil, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list users")
	}

	return users, nil
}

func (s *UserService) UpdateUser(ctx context.Context, id string, req dto.UpdateUserRequest) (db.User, error) {
	userID, err := parseUserID(id)
	if err != nil {
		return db.User{}, err
	}

	user, err := s.repo.UpdateUser(ctx, db.UpdateUserParams{
		ID:       userID,
		Email:    req.Email,
		FullName: req.FullName,
	})
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return db.User{}, apperrors.New(http.StatusNotFound, "USER_NOT_FOUND", "user not found")
		}

		return db.User{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update user")
	}

	return user, nil
}

func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	userID, err := parseUserID(id)
	if err != nil {
		return err
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return apperrors.New(http.StatusNotFound, "USER_NOT_FOUND", "user not found")
		}

		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get user")
	}

	if err := s.repo.DeleteUser(ctx, user.ID); err != nil {
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete user")
	}

	return nil
}

func parseUserID(id string) (uuid.UUID, error) {
	userID, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, apperrors.New(http.StatusBadRequest, "INVALID_USER_ID", "invalid user id")
	}

	return userID, nil
}
