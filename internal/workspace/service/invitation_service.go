package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/email"
	"worksphere-api/internal/workspace/dto"
	"worksphere-api/internal/workspace/repository"
	apperrors "worksphere-api/pkg/errors"
)

// Error messages
const (
	errInvitationNotFound    = "invitation not found"
	errInvitationAlreadyUsed = "invitation has already been used"
	errNotWorkspaceOwner     = "only workspace owners can manage invitations"
	errEmailAlreadyInvited   = "this email has already been invited to this workspace"
	errUserAlreadyMember     = "user is already a member of this workspace"
	errCannotInviteSelf      = "cannot invite yourself"
)

type InvitationService interface {
	SendInvitation(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID, req dto.SendInvitationRequest) (dto.InvitationResponse, error)
	GetInvitation(ctx context.Context, requesterID uuid.UUID, invitationID uuid.UUID) (dto.InvitationResponse, error)
	ListInvitations(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID) ([]dto.InvitationResponse, error)
	AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) error
	DeclineInvitation(ctx context.Context, token string, userID uuid.UUID) error
	CancelInvitation(ctx context.Context, requesterID uuid.UUID, invitationID uuid.UUID) error
	ResendInvitation(ctx context.Context, requesterID uuid.UUID, invitationID uuid.UUID) error
}

type invitationService struct {
	dbPool         *pgxpool.Pool
	invitationRepo repository.InvitationRepository
	memberRepo     repository.MemberRepository
	workspaceRepo  repository.WorkspaceRepository
	emailService   email.Service
	frontendURL    string
	smtpFrom       string
}

func NewInvitationService(
	dbPool *pgxpool.Pool,
	invitationRepo repository.InvitationRepository,
	memberRepo repository.MemberRepository,
	workspaceRepo repository.WorkspaceRepository,
	emailService email.Service,
	frontendURL string,
	smtpFrom string,
) InvitationService {
	return &invitationService{
		dbPool:         dbPool,
		invitationRepo: invitationRepo,
		memberRepo:     memberRepo,
		workspaceRepo:  workspaceRepo,
		emailService:   emailService,
		frontendURL:    frontendURL,
		smtpFrom:       smtpFrom,
	}
}

// SendInvitation sends an invitation to a user's email
func (s *invitationService) SendInvitation(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID, req dto.SendInvitationRequest) (dto.InvitationResponse, error) {
	// Check requester is OWNER
	requester, err := s.memberRepo.GetMember(ctx, workspaceID, requesterID)
	if err != nil {
		return dto.InvitationResponse{}, apperrors.New(http.StatusForbidden, "FORBIDDEN_ACCESS", "you don't have access to this workspace")
	}
	if requester.Role != "OWNER" {
		return dto.InvitationResponse{}, apperrors.New(http.StatusForbidden, "NOT_WORKSPACE_OWNER", errNotWorkspaceOwner)
	}

	// Normalize email
	inviteeEmail := strings.ToLower(strings.TrimSpace(req.Email))

	// Check if already invited (record exists = pending)
	_, err = s.invitationRepo.GetInvitationByEmailAndWorkspace(ctx, inviteeEmail, workspaceID)
	if err == nil {
		return dto.InvitationResponse{}, apperrors.New(http.StatusConflict, "ALREADY_INVITED", errEmailAlreadyInvited)
	}

	// Generate token
	token := uuid.New().String() + "-" + uuid.New().String()

	// Create invitation
	invitation, err := s.invitationRepo.CreateInvitation(ctx, db.CreateWorkspaceInvitationParams{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Lower:       inviteeEmail,
		Token:       token,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return dto.InvitationResponse{}, apperrors.New(http.StatusConflict, "ALREADY_INVITED", errEmailAlreadyInvited)
		}
		return dto.InvitationResponse{}, mapInvitationError(err, "failed to create invitation")
	}

	// Get workspace info for email
	workspace, err := s.workspaceRepo.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return dto.InvitationResponse{}, apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get workspace info")
	}

	// Send email
	acceptLink := fmt.Sprintf("%s?token=%s", s.frontendURL, token)
	emailBody := s.buildInvitationEmail(workspace.Name, acceptLink)

	if err := s.emailService.SendHTML(ctx, inviteeEmail, "You're invited to join a workspace!", emailBody); err != nil {
		// Don't fail the invitation if email fails
	}

	return toInvitationResponse(invitation), nil
}

