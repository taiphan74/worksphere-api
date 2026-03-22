package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	db "worksphere-api/internal/database/sqlc"
)

type MemberRepository interface {
	AddMember(ctx context.Context, params db.AddWorkspaceMemberParams) (db.WorkspaceMember, error)
	GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (db.WorkspaceMember, error)
	GetMemberWithUserInfo(ctx context.Context, workspaceID, userID uuid.UUID) (db.GetWorkspaceMemberWithUserInfoRow, error)
	ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]db.ListWorkspaceMembersByWorkspaceRow, error)
	UpdateMemberRole(ctx context.Context, id uuid.UUID, role string) (db.WorkspaceMember, error)
	DeleteMember(ctx context.Context, id uuid.UUID) error
	CountMembersByRole(ctx context.Context, workspaceID uuid.UUID, role string) (int64, error)
	WithTx(tx pgx.Tx) MemberRepository
}

type memberRepository struct {
	queries *db.Queries
}

func NewMemberRepository(queries *db.Queries) MemberRepository {
	return &memberRepository{queries: queries}
}

func (r *memberRepository) WithTx(tx pgx.Tx) MemberRepository {
	return &memberRepository{queries: r.queries.WithTx(tx)}
}

func (r *memberRepository) AddMember(ctx context.Context, params db.AddWorkspaceMemberParams) (db.WorkspaceMember, error) {
	return r.queries.AddWorkspaceMember(ctx, params)
}

func (r *memberRepository) GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (db.WorkspaceMember, error) {
	return r.queries.GetWorkspaceMember(ctx, db.GetWorkspaceMemberParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
	})
}

func (r *memberRepository) GetMemberWithUserInfo(ctx context.Context, workspaceID, userID uuid.UUID) (db.GetWorkspaceMemberWithUserInfoRow, error) {
	return r.queries.GetWorkspaceMemberWithUserInfo(ctx, db.GetWorkspaceMemberWithUserInfoParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
	})
}

func (r *memberRepository) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]db.ListWorkspaceMembersByWorkspaceRow, error) {
	return r.queries.ListWorkspaceMembersByWorkspace(ctx, workspaceID)
}

func (r *memberRepository) UpdateMemberRole(ctx context.Context, id uuid.UUID, role string) (db.WorkspaceMember, error) {
	return r.queries.UpdateMemberRole(ctx, db.UpdateMemberRoleParams{
		ID:   id,
		Role: role,
	})
}

func (r *memberRepository) DeleteMember(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteWorkspaceMember(ctx, id)
}

func (r *memberRepository) CountMembersByRole(ctx context.Context, workspaceID uuid.UUID, role string) (int64, error) {
	return r.queries.CountWorkspaceMembersByRole(ctx, db.CountWorkspaceMembersByRoleParams{
		WorkspaceID: workspaceID,
		Role:        role,
	})
}
