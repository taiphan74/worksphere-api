package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/workspace/dto"
	"worksphere-api/internal/workspace/repository"
	apperrors "worksphere-api/pkg/errors"
)

// Error messages
const (
	errMemberNotFound   = "member not found"
	errNotAMember       = "you are not a member of this workspace"
	errAlreadyMember    = "user is already a member of this workspace"
	errInsufficientRole = "insufficient permissions"
	errCannotRemoveLast = "cannot remove the last owner of the workspace"
	errCannotDemoteLast = "cannot demote the last owner of the workspace"
	errInvalidRole      = "invalid role"
)

var validRoles = map[string]bool{
	"OWNER": true, "MEMBER": true,
}

type MemberService interface {
	AddMember(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID, req dto.AddMemberRequest) (dto.MemberResponse, error)
	ListMembers(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID) ([]dto.MemberResponse, error)
	GetMember(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID, targetUserID uuid.UUID) (dto.MemberResponse, error)
	UpdateMemberRole(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID, targetUserID uuid.UUID, req dto.UpdateMemberRoleRequest) (dto.MemberResponse, error)
	RemoveMember(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID, targetUserID uuid.UUID) error
}

type memberService struct {
	dbPool     *pgxpool.Pool
	memberRepo repository.MemberRepository
}

func NewMemberService(dbPool *pgxpool.Pool, memberRepo repository.MemberRepository) MemberService {
	return &memberService{dbPool: dbPool, memberRepo: memberRepo}
}

// AddMember adds a user to a workspace. Only OWNER can add members.
func (s *memberService) AddMember(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID, req dto.AddMemberRequest) (dto.MemberResponse, error) {
	// Check requester permission
	requester, err := s.memberRepo.GetMember(ctx, workspaceID, requesterID)
	if err != nil {
		return dto.MemberResponse{}, apperrors.New(http.StatusForbidden, "FORBIDDEN_ACCESS", errNotAMember)
	}
	if !isOwner(requester.Role) {
		return dto.MemberResponse{}, apperrors.New(http.StatusForbidden, "INSUFFICIENT_PERMISSIONS", errInsufficientRole)
	}

	if !validRoles[req.Role] {
		return dto.MemberResponse{}, apperrors.New(http.StatusBadRequest, "INVALID_ROLE", errInvalidRole)
	}

	targetUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		return dto.MemberResponse{}, apperrors.New(http.StatusBadRequest, "INVALID_USER_ID", "invalid user id")
	}

	member, err := s.memberRepo.AddMember(ctx, db.AddWorkspaceMemberParams{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		UserID:      targetUserID,
		Role:        req.Role,
	})
	if err != nil {
		// Check for unique constraint violation (already a member)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return dto.MemberResponse{}, apperrors.New(http.StatusConflict, "ALREADY_MEMBER", errAlreadyMember)
		}
		return dto.MemberResponse{}, mapMemberRepoError(err, "failed to add member")
	}

	return toMemberResponse(member), nil
}

// ListMembers lists all members of a workspace. Requester must be a member.
func (s *memberService) ListMembers(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID) ([]dto.MemberResponse, error) {
	// Verify requester is a member
	_, err := s.memberRepo.GetMember(ctx, workspaceID, requesterID)
	if err != nil {
		return nil, apperrors.New(http.StatusForbidden, "FORBIDDEN_ACCESS", errNotAMember)
	}

	members, err := s.memberRepo.ListMembers(ctx, workspaceID)
	if err != nil {
		return nil, mapMemberRepoError(err, "failed to list members")
	}

	res := make([]dto.MemberResponse, len(members))
	for i, m := range members {
		res[i] = toListMemberResponse(m)
	}
	return res, nil
}

// GetMember gets a single member. Requester must be a member.
func (s *memberService) GetMember(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID, targetUserID uuid.UUID) (dto.MemberResponse, error) {
	_, err := s.memberRepo.GetMember(ctx, workspaceID, requesterID)
	if err != nil {
		return dto.MemberResponse{}, apperrors.New(http.StatusForbidden, "FORBIDDEN_ACCESS", errNotAMember)
	}

	member, err := s.memberRepo.GetMemberWithUserInfo(ctx, workspaceID, targetUserID)
	if err != nil {
		return dto.MemberResponse{}, mapMemberRepoError(err, errMemberNotFound)
	}

	return toMemberWithUserInfoResponse(member), nil
}