// GetInvitation gets a single invitation by ID
func (s *invitationService) GetInvitation(ctx context.Context, requesterID uuid.UUID, invitationID uuid.UUID) (dto.InvitationResponse, error) {
	invitation, err := s.invitationRepo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return dto.InvitationResponse{}, apperrors.New(http.StatusNotFound, "INVITATION_NOT_FOUND", errInvitationNotFound)
	}

	_, err = s.memberRepo.GetMember(ctx, invitation.WorkspaceID, requesterID)
	if err != nil {
		return dto.InvitationResponse{}, apperrors.New(http.StatusForbidden, "FORBIDDEN_ACCESS", "you don't have access to this invitation")
	}

	return toInvitationResponse(invitation), nil
}

// ListInvitations lists all pending invitations for a workspace
func (s *invitationService) ListInvitations(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID) ([]dto.InvitationResponse, error) {
	_, err := s.memberRepo.GetMember(ctx, workspaceID, requesterID)
	if err != nil {
		return nil, apperrors.New(http.StatusForbidden, "FORBIDDEN_ACCESS", "you don't have access to this workspace")
	}

	invitations, err := s.invitationRepo.ListInvitationsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, mapInvitationError(err, "failed to list invitations")
	}

	res := make([]dto.InvitationResponse, len(invitations))
	for i, inv := range invitations {
		res[i] = toInvitationResponse(inv)
	}
	return res, nil
}

// AcceptInvitation accepts an invitation: add member + delete invitation record
func (s *invitationService) AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) error {
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

	txInvitationRepo := s.invitationRepo.WithTx(tx)
	txMemberRepo := s.memberRepo.WithTx(tx)

	// Get invitation by token
	invitation, err := txInvitationRepo.GetInvitationByToken(ctx, token)
	if err != nil {
		_ = tx.Rollback(ctx)
		return apperrors.New(http.StatusNotFound, "INVITATION_NOT_FOUND", errInvitationNotFound)
	}

	// Check if user is already a member
	_, err = txMemberRepo.GetMember(ctx, invitation.WorkspaceID, userID)
	if err == nil {
		_ = tx.Rollback(ctx)
		return apperrors.New(http.StatusConflict, "ALREADY_MEMBER", errUserAlreadyMember)
	}

	// Add user as MEMBER
	_, err = txMemberRepo.AddMember(ctx, db.AddWorkspaceMemberParams{
		ID:          uuid.New(),
		WorkspaceID: invitation.WorkspaceID,
		UserID:      userID,
		Role:        "MEMBER",
	})
	if err != nil {
		_ = tx.Rollback(ctx)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperrors.New(http.StatusConflict, "ALREADY_MEMBER", errUserAlreadyMember)
		}
		return mapInvitationError(err, "failed to add member")
	}

	// Delete invitation record (no longer pending)
	if err := txInvitationRepo.DeleteInvitation(ctx, invitation.ID); err != nil {
		_ = tx.Rollback(ctx)
		return mapInvitationError(err, "failed to remove invitation")
	}

	if err := tx.Commit(ctx); err != nil {
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to commit transaction")
	}

	return nil
}

// DeclineInvitation declines an invitation: just delete the record
func (s *invitationService) DeclineInvitation(ctx context.Context, token string, userID uuid.UUID) error {
	invitation, err := s.invitationRepo.GetInvitationByToken(ctx, token)
	if err != nil {
		return apperrors.New(http.StatusNotFound, "INVITATION_NOT_FOUND", errInvitationNotFound)
	}

	if err := s.invitationRepo.DeleteInvitation(ctx, invitation.ID); err != nil {
		return mapInvitationError(err, "failed to decline invitation")
	}

	return nil
}

