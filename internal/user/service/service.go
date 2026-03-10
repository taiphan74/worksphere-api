package service

import (
	"context"
	stderrors "errors"
	"net/http"
	"net/mail"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/user"
	"worksphere-api/internal/user/dto"
	"worksphere-api/internal/user/repository"
	apperrors "worksphere-api/pkg/errors"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) CreateUser(ctx context.Context, req dto.CreateUserRequest) (user.User, error) {
	input, err := normalizeCreateInput(req)
	if err != nil {
		return user.User{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return user.User{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
	}

	record, err := s.repo.CreateUser(ctx, db.CreateUserParams{
		ID:           uuid.New(),
		Email:        input.Email,
		PasswordHash: string(passwordHash),
		FullName:     input.FullName,
		Username:     input.Username,
		AvatarUrl:    input.AvatarURL,
		Phone:        input.Phone,
		JobTitle:     input.JobTitle,
		Status:       input.Status,
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
		return user.User{}, mapRepositoryError(err, "failed to get user")
	}

	return record, nil
}

func (s *UserService) ListUsers(ctx context.Context, req dto.ListUsersRequest) ([]user.User, error) {
	params, err := normalizeListInput(req)
	if err != nil {
		return nil, err
	}

	records, err := s.repo.ListUsers(ctx, params)
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

	params, err := normalizeUpdateInput(userID, req)
	if err != nil {
		return user.User{}, err
	}

	record, err := s.repo.UpdateUser(ctx, params)
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

func (s *UserService) RestoreUser(ctx context.Context, id string) (user.User, error) {
	userID, err := parseUserID(id)
	if err != nil {
		return user.User{}, err
	}

	record, err := s.repo.RestoreUser(ctx, userID)
	if err != nil {
		return user.User{}, mapRepositoryError(err, "failed to restore user")
	}

	return record, nil
}

type createInput struct {
	Email     string
	Password  string
	FullName  pgtype.Text
	Username  pgtype.Text
	AvatarURL pgtype.Text
	Phone     pgtype.Text
	JobTitle  pgtype.Text
	Status    string
}

func normalizeCreateInput(req dto.CreateUserRequest) (createInput, error) {
	email, err := normalizeRequiredEmail(req.Email)
	if err != nil {
		return createInput{}, err
	}

	// full_name is optional
	var fullName pgtype.Text
	if req.FullName != nil {
		fullName = pgtype.Text{String: strings.TrimSpace(*req.FullName), Valid: true}
	}

	password := strings.TrimSpace(req.Password)
	if len(password) < 8 {
		return createInput{}, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "password must be at least 8 characters")
	}

	username, err := normalizeOptionalUsername(req.Username)
	if err != nil {
		return createInput{}, err
	}

	avatarURL, err := normalizeOptionalURL(req.AvatarURL)
	if err != nil {
		return createInput{}, err
	}

	phone := normalizeOptionalText(req.Phone, 20)
	jobTitle := normalizeOptionalText(req.JobTitle, 100)

	return createInput{
		Email:     email,
		Password:  password,
		FullName:  fullName,
		Username:  username,
		AvatarURL: avatarURL,
		Phone:     phone,
		JobTitle:  jobTitle,
		Status:    "ACTIVE",
	}, nil
}

func normalizeListInput(req dto.ListUsersRequest) (db.ListUsersParams, error) {
	params := db.ListUsersParams{}

	status := strings.TrimSpace(strings.ToUpper(req.Status))
	if status != "" {
		if err := validateStatus(status); err != nil {
			return db.ListUsersParams{}, err
		}
		params.Status = pgtype.Text{String: status, Valid: true}
	}

	search := strings.TrimSpace(req.Search)
	if search != "" {
		params.Search = pgtype.Text{String: search, Valid: true}
	}

	return params, nil
}

func normalizeUpdateInput(userID uuid.UUID, req dto.UpdateUserRequest) (db.UpdateUserParams, error) {
	params := db.UpdateUserParams{ID: userID}

	if req.Email != nil {
		email, err := normalizeRequiredEmail(*req.Email)
		if err != nil {
			return db.UpdateUserParams{}, err
		}

		params.SetEmail = true
		params.Email = email
	}

	if req.FullName != nil {
		fullName := strings.TrimSpace(*req.FullName)
		params.SetFullName = true
		params.FullName = pgtype.Text{String: fullName, Valid: true}
	}

	if req.AvatarURL != nil {
		avatarURL, err := normalizeOptionalURL(req.AvatarURL)
		if err != nil {
			return db.UpdateUserParams{}, err
		}

		params.SetAvatarUrl = true
		params.AvatarUrl = avatarURL
	}

	if req.Phone != nil {
		params.SetPhone = true
		params.Phone = normalizeOptionalText(req.Phone, 20)
	}

	if req.JobTitle != nil {
		params.SetJobTitle = true
		params.JobTitle = normalizeOptionalText(req.JobTitle, 100)
	}

	if req.Status != nil {
		status := strings.TrimSpace(strings.ToUpper(*req.Status))
		if err := validateStatus(status); err != nil {
			return db.UpdateUserParams{}, err
		}

		params.SetStatus = true
		params.Status = status
	}

	if req.Password != nil {
		password := strings.TrimSpace(*req.Password)
		if len(password) < 8 {
			return db.UpdateUserParams{}, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "password must be at least 8 characters")
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return db.UpdateUserParams{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
		}

		params.SetPasswordHash = true
		params.PasswordHash = string(hashedPassword)
	}

	return params, nil
}

func normalizeRequiredEmail(email string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(email))
	if value == "" {
		return "", apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "email is required")
	}

	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value {
		return "", apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "invalid email")
	}

	return value, nil
}

