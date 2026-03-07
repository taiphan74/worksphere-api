package service

import (
	"context"
	stderrors "errors"
	"net/http"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/user"
	"worksphere-api/internal/user/dto"
	"worksphere-api/internal/user/repository"
	apperrors "worksphere-api/pkg/errors"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) CreateUser(ctx context.Context, req dto.CreateUserRequest) (user.User, error) {
	email, fullName, err := normalizeCreateRequest(req)
	if err != nil {
		return user.User{}, err
	}

	record, err := s.repo.CreateUser(ctx, db.CreateUserParams{
		ID:       uuid.New(),
		Email:    email,
		FullName: fullName,
	})
	if err != nil {
		return user.User{}, mapRepositoryError(err, "failed to create user")
	}

	return record, nil
}

func (s *UserService) GetUserByID(ctx context.Context, id string) (user.User, error) {
	userID, err := parseUserID(id)
	if err != nil {
		return user.User{}, err
	}

	record, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return user.User{}, apperrors.New(http.StatusNotFound, "USER_NOT_FOUND", "user not found")
		}

		return user.User{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get user")
	}

	return record, nil
}

func (s *UserService) ListUsers(ctx context.Context) ([]user.User, error) {
	records, err := s.repo.ListUsers(ctx)
	if err != nil {
		return nil, mapRepositoryError(err, "failed to list users")
	}

	return records, nil
}

func (s *UserService) UpdateUser(ctx context.Context, id string, req dto.UpdateUserRequest) (user.User, error) {
	userID, err := parseUserID(id)
	if err != nil {
		return user.User{}, err
	}

	currentUser, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return user.User{}, mapRepositoryError(err, "failed to get user")
	}

	email, fullName, err := normalizeUpdateRequest(req, currentUser)
	if err != nil {
		return user.User{}, err
	}

	record, err := s.repo.UpdateUser(ctx, db.UpdateUserParams{
		ID:       userID,
		Email:    email,
		FullName: fullName,
	})
	if err != nil {
		return user.User{}, mapRepositoryError(err, "failed to update user")
	}

	return record, nil
}

func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	userID, err := parseUserID(id)
	if err != nil {
		return err
	}

	if _, err := s.repo.DeleteUser(ctx, userID); err != nil {
		return mapRepositoryError(err, "failed to delete user")
	}

	return nil
}

func normalizeCreateRequest(req dto.CreateUserRequest) (string, string, error) {
	email := normalizeEmail(req.Email)
	fullName := strings.TrimSpace(req.FullName)

	if err := validateUserInput(email, fullName); err != nil {
		return "", "", err
	}

	return email, fullName, nil
}

func normalizeUpdateRequest(req dto.UpdateUserRequest, currentUser user.User) (string, string, error) {
	email := currentUser.Email
	fullName := strings.TrimSpace(currentUser.FullName)

	if req.Email != nil {
		email = normalizeEmail(*req.Email)
	}

	if req.FullName != nil {
		fullName = strings.TrimSpace(*req.FullName)
	}

	if err := validateUserInput(email, fullName); err != nil {
		return "", "", err
	}

	return email, fullName, nil
}

func validateUserInput(email, fullName string) error {
	if email == "" || fullName == "" {
		return apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "email and full_name are required")
	}

	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "invalid email")
	}

	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func mapRepositoryError(err error, fallbackMessage string) error {
	if stderrors.Is(err, pgx.ErrNoRows) {
		return apperrors.New(http.StatusNotFound, "USER_NOT_FOUND", "user not found")
	}

	var pgErr *pgconn.PgError
	if apperrors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperrors.New(http.StatusConflict, "EMAIL_ALREADY_EXISTS", "email already exists")
	}

	return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", fallbackMessage)
}

func parseUserID(id string) (uuid.UUID, error) {
	userID, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, apperrors.New(http.StatusBadRequest, "INVALID_USER_ID", "invalid user id")
	}

	return userID, nil
}
