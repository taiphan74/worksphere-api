package repository

import (
	"context"

	"github.com/google/uuid"

	db "worksphere-api/internal/database/sqlc"
)

type TaskRepository interface {
	CreateTask(ctx context.Context, params db.CreateTaskParams) (db.Task, error)
	GetTaskByID(ctx context.Context, workspaceID, taskID uuid.UUID) (db.Task, error)
	ListTasksByWorkspace(ctx context.Context, params db.ListTasksByWorkspaceParams) ([]db.Task, error)
	UpdateTask(ctx context.Context, params db.UpdateTaskParams) (db.Task, error)
	SoftDeleteTask(ctx context.Context, workspaceID, taskID uuid.UUID) error
}

type taskRepository struct {
	queries *db.Queries
}

func NewTaskRepository(queries *db.Queries) TaskRepository {
	return &taskRepository{queries: queries}
}

func (r *taskRepository) CreateTask(ctx context.Context, params db.CreateTaskParams) (db.Task, error) {
	return r.queries.CreateTask(ctx, params)
}

func (r *taskRepository) GetTaskByID(ctx context.Context, workspaceID, taskID uuid.UUID) (db.Task, error) {
	return r.queries.GetTaskByID(ctx, db.GetTaskByIDParams{
		ID:          taskID,
		WorkspaceID: workspaceID,
	})
}

func (r *taskRepository) ListTasksByWorkspace(ctx context.Context, params db.ListTasksByWorkspaceParams) ([]db.Task, error) {
	return r.queries.ListTasksByWorkspace(ctx, params)
}

func (r *taskRepository) UpdateTask(ctx context.Context, params db.UpdateTaskParams) (db.Task, error) {
	return r.queries.UpdateTask(ctx, params)
}

func (r *taskRepository) SoftDeleteTask(ctx context.Context, workspaceID, taskID uuid.UUID) error {
	return r.queries.SoftDeleteTask(ctx, db.SoftDeleteTaskParams{
		ID:          taskID,
		WorkspaceID: workspaceID,
	})
}