func normalizeOptionalUsername(username *string) (pgtype.Text, error) {
	if username == nil {
		return pgtype.Text{}, nil
	}

	value := strings.TrimSpace(*username)
	if value == "" {
		return pgtype.Text{}, nil
	}

	if len(value) < 3 || len(value) > 50 || !usernamePattern.MatchString(value) {
		return pgtype.Text{}, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "invalid username")
	}

	return pgtype.Text{String: strings.ToLower(value), Valid: true}, nil
}

func normalizeOptionalURL(value *string) (pgtype.Text, error) {
	if value == nil {
		return pgtype.Text{}, nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return pgtype.Text{}, nil
	}

	if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
		return pgtype.Text{}, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "invalid avatar_url")
	}

	return pgtype.Text{String: trimmed, Valid: true}, nil
}

func normalizeOptionalText(value *string, maxLen int) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return pgtype.Text{}
	}

	if len(trimmed) > maxLen {
		trimmed = trimmed[:maxLen]
	}

	return pgtype.Text{String: trimmed, Valid: true}
}

func validateStatus(status string) error {
	switch status {
	case "ACTIVE", "INACTIVE", "SUSPENDED":
		return nil
	default:
		return apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "invalid status")
	}
}

func mapRepositoryError(err error, fallbackMessage string) error {
	if stderrors.Is(err, pgx.ErrNoRows) {
		return apperrors.New(http.StatusNotFound, "USER_NOT_FOUND", "user not found")
	}

	var pgErr *pgconn.PgError
	if apperrors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "users_email_active_unique_idx":
			return apperrors.New(http.StatusConflict, "EMAIL_ALREADY_EXISTS", "email already exists")
		case "users_username_active_unique_idx":
			return apperrors.New(http.StatusConflict, "USERNAME_ALREADY_EXISTS", "username already exists")
		default:
			if strings.Contains(pgErr.Message, "email") {
				return apperrors.New(http.StatusConflict, "EMAIL_ALREADY_EXISTS", "email already exists")
			}
			if strings.Contains(pgErr.Message, "username") {
				return apperrors.New(http.StatusConflict, "USERNAME_ALREADY_EXISTS", "username already exists")
			}
		}
	}

	return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", fallbackMessage)
}

func parseUserID(id string) (uuid.UUID, error) {
	userID, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "invalid user id")
	}

	return userID, nil
}
