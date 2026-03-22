package service

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/anyascii/go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/workspace/dto"
	"worksphere-api/internal/workspace/repository"
	apperrors "worksphere-api/pkg/errors"
)

// Error messages
const (
	errWorkspaceNotFound  = "workspace not found"
	errWorkspaceForbidden = "workspace access forbidden"
	errSlugConflict       = "slug already taken"
)

type WorkspaceService interface {
	CreateWorkspace(ctx context.Context, userID uuid.UUID, req dto.CreateWorkspaceRequest) (dto.WorkspaceResponse, error)
	GetWorkspaceByID(ctx context.Context, userID, id uuid.UUID) (dto.WorkspaceResponse, error)
	GetWorkspaceBySlug(ctx context.Context, userID uuid.UUID, slug string) (dto.WorkspaceResponse, error)
	ListWorkspacesByUser(ctx context.Context, userID uuid.UUID) ([]dto.WorkspaceResponse, error)
	UpdateWorkspace(ctx context.Context, userID, id uuid.UUID, req dto.UpdateWorkspaceRequest) (dto.WorkspaceResponse, error)
	DeleteWorkspace(ctx context.Context, userID, id uuid.UUID) error
}

type workspaceService struct {
	repo       repository.WorkspaceRepository
	memberRepo repository.MemberRepository
}

func NewWorkspaceService(repo repository.WorkspaceRepository, memberRepo repository.MemberRepository) WorkspaceService {
	return &workspaceService{repo: repo, memberRepo: memberRepo}
}

func (s *workspaceService) CreateWorkspace(ctx context.Context, userID uuid.UUID, req dto.CreateWorkspaceRequest) (dto.WorkspaceResponse, error) {
	slug, err := generateSlug(req.Name)
	if err != nil {
		if errors.Is(err, dto.ErrInvalidSlug) {
			return dto.WorkspaceResponse{}, apperrors.New(http.StatusBadRequest, "INVALID_WORKSPACE_NAME", "Workspace name must contain at least one valid alphanumeric character")
		}
		return dto.WorkspaceResponse{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate workspace slug")
	}

	exists, err := s.repo.CheckSlugExists(ctx, slug, uuid.Nil)
	if err != nil {
		return dto.WorkspaceResponse{}, mapRepositoryError(err)
	}
	if exists {
		// generate unique slug
		slug = slug + "-" + strings.Split(uuid.New().String(), "-")[0]
	}

	workspaceID := uuid.New()

	params := db.CreateWorkspaceParams{
		ID:   workspaceID,
		Name: req.Name,
		Slug: slug,
	}

	w, err := s.repo.CreateWorkspace(ctx, params)
	if err != nil {
		return dto.WorkspaceResponse{}, mapRepositoryError(err)
	}

	// Auto-add the creator as OWNER member
	_, err = s.memberRepo.AddMember(ctx, db.AddWorkspaceMemberParams{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        "OWNER",
	})
	if err != nil {
		return dto.WorkspaceResponse{}, mapRepositoryError(err)
	}

	return toWorkspaceResponse(w), nil
}

func (s *workspaceService) GetWorkspaceByID(ctx context.Context, userID, id uuid.UUID) (dto.WorkspaceResponse, error) {
	w, err := s.repo.GetWorkspaceByID(ctx, id)
	if err != nil {
		return dto.WorkspaceResponse{}, mapRepositoryError(err)
	}

	// Check user is a member of this workspace
	_, err = s.memberRepo.GetMember(ctx, id, userID)
	if err != nil {
		return dto.WorkspaceResponse{}, apperrors.New(http.StatusForbidden, "FORBIDDEN", errWorkspaceForbidden)
	}

	return toWorkspaceResponse(w), nil
}

func (s *workspaceService) GetWorkspaceBySlug(ctx context.Context, userID uuid.UUID, slug string) (dto.WorkspaceResponse, error) {
	w, err := s.repo.GetWorkspaceBySlug(ctx, slug)
	if err != nil {
		return dto.WorkspaceResponse{}, mapRepositoryError(err)
	}

	// Check user is a member of this workspace
	_, err = s.memberRepo.GetMember(ctx, w.ID, userID)
	if err != nil {
		return dto.WorkspaceResponse{}, apperrors.New(http.StatusForbidden, "FORBIDDEN", errWorkspaceForbidden)
	}

	return toWorkspaceResponse(w), nil
}

func (s *workspaceService) ListWorkspacesByUser(ctx context.Context, userID uuid.UUID) ([]dto.WorkspaceResponse, error) {
	workspaces, err := s.repo.ListWorkspacesByUser(ctx, userID)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	res := make([]dto.WorkspaceResponse, len(workspaces))
	for i, w := range workspaces {
		res[i] = toWorkspaceResponse(w)
	}
	return res, nil
}

func (s *workspaceService) UpdateWorkspace(ctx context.Context, userID, id uuid.UUID, req dto.UpdateWorkspaceRequest) (dto.WorkspaceResponse, error) {
	// Check user is OWNER
	member, err := s.memberRepo.GetMember(ctx, id, userID)
	if err != nil {
		return dto.WorkspaceResponse{}, apperrors.New(http.StatusForbidden, "FORBIDDEN", errWorkspaceForbidden)
	}
	if member.Role != "OWNER" {
		return dto.WorkspaceResponse{}, apperrors.New(http.StatusForbidden, "FORBIDDEN", "only owners can update workspace")
	}

	params := db.UpdateWorkspaceParams{
		ID: id,
	}

	if req.Name != nil {
		params.UpdateName = true
		params.Name = *req.Name
	}

	updated, err := s.repo.UpdateWorkspace(ctx, params)
	if err != nil {
		return dto.WorkspaceResponse{}, mapRepositoryError(err)
	}

	return toWorkspaceResponse(updated), nil
}

func (s *workspaceService) DeleteWorkspace(ctx context.Context, userID, id uuid.UUID) error {
	// Check user is OWNER
	member, err := s.memberRepo.GetMember(ctx, id, userID)
	if err != nil {
		return apperrors.New(http.StatusForbidden, "FORBIDDEN", errWorkspaceForbidden)
	}
	if member.Role != "OWNER" {
		return apperrors.New(http.StatusForbidden, "FORBIDDEN", "only owners can delete workspace")
	}

	err = s.repo.DeleteWorkspace(ctx, id)
	if err != nil {
		return mapRepositoryError(err)
	}

	return nil
}

// Helpers

func generateSlug(s string) (string, error) {
	// Convert Unicode to ASCII (e.g., "Tiếng Việt" → "Tieng Viet")
	ascii := anyascii.Transliterate(s)

	slug := strings.ToLower(strings.TrimSpace(ascii))
	reg := regexp.MustCompile("[^a-z0-9]+")
	slug = reg.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "", dto.ErrInvalidSlug
	}
	return slug, nil
}

func toWorkspaceResponse(w db.Workspace) dto.WorkspaceResponse {
	return dto.WorkspaceResponse{
		ID:        w.ID.String(),
		Name:      w.Name,
		Slug:      w.Slug,
		CreatedAt: w.CreatedAt.Time,
		UpdatedAt: w.UpdatedAt.Time,
	}
}

func mapRepositoryError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.New(http.StatusNotFound, "NOT_FOUND", errWorkspaceNotFound)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return apperrors.New(http.StatusRequestTimeout, "TIMEOUT", "request timed out")
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperrors.New(http.StatusConflict, "CONFLICT", "a record with this value already exists")
	}
	return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "an internal database error occurred")
}
