package service

import (
	"context"
	stderrors "errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/user"
	"worksphere-api/internal/user/dto"
	"worksphere-api/internal/user/repository"
	apperrors "worksphere-api/pkg/errors"
	"worksphere-api/pkg/utils"
	"worksphere-api/pkg/validation"
)

type UserService interface {
	CreateUser(ctx context.Context, req dto.CreateUserRequest) (user.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (user.User, error)
	ListUsers(ctx context.Context, status *string, search *string) ([]user.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req dto.UpdateUserRequest) (user.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	RestoreUser(ctx context.Context, id uuid.UUID) (user.User, error)
}

type userService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) CreateUser(ctx context.Context, req dto.CreateUserRequest) (user.User, error) {
	input, err := normalizeCreateInput(req)
	if err != nil {
		return user.User{}, err
	}

	passwordHash, err := utils.HashPassword(input.Password)
	if err != nil {
		return user.User{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
	}

	result, err := s.repo.CreateUser(ctx, db.CreateUserParams{
		ID:           uuid.New(),
		Email:        input.Email,
		PasswordHash: passwordHash,
		FullName:     input.FullName,
		IsVerified:   false,
		Status:       input.Status,
	})
	if err != nil {
		return user.User{}, mapRepositoryError(err, "failed to create user")
	}

	return result, nil
}

func (s *userService) GetUserByID(ctx context.Context, id uuid.UUID) (user.User, error) {
	result, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return user.User{}, mapRepositoryError(err, "failed to get user")
	}

	return result, nil
}

func (s *userService) ListUsers(ctx context.Context, status *string, search *string) ([]user.User, error) {
	params := db.ListUsersParams{}

	if status != nil {
		if !validation.IsValidStatus(*status) {
			return nil, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "invalid status")
		}
		params.Status = pgtype.Text{String: validation.NormalizeStatus(*status), Valid: true}
	}
	if search != nil {
		params.Search = pgtype.Text{String: strings.TrimSpace(*search), Valid: true}
	}

	users, err := s.repo.ListUsers(ctx, params)
	if err != nil {
		return nil, mapRepositoryError(err, "failed to list users")
	}

	return users, nil
}

func (s *userService) UpdateUser(ctx context.Context, id uuid.UUID, req dto.UpdateUserRequest) (user.User, error) {
	params := db.UpdateUserParams{ID: id}

	if req.FullName != nil {
		if !validation.IsValidFullName(req.FullName) {
			return user.User{}, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "full_name cannot be empty")
		}
		params.SetFullName = true
		params.FullName = pgtype.Text{String: strings.TrimSpace(*req.FullName), Valid: true}
	}

	if req.Password != nil {
		password := strings.TrimSpace(*req.Password)
		if len(password) < 8 {
			return user.User{}, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "password must be at least 8 characters")
		}

		passwordHash, err := utils.HashPassword(password)
		if err != nil {
			return user.User{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
		}
		params.SetPasswordHash = true
		params.PasswordHash = passwordHash
	}

	result, err := s.repo.UpdateUser(ctx, params)
	if err != nil {
		return user.User{}, mapRepositoryError(err, "failed to update user")
	}

	return result, nil
}

func (s *userService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	_, err := s.repo.DeleteUser(ctx, id)
	if err != nil {
		return mapRepositoryError(err, "failed to delete user")
	}

	return nil
}

func (s *userService) RestoreUser(ctx context.Context, id uuid.UUID) (user.User, error) {
	result, err := s.repo.RestoreUser(ctx, id)
	if err != nil {
		return user.User{}, mapRepositoryError(err, "failed to restore user")
	}

	return result, nil
}

type createInput struct {
	Email    string
	Password string
	FullName pgtype.Text
	Status   string
}

func normalizeCreateInput(req dto.CreateUserRequest) (createInput, error) {
	email := validation.NormalizeEmail(req.Email)
	if email == "" {
		return createInput{}, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "email is required")
	}
	if !validation.IsValidEmail(email) {
		return createInput{}, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "invalid email")
	}

	var fullName pgtype.Text
	if req.FullName != nil {
		if !validation.IsValidFullName(req.FullName) {
			return createInput{}, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "full_name cannot be empty")
		}
		fullName = pgtype.Text{String: strings.TrimSpace(*req.FullName), Valid: true}
	}

	password := strings.TrimSpace(req.Password)
	if len(password) < 8 {
		return createInput{}, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "password must be at least 8 characters")
	}

	return createInput{
		Email:    email,
		Password: password,
		FullName: fullName,
		Status:   "ACTIVE",
	}, nil
}

func mapRepositoryError(err error, fallbackMessage string) error {
	if stderrors.Is(err, pgx.ErrNoRows) {
		return user.ErrUserNotFound
	}

	var pgErr *pgconn.PgError
	if apperrors.As(err, &pgErr) && pgErr.Code == "23505" {
		return user.ErrEmailAlreadyExists
	}

	return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", fallbackMessage)
}
