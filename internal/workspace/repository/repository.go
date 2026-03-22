package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/jackc/pgx/v5/pgtype"

	db "worksphere-api/internal/database/sqlc"
)

type WorkspaceRepository interface {
	CreateWorkspace(ctx context.Context, params db.CreateWorkspaceParams) (db.Workspace, error)
	GetWorkspaceByID(ctx context.Context, id uuid.UUID) (db.Workspace, error)
	GetWorkspaceBySlug(ctx context.Context, slug string) (db.Workspace, error)
	ListWorkspacesByUser(ctx context.Context, userID uuid.UUID) ([]db.Workspace, error)
	UpdateWorkspace(ctx context.Context, params db.UpdateWorkspaceParams) (db.Workspace, error)
	DeleteWorkspace(ctx context.Context, id uuid.UUID) error
	CheckSlugExists(ctx context.Context, slug string, excludeID uuid.UUID) (bool, error)
	WithTx(tx pgx.Tx) WorkspaceRepository
}

type workspaceRepository struct {
	queries *db.Queries
}

func NewWorkspaceRepository(queries *db.Queries) WorkspaceRepository {
	return &workspaceRepository{queries: queries}
}

func (r *workspaceRepository) WithTx(tx pgx.Tx) WorkspaceRepository {
	return &workspaceRepository{queries: r.queries.WithTx(tx)}
}

func (r *workspaceRepository) CreateWorkspace(ctx context.Context, params db.CreateWorkspaceParams) (db.Workspace, error) {
	return r.queries.CreateWorkspace(ctx, params)
}

func (r *workspaceRepository) GetWorkspaceByID(ctx context.Context, id uuid.UUID) (db.Workspace, error) {
	return r.queries.GetWorkspaceByID(ctx, id)
}

func (r *workspaceRepository) GetWorkspaceBySlug(ctx context.Context, slug string) (db.Workspace, error) {
	return r.queries.GetWorkspaceBySlug(ctx, slug)
}

func (r *workspaceRepository) ListWorkspacesByUser(ctx context.Context, userID uuid.UUID) ([]db.Workspace, error) {
	return r.queries.ListWorkspacesByUser(ctx, userID)
}

func (r *workspaceRepository) UpdateWorkspace(ctx context.Context, params db.UpdateWorkspaceParams) (db.Workspace, error) {
	return r.queries.UpdateWorkspace(ctx, params)
}

func (r *workspaceRepository) DeleteWorkspace(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteWorkspace(ctx, id)
}

func (r *workspaceRepository) CheckSlugExists(ctx context.Context, slug string, excludeID uuid.UUID) (bool, error) {
	params := db.CheckSlugExistsParams{
		Slug: slug,
	}

	if excludeID != uuid.Nil {
		params.ExcludeID = pgtype.UUID{
			Bytes: excludeID,
			Valid: true,
		}
	}

	return r.queries.CheckSlugExists(ctx, params)
}