// UpdateMemberRole updates the role of a member. Only OWNER can change roles.
func (s *memberService) UpdateMemberRole(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID, targetUserID uuid.UUID, req dto.UpdateMemberRoleRequest) (dto.MemberResponse, error) {
	// Start transaction
	tx, err := s.dbPool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return dto.MemberResponse{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to start transaction")
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	txRepo := s.memberRepo.WithTx(tx)

	// Check requester is OWNER
	requester, err := txRepo.GetMember(ctx, workspaceID, requesterID)
	if err != nil {
		_ = tx.Rollback(ctx)
		return dto.MemberResponse{}, apperrors.New(http.StatusForbidden, "FORBIDDEN_ACCESS", errNotAMember)
	}
	if requester.Role != "OWNER" {
		_ = tx.Rollback(ctx)
		return dto.MemberResponse{}, apperrors.New(http.StatusForbidden, "INSUFFICIENT_PERMISSIONS", "only workspace owners can change roles")
	}

	if !validRoles[req.Role] {
		_ = tx.Rollback(ctx)
		return dto.MemberResponse{}, apperrors.New(http.StatusBadRequest, "INVALID_ROLE", errInvalidRole)
	}

	target, err := txRepo.GetMember(ctx, workspaceID, targetUserID)
	if err != nil {
		_ = tx.Rollback(ctx)
		return dto.MemberResponse{}, apperrors.New(http.StatusNotFound, "MEMBER_NOT_FOUND", errMemberNotFound)
	}

	if target.Role == "OWNER" && req.Role != "OWNER" {
		count, err := txRepo.CountMembersByRole(ctx, workspaceID, "OWNER")
		if err != nil {
			_ = tx.Rollback(ctx)
			return dto.MemberResponse{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to validate role change")
		}
		if count <= 1 {
			_ = tx.Rollback(ctx)
			return dto.MemberResponse{}, apperrors.New(http.StatusBadRequest, "LAST_OWNER_PROTECTION", errCannotDemoteLast)
		}
	}

	updated, err := txRepo.UpdateMemberRole(ctx, target.ID, req.Role)
	if err != nil {
		_ = tx.Rollback(ctx)
		return dto.MemberResponse{}, mapMemberRepoError(err, "failed to update role")
	}

	if err := tx.Commit(ctx); err != nil {
		return dto.MemberResponse{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to commit transaction")
	}

	return toMemberResponse(updated), nil
}

// RemoveMember removes a member from the workspace. OWNER can remove. Self-leave is allowed.
func (s *memberService) RemoveMember(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID, targetUserID uuid.UUID) error {
	tx, err := s.dbPool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to start transaction")
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	txRepo := s.memberRepo.WithTx(tx)

	requester, err := txRepo.GetMember(ctx, workspaceID, requesterID)
	if err != nil {
		_ = tx.Rollback(ctx)
		return apperrors.New(http.StatusForbidden, "FORBIDDEN_ACCESS", errNotAMember)
	}

	target, err := txRepo.GetMember(ctx, workspaceID, targetUserID)
	if err != nil {
		_ = tx.Rollback(ctx)
		return apperrors.New(http.StatusNotFound, "MEMBER_NOT_FOUND", errMemberNotFound)
	}

	isSelfLeave := requesterID == targetUserID
	if !isSelfLeave && !isOwner(requester.Role) {
		_ = tx.Rollback(ctx)
		return apperrors.New(http.StatusForbidden, "INSUFFICIENT_PERMISSIONS", errInsufficientRole)
	}

	if target.Role == "OWNER" {
		count, err := txRepo.CountMembersByRole(ctx, workspaceID, "OWNER")
		if err != nil {
			_ = tx.Rollback(ctx)
			return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to validate removal")
		}
		if count <= 1 {
			_ = tx.Rollback(ctx)
			return apperrors.New(http.StatusBadRequest, "LAST_OWNER_PROTECTION", errCannotRemoveLast)
		}
	}

	err = txRepo.DeleteMember(ctx, target.ID)
	if err != nil {
		_ = tx.Rollback(ctx)
		return mapMemberRepoError(err, "failed to remove member")
	}

	if err := tx.Commit(ctx); err != nil {
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to commit transaction")
	}

	return nil
}

// ── Helpers ──

func isOwner(role string) bool {
	return role == "OWNER"
}

func toMemberResponse(m db.WorkspaceMember) dto.MemberResponse {
	return dto.MemberResponse{
		ID:          m.ID.String(),
		WorkspaceID: m.WorkspaceID.String(),
		UserID:      m.UserID.String(),
		Role:        m.Role,
		CreatedAt:   m.CreatedAt.Time,
		UpdatedAt:   m.UpdatedAt.Time,
	}
}

func toListMemberResponse(m db.ListWorkspaceMembersByWorkspaceRow) dto.MemberResponse {
	res := dto.MemberResponse{
		ID:          m.ID.String(),
		WorkspaceID: m.WorkspaceID.String(),
		UserID:      m.UserID.String(),
		Role:        m.Role,
		Email:       m.Email,
		UserStatus:  m.UserStatus,
		CreatedAt:   m.CreatedAt.Time,
		UpdatedAt:   m.UpdatedAt.Time,
	}

	if m.FullName.Valid {
		res.FullName = &m.FullName.String
	}
	if m.AvatarKey.Valid {
		res.AvatarKey = &m.AvatarKey.String
	}

	return res
}

func toMemberWithUserInfoResponse(m db.GetWorkspaceMemberWithUserInfoRow) dto.MemberResponse {
	res := dto.MemberResponse{
		ID:          m.ID.String(),
		WorkspaceID: m.WorkspaceID.String(),
		UserID:      m.UserID.String(),
		Role:        m.Role,
		Email:       m.Email,
		UserStatus:  m.UserStatus,
		CreatedAt:   m.CreatedAt.Time,
		UpdatedAt:   m.UpdatedAt.Time,
	}

	if m.FullName.Valid {
		res.FullName = &m.FullName.String
	}
	if m.AvatarKey.Valid {
		res.AvatarKey = &m.AvatarKey.String
	}

	return res
}

func mapMemberRepoError(err error, notFoundMsg string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.New(http.StatusNotFound, "RESOURCE_NOT_FOUND", notFoundMsg)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return apperrors.New(http.StatusRequestTimeout, "REQUEST_TIMEOUT", "request timed out")
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperrors.New(http.StatusConflict, "ALREADY_EXISTS", "this record already exists")
	}
	return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred")
}
