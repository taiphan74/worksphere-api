package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/task"
	"worksphere-api/internal/task/dto"
	"worksphere-api/internal/task/repository"
	workspacerepository "worksphere-api/internal/workspace/repository"
	apperrors "worksphere-api/pkg/errors"
	pkgmapper "worksphere-api/pkg/mapper"
)

var validStatuses = map[string]bool{
	"TODO":        true,
	"IN_PROGRESS": true,
	"IN_REVIEW":   true,
	"DONE":        true,
}

var validPriorities = map[string]bool{
	"LOW":    true,
	"MEDIUM": true,
	"HIGH":   true,
	"URGENT": true,
}

type TaskService interface {
	CreateTask(ctx context.Context, requesterID, workspaceID uuid.UUID, req dto.CreateTaskRequest) (dto.TaskResponse, error)
	ListTasks(ctx context.Context, requesterID, workspaceID uuid.UUID, req dto.ListTasksRequest) ([]dto.TaskResponse, error)
	GetTaskByID(ctx context.Context, requesterID, workspaceID, taskID uuid.UUID) (dto.TaskResponse, error)
	UpdateTask(ctx context.Context, requesterID, workspaceID, taskID uuid.UUID, req dto.UpdateTaskRequest) (dto.TaskResponse, error)
	DeleteTask(ctx context.Context, requesterID, workspaceID, taskID uuid.UUID) error
}

type taskService struct {
	repo       repository.TaskRepository
	memberRepo workspacerepository.MemberRepository
}

func NewTaskService(repo repository.TaskRepository, memberRepo workspacerepository.MemberRepository) TaskService {
	return &taskService{repo: repo, memberRepo: memberRepo}
}

func (s *taskService) CreateTask(ctx context.Context, requesterID, workspaceID uuid.UUID, req dto.CreateTaskRequest) (dto.TaskResponse, error) {
	if err := s.ensureWorkspaceMember(ctx, workspaceID, requesterID); err != nil {
		return dto.TaskResponse{}, err
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return dto.TaskResponse{}, task.ErrTaskTitleRequired
	}
	if len(title) > 255 {
		return dto.TaskResponse{}, task.ErrTaskTitleTooLong
	}

	status := "TODO"
	if req.Status != "" {
		status = normalizeUpper(req.Status)
		if !validStatuses[status] {
			return dto.TaskResponse{}, task.ErrInvalidTaskStatus
		}
	}

	priority := "MEDIUM"
	if req.Priority != "" {
		priority = normalizeUpper(req.Priority)
		if !validPriorities[priority] {
			return dto.TaskResponse{}, task.ErrInvalidTaskPriority
		}
	}

	if req.SprintPoints != nil && *req.SprintPoints < 0 {
		return dto.TaskResponse{}, task.ErrInvalidSprintPoints
	}

	assigneeID, err := s.resolveOptionalMember(ctx, workspaceID, req.AssigneeID)
	if err != nil {
		return dto.TaskResponse{}, err
	}

	parentID, err := s.resolveOptionalParent(ctx, workspaceID, uuid.Nil, req.ParentID)
	if err != nil {
		return dto.TaskResponse{}, err
	}

	created, err := s.repo.CreateTask(ctx, db.CreateTaskParams{
		ID:           uuid.New(),
		WorkspaceID:  workspaceID,
		CreatorID:    requesterID,
		AssigneeID:   uuidToPgtype(assigneeID),
		Title:        title,
		Description:  pkgmapper.StringToText(trimStringPtr(req.Description)),
		Status:       status,
		Priority:     priority,
		DueAt:        pkgmapper.TimeToTimestamptz(req.DueAt),
		SprintPoints: int32ToPgtype(req.SprintPoints),
		ParentID:     uuidToPgtype(parentID),
	})
	if err != nil {
		return dto.TaskResponse{}, mapRepositoryError(err)
	}

	return toTaskResponse(created), nil
}

