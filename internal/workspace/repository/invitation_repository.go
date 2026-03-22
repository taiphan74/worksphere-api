package repository

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	db "worksphere-api/internal/database/sqlc"
)

type InvitationRepository interface {
	CreateInvitation(ctx context.Context, params db.CreateWorkspaceInvitationParams) (db.WorkspaceInvitation, error)
	GetInvitationByID(ctx context.Context, id uuid.UUID) (db.WorkspaceInvitation, error)
	GetInvitationByToken(ctx context.Context, token string) (db.WorkspaceInvitation, error)
	GetInvitationByEmailAndWorkspace(ctx context.Context, email string, workspaceID uuid.UUID) (db.WorkspaceInvitation, error)
	ListInvitationsByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]db.WorkspaceInvitation, error)
	DeleteInvitation(ctx context.Context, id uuid.UUID) error
	CountPendingInvitationsByEmail(ctx context.Context, email string) (int64, error)
	WithTx(tx pgx.Tx) InvitationRepository
}

type invitationRepository struct {
	queries *db.Queries
}

func NewInvitationRepository(queries *db.Queries) InvitationRepository {
	return &invitationRepository{queries: queries}
}

func (r *invitationRepository) WithTx(tx pgx.Tx) InvitationRepository {
	return &invitationRepository{queries: r.queries.WithTx(tx)}
}

func (r *invitationRepository) CreateInvitation(ctx context.Context, params db.CreateWorkspaceInvitationParams) (db.WorkspaceInvitation, error) {
	return r.queries.CreateWorkspaceInvitation(ctx, params)
}

func (r *invitationRepository) GetInvitationByID(ctx context.Context, id uuid.UUID) (db.WorkspaceInvitation, error) {
	return r.queries.GetWorkspaceInvitationByID(ctx, id)
}

func (r *invitationRepository) GetInvitationByToken(ctx context.Context, token string) (db.WorkspaceInvitation, error) {
	return r.queries.GetWorkspaceInvitationByToken(ctx, token)
}

func (r *invitationRepository) GetInvitationByEmailAndWorkspace(ctx context.Context, email string, workspaceID uuid.UUID) (db.WorkspaceInvitation, error) {
	return r.queries.GetWorkspaceInvitationByEmailAndWorkspace(ctx, db.GetWorkspaceInvitationByEmailAndWorkspaceParams{
		Lower:       strings.ToLower(email),
		WorkspaceID: workspaceID,
	})
}

func (r *invitationRepository) ListInvitationsByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]db.WorkspaceInvitation, error) {
	return r.queries.ListWorkspaceInvitationsByWorkspace(ctx, workspaceID)
}

func (r *invitationRepository) DeleteInvitation(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteWorkspaceInvitation(ctx, id)
}

func (r *invitationRepository) CountPendingInvitationsByEmail(ctx context.Context, email string) (int64, error) {
	return r.queries.CountPendingInvitationsByEmail(ctx, email)
}
