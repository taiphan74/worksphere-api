package repository

import (
	"context"
	"strings"

	"github.com/google/uuid"

	db "worksphere-api/internal/database/sqlc"
)

type InvitationRepository interface {
	CreateInvitation(ctx context.Context, params db.CreateWorkspaceInvitationParams) (db.WorkspaceInvitation, error)
	GetInvitationByID(ctx context.Context, id uuid.UUID) (db.WorkspaceInvitation, error)
	GetInvitationByToken(ctx context.Context, token string) (db.WorkspaceInvitation, error)
	GetInvitationByEmailAndWorkspace(ctx context.Context, email string, workspaceID uuid.UUID) (db.WorkspaceInvitation, error)
	ListInvitationsByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]db.WorkspaceInvitation, error)
	ListPendingInvitationsByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]db.WorkspaceInvitation, error)
	AcceptInvitation(ctx context.Context, id uuid.UUID) (db.WorkspaceInvitation, error)
	DeclineInvitation(ctx context.Context, id uuid.UUID) (db.WorkspaceInvitation, error)
	CancelInvitation(ctx context.Context, id uuid.UUID) (db.WorkspaceInvitation, error)
	DeleteInvitation(ctx context.Context, id uuid.UUID) error
	CountPendingInvitationsByEmail(ctx context.Context, email string) (int64, error)
}

type invitationRepository struct {
	queries *db.Queries
}

func NewInvitationRepository(queries *db.Queries) InvitationRepository {
	return &invitationRepository{queries: queries}
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

func (r *invitationRepository) ListPendingInvitationsByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]db.WorkspaceInvitation, error) {
	return r.queries.ListPendingInvitationsByWorkspace(ctx, workspaceID)
}

func (r *invitationRepository) AcceptInvitation(ctx context.Context, id uuid.UUID) (db.WorkspaceInvitation, error) {
	return r.queries.AcceptWorkspaceInvitation(ctx, id)
}

func (r *invitationRepository) DeclineInvitation(ctx context.Context, id uuid.UUID) (db.WorkspaceInvitation, error) {
	return r.queries.DeclineWorkspaceInvitation(ctx, id)
}

func (r *invitationRepository) CancelInvitation(ctx context.Context, id uuid.UUID) (db.WorkspaceInvitation, error) {
	return r.queries.CancelWorkspaceInvitation(ctx, id)
}

func (r *invitationRepository) DeleteInvitation(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteWorkspaceInvitation(ctx, id)
}

func (r *invitationRepository) CountPendingInvitationsByEmail(ctx context.Context, email string) (int64, error) {
	return r.queries.CountPendingInvitationsByEmail(ctx, email)
}