func (s *taskService) ListTasks(ctx context.Context, requesterID, workspaceID uuid.UUID, req dto.ListTasksRequest) ([]dto.TaskResponse, error) {
	if err := s.ensureWorkspaceMember(ctx, workspaceID, requesterID); err != nil {
		return nil, err
	}

	params := db.ListTasksByWorkspaceParams{
		WorkspaceID:    workspaceID,
		IncludeDeleted: req.IncludeDeleted,
	}

	if req.Status != "" {
		status := normalizeUpper(req.Status)
		if !validStatuses[status] {
			return nil, task.ErrInvalidTaskStatus
		}
		params.Status = pgtype.Text{String: status, Valid: true}
	}

	if req.AssigneeID != "" {
		parsedID, err := uuid.Parse(strings.TrimSpace(req.AssigneeID))
		if err != nil {
			return nil, task.ErrInvalidAssigneeID
		}
		params.AssigneeID = pgtype.UUID{Bytes: parsedID, Valid: true}
	}

	if req.ParentID != "" {
		parsedID, err := uuid.Parse(strings.TrimSpace(req.ParentID))
		if err != nil {
			return nil, task.ErrInvalidParentTaskID
		}
		params.ParentID = pgtype.UUID{Bytes: parsedID, Valid: true}
	}

	if req.DueFrom != "" {
		dueFrom, err := time.Parse(time.RFC3339, strings.TrimSpace(req.DueFrom))
		if err != nil {
			return nil, apperrors.New(http.StatusBadRequest, "INVALID_DUE_FROM", "due_from must be a valid RFC3339 timestamp")
		}
		params.DueFrom = pgtype.Timestamptz{Time: dueFrom, Valid: true}
	}

	if req.DueTo != "" {
		dueTo, err := time.Parse(time.RFC3339, strings.TrimSpace(req.DueTo))
		if err != nil {
			return nil, apperrors.New(http.StatusBadRequest, "INVALID_DUE_TO", "due_to must be a valid RFC3339 timestamp")
		}
		params.DueTo = pgtype.Timestamptz{Time: dueTo, Valid: true}
	}

	tasks, err := s.repo.ListTasksByWorkspace(ctx, params)
	if err != nil {
		return nil, mapRepositoryError(err)
	}

	items := make([]dto.TaskResponse, len(tasks))
	for i, item := range tasks {
		items[i] = toTaskResponse(item)
	}
	return items, nil
}

func (s *taskService) GetTaskByID(ctx context.Context, requesterID, workspaceID, taskID uuid.UUID) (dto.TaskResponse, error) {
	if err := s.ensureWorkspaceMember(ctx, workspaceID, requesterID); err != nil {
		return dto.TaskResponse{}, err
	}

	item, err := s.repo.GetTaskByID(ctx, workspaceID, taskID)
	if err != nil {
		return dto.TaskResponse{}, mapRepositoryError(err)
	}

	return toTaskResponse(item), nil
}

func (s *taskService) UpdateTask(ctx context.Context, requesterID, workspaceID, taskID uuid.UUID, req dto.UpdateTaskRequest) (dto.TaskResponse, error) {
	if err := s.ensureWorkspaceMember(ctx, workspaceID, requesterID); err != nil {
		return dto.TaskResponse{}, err
	}
	if !req.HasChanges() {
		return dto.TaskResponse{}, apperrors.New(http.StatusBadRequest, "NO_TASK_CHANGES", "no task changes provided")
	}

	params := db.UpdateTaskParams{
		ID:          taskID,
		WorkspaceID: workspaceID,
	}

	if req.HasTitle {
		if req.Title == nil {
			return dto.TaskResponse{}, task.ErrTaskTitleRequired
		}
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return dto.TaskResponse{}, task.ErrTaskTitleRequired
		}
		if len(title) > 255 {
			return dto.TaskResponse{}, task.ErrTaskTitleTooLong
		}
		params.UpdateTitle = true
		params.Title = title
	}

	if req.HasDescription {
		params.UpdateDescription = true
		params.Description = pkgmapper.StringToText(trimStringPtr(req.Description))
	}

	if req.HasStatus {
		if req.Status == nil {
			return dto.TaskResponse{}, task.ErrInvalidTaskStatus
		}
		status := normalizeUpper(*req.Status)
		if !validStatuses[status] {
			return dto.TaskResponse{}, task.ErrInvalidTaskStatus
		}
		params.UpdateStatus = true
		params.Status = status
	}

	if req.HasPriority {
		if req.Priority == nil {
			return dto.TaskResponse{}, task.ErrInvalidTaskPriority
		}
		priority := normalizeUpper(*req.Priority)
		if !validPriorities[priority] {
			return dto.TaskResponse{}, task.ErrInvalidTaskPriority
		}
		params.UpdatePriority = true
		params.Priority = priority
	}

	if req.HasDueAt {
		params.UpdateDueAt = true
		params.DueAt = pkgmapper.TimeToTimestamptz(req.DueAt)
	}

	if req.HasSprintPoints {
		params.UpdateSprintPoints = true
		if req.SprintPoints != nil {
			if *req.SprintPoints < 0 {
				return dto.TaskResponse{}, task.ErrInvalidSprintPoints
			}
			params.SprintPoints = pgtype.Int4{Int32: *req.SprintPoints, Valid: true}
		}
	}

	if req.HasAssigneeID {
		params.UpdateAssigneeID = true
		assigneeID, err := s.resolveOptionalMember(ctx, workspaceID, req.AssigneeID)
		if err != nil {
			return dto.TaskResponse{}, err
		}
		params.AssigneeID = uuidToPgtype(assigneeID)
	}

	if req.HasParentID {
		params.UpdateParentID = true
		parentID, err := s.resolveOptionalParent(ctx, workspaceID, taskID, req.ParentID)
		if err != nil {
			return dto.TaskResponse{}, err
		}
		params.ParentID = uuidToPgtype(parentID)
	}

	updated, err := s.repo.UpdateTask(ctx, params)
	if err != nil {
		return dto.TaskResponse{}, mapRepositoryError(err)
	}

	return toTaskResponse(updated), nil
}

