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
	"github.com/jackc/pgx/v5/pgxpool"

	db "worksphere-api/internal/database/sqlc"
	taskrepository "worksphere-api/internal/task/repository"
	"worksphere-api/internal/workspace"
	"worksphere-api/internal/workspace/dto"
	"worksphere-api/internal/workspace/repository"
	apperrors "worksphere-api/pkg/errors"
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
	dbPool     *pgxpool.Pool
	repo       repository.WorkspaceRepository
	memberRepo repository.MemberRepository
	taskRepo   taskrepository.TaskRepository
}

func NewWorkspaceService(dbPool *pgxpool.Pool, repo repository.WorkspaceRepository, memberRepo repository.MemberRepository, taskRepo taskrepository.TaskRepository) WorkspaceService {
	return &workspaceService{dbPool: dbPool, repo: repo, memberRepo: memberRepo, taskRepo: taskRepo}
}

func (s *workspaceService) CreateWorkspace(ctx context.Context, userID uuid.UUID, req dto.CreateWorkspaceRequest) (dto.WorkspaceResponse, error) {
	slug, err := generateSlug(req.Name)
	if err != nil {
		if errors.Is(err, dto.ErrInvalidSlug) {
			return dto.WorkspaceResponse{}, workspace.ErrInvalidWorkspaceName
		}
		return dto.WorkspaceResponse{}, workspace.ErrInternalServer
	}

	// Start transaction
	tx, err := s.dbPool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return dto.WorkspaceResponse{}, workspace.ErrInternalServer
	}

	// Defer rollback - if commit fails or panic occurs, rollback will be called
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	// Use transaction-aware repositories
	txRepo := s.repo.WithTx(tx)
	txMemberRepo := s.memberRepo.WithTx(tx)
	txTaskRepo := s.taskRepo.WithTx(tx)

	exists, err := txRepo.CheckSlugExists(ctx, slug, uuid.Nil)
	if err != nil {
		_ = tx.Rollback(ctx)
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

	w, err := txRepo.CreateWorkspace(ctx, params)
	if err != nil {
		_ = tx.Rollback(ctx)
		return dto.WorkspaceResponse{}, mapRepositoryError(err)
	}

	// Auto-add the creator as OWNER member
	_, err = txMemberRepo.AddMember(ctx, db.AddWorkspaceMemberParams{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        "OWNER",
	})
	if err != nil {
		_ = tx.Rollback(ctx)
		return dto.WorkspaceResponse{}, mapRepositoryError(err)
	}

	// Tạo task mẫu trong cùng transaction để workspace mới luôn có dữ liệu khởi đầu.
	defaultTasks := []db.CreateTaskParams{
		{
			ID:          uuid.New(),
			WorkspaceID: workspaceID,
			CreatorID:   userID,
			Title:       "Set up your workspace",
			Status:      "TODO",
			Priority:    "MEDIUM",
		},
		{
			ID:          uuid.New(),
			WorkspaceID: workspaceID,
			CreatorID:   userID,
			Title:       "Invite team members",
			Status:      "TODO",
			Priority:    "MEDIUM",
		},
		{
			ID:          uuid.New(),
			WorkspaceID: workspaceID,
			CreatorID:   userID,
			Title:       "Create your first project",
			Status:      "TODO",
			Priority:    "MEDIUM",
		},
	}

	for _, params := range defaultTasks {
		if _, err := txTaskRepo.CreateTask(ctx, params); err != nil {
			_ = tx.Rollback(ctx)
			return dto.WorkspaceResponse{}, mapRepositoryError(err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return dto.WorkspaceResponse{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to commit transaction")
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
		return dto.WorkspaceResponse{}, workspace.ErrForbiddenAccess
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
		return dto.WorkspaceResponse{}, workspace.ErrForbiddenAccess
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
		return dto.WorkspaceResponse{}, workspace.ErrForbiddenAccess
	}
	if member.Role != "OWNER" {
		return dto.WorkspaceResponse{}, workspace.ErrInsufficientPermissions
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
		return workspace.ErrForbiddenAccess
	}
	if member.Role != "OWNER" {
		return workspace.ErrInsufficientPermissions
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
		return workspace.ErrWorkspaceNotFound
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return apperrors.New(http.StatusRequestTimeout, "REQUEST_TIMEOUT", "request timed out")
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return workspace.ErrSlugAlreadyExists
	}
	return workspace.ErrInternalServer
}