// CancelInvitation cancels a pending invitation: owner deletes the record
func (s *invitationService) CancelInvitation(ctx context.Context, requesterID uuid.UUID, invitationID uuid.UUID) error {
	invitation, err := s.invitationRepo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return apperrors.New(http.StatusNotFound, "INVITATION_NOT_FOUND", errInvitationNotFound)
	}

	requester, err := s.memberRepo.GetMember(ctx, invitation.WorkspaceID, requesterID)
	if err != nil {
		return apperrors.New(http.StatusForbidden, "FORBIDDEN_ACCESS", "you don't have access to this workspace")
	}
	if requester.Role != "OWNER" {
		return apperrors.New(http.StatusForbidden, "NOT_WORKSPACE_OWNER", errNotWorkspaceOwner)
	}

	if err := s.invitationRepo.DeleteInvitation(ctx, invitationID); err != nil {
		return mapInvitationError(err, "failed to cancel invitation")
	}

	return nil
}

// ResendInvitation resends the invitation email
func (s *invitationService) ResendInvitation(ctx context.Context, requesterID uuid.UUID, invitationID uuid.UUID) error {
	invitation, err := s.invitationRepo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return apperrors.New(http.StatusNotFound, "INVITATION_NOT_FOUND", errInvitationNotFound)
	}

	requester, err := s.memberRepo.GetMember(ctx, invitation.WorkspaceID, requesterID)
	if err != nil {
		return apperrors.New(http.StatusForbidden, "FORBIDDEN_ACCESS", "you don't have access to this workspace")
	}
	if requester.Role != "OWNER" {
		return apperrors.New(http.StatusForbidden, "NOT_WORKSPACE_OWNER", errNotWorkspaceOwner)
	}

	workspace, err := s.workspaceRepo.GetWorkspaceByID(ctx, invitation.WorkspaceID)
	if err != nil {
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get workspace info")
	}

	acceptLink := fmt.Sprintf("%s?token=%s", s.frontendURL, invitation.Token)
	emailBody := s.buildInvitationEmail(workspace.Name, acceptLink)

	if err := s.emailService.SendHTML(ctx, invitation.Email, "Reminder: You're invited to join a workspace!", emailBody); err != nil {
		return apperrors.New(http.StatusInternalServerError, "EMAIL_SEND_FAILED", "failed to send email")
	}

	return nil
}

// buildInvitationEmail builds the HTML email body
func (s *invitationService) buildInvitationEmail(workspaceName, acceptLink string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 30px; border-radius: 10px 10px 0 0; text-align: center; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 10px 10px; }
        .button { display: inline-block; background: #667eea; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; margin: 20px 0; font-weight: bold; }
        .button:hover { background: #5568d3; }
        .footer { margin-top: 20px; padding-top: 20px; border-top: 1px solid #ddd; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎉 You're Invited!</h1>
        </div>
        <div class="content">
            <p>Hi there,</p>
            <p>You've been invited to join the workspace <strong>%s</strong>.</p>
            <p>Click the button below to accept the invitation and start collaborating:</p>
            <p style="text-align: center;">
                <a href="%s" class="button">Accept Invitation</a>
            </p>
            <p>If the button doesn't work, copy and paste this link into your browser:</p>
            <p style="word-break: break-all; color: #667eea;">%s</p>
            <div class="footer">
                <p>This email was sent from Worksphere. If you didn't expect this invitation, you can safely ignore it.</p>
            </div>
        </div>
    </div>
</body>
</html>
`, workspaceName, acceptLink, acceptLink)
}

// ── Helpers ──

func toInvitationResponse(inv db.WorkspaceInvitation) dto.InvitationResponse {
	return dto.InvitationResponse{
		ID:          inv.ID.String(),
		WorkspaceID: inv.WorkspaceID.String(),
		Email:       inv.Email,
		CreatedAt:   inv.CreatedAt.Time,
		UpdatedAt:   inv.UpdatedAt.Time,
	}
}

func mapInvitationError(err error, defaultMsg string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.New(http.StatusNotFound, "RESOURCE_NOT_FOUND", defaultMsg)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return apperrors.New(http.StatusRequestTimeout, "REQUEST_TIMEOUT", "request timed out")
	}
	return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred")
}