func (s *taskService) DeleteTask(ctx context.Context, requesterID, workspaceID, taskID uuid.UUID) error {
	if err := s.ensureWorkspaceMember(ctx, workspaceID, requesterID); err != nil {
		return err
	}

	if _, err := s.repo.GetTaskByID(ctx, workspaceID, taskID); err != nil {
		return mapRepositoryError(err)
	}

	if err := s.repo.SoftDeleteTask(ctx, workspaceID, taskID); err != nil {
		return mapRepositoryError(err)
	}

	return nil
}

func (s *taskService) ensureWorkspaceMember(ctx context.Context, workspaceID, userID uuid.UUID) error {
	if _, err := s.memberRepo.GetMember(ctx, workspaceID, userID); err != nil {
		return task.ErrForbiddenAccess
	}
	return nil
}

func (s *taskService) resolveOptionalMember(ctx context.Context, workspaceID uuid.UUID, raw *string) (uuid.UUID, error) {
	if raw == nil {
		return uuid.Nil, nil
	}

	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return uuid.Nil, task.ErrInvalidAssigneeID
	}

	assigneeID, err := uuid.Parse(trimmed)
	if err != nil {
		return uuid.Nil, task.ErrInvalidAssigneeID
	}

	if _, err := s.memberRepo.GetMember(ctx, workspaceID, assigneeID); err != nil {
		return uuid.Nil, task.ErrAssigneeNotWorkspaceMember
	}

	return assigneeID, nil
}

func (s *taskService) resolveOptionalParent(ctx context.Context, workspaceID, taskID uuid.UUID, raw *string) (uuid.UUID, error) {
	if raw == nil {
		return uuid.Nil, nil
	}

	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return uuid.Nil, task.ErrInvalidParentTaskID
	}

	parentID, err := uuid.Parse(trimmed)
	if err != nil {
		return uuid.Nil, task.ErrInvalidParentTaskID
	}
	if taskID != uuid.Nil && parentID == taskID {
		return uuid.Nil, task.ErrParentTaskSelfReference
	}

	if _, err := s.repo.GetTaskByID(ctx, workspaceID, parentID); err != nil {
		return uuid.Nil, task.ErrParentTaskNotFound
	}

	return parentID, nil
}

func toTaskResponse(item db.Task) dto.TaskResponse {
	return dto.TaskResponse{
		ID:           item.ID.String(),
		WorkspaceID:  item.WorkspaceID.String(),
		CreatorID:    item.CreatorID.String(),
		AssigneeID:   pgUUIDToStringPtr(item.AssigneeID),
		Title:        item.Title,
		Description:  pkgmapper.TextPtr(item.Description),
		Status:       item.Status,
		Priority:     item.Priority,
		DueAt:        pkgmapper.TimestamptzPtr(item.DueAt),
		SprintPoints: pgInt4ToInt32Ptr(item.SprintPoints),
		ParentID:     pgUUIDToStringPtr(item.ParentID),
		CreatedAt:    item.CreatedAt.Time,
		UpdatedAt:    item.UpdatedAt.Time,
		DeletedAt:    pkgmapper.TimestamptzPtr(item.DeletedAt),
	}
}

func mapRepositoryError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return task.ErrTaskNotFound
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return apperrors.New(http.StatusRequestTimeout, "REQUEST_TIMEOUT", "request timed out")
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			return task.ErrTaskNotFound
		case "23514":
			return apperrors.New(http.StatusBadRequest, "INVALID_TASK_DATA", "task data violates database constraints")
		}
	}

	return task.ErrInternalServer
}

func normalizeUpper(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func trimStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func uuidToPgtype(value uuid.UUID) pgtype.UUID {
	if value == uuid.Nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: value, Valid: true}
}

func int32ToPgtype(value *int32) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *value, Valid: true}
}

func pgUUIDToStringPtr(value pgtype.UUID) *string {
	if !value.Valid {
		return nil
	}
	result := uuid.UUID(value.Bytes).String()
	return &result
}

func pgInt4ToInt32Ptr(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	result := value.Int32
	return &result
}
