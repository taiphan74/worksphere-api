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

	db "worksphere-api/internal/database/sqlc"
	"worksphere-api/internal/email"
	"worksphere-api/internal/workspace/dto"
	"worksphere-api/internal/workspace/repository"
	apperrors "worksphere-api/pkg/errors"
)

// Error messages
const (
	errInvitationNotFound     = "invitation not found"
	errInvitationExpired      = "invitation has expired"
	errInvitationAlreadyUsed  = "invitation has already been used"
	errInvitationCancelled    = "invitation has been cancelled"
	errNotWorkspaceOwner      = "only workspace owners can send invitations"
	errEmailAlreadyInvited    = "this email has already been invited to this workspace"
	errUserAlreadyMember      = "user is already a member of this workspace"
	errCannotInviteSelf       = "cannot invite yourself"
)

type InvitationService interface {
	SendInvitation(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID, req dto.SendInvitationRequest) (dto.InvitationResponse, error)
	GetInvitation(ctx context.Context, requesterID uuid.UUID, invitationID uuid.UUID) (dto.InvitationResponse, error)
	ListInvitations(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID) ([]dto.InvitationResponse, error)
	AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) (dto.InvitationResponse, error)
	DeclineInvitation(ctx context.Context, token string, userID uuid.UUID) error
	CancelInvitation(ctx context.Context, requesterID uuid.UUID, invitationID uuid.UUID) error
	ResendInvitation(ctx context.Context, requesterID uuid.UUID, invitationID uuid.UUID) error
}

type invitationService struct {
	invitationRepo repository.InvitationRepository
	memberRepo     repository.MemberRepository
	workspaceRepo  repository.WorkspaceRepository
	emailService   email.Service
	frontendURL    string
	smtpFrom       string
}

func NewInvitationService(
	invitationRepo repository.InvitationRepository,
	memberRepo repository.MemberRepository,
	workspaceRepo repository.WorkspaceRepository,
	emailService email.Service,
	frontendURL string,
	smtpFrom string,
) InvitationService {
	return &invitationService{
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
		return dto.InvitationResponse{}, mapInvitationError(err, errNotWorkspaceOwner)
	}
	if requester.Role != "OWNER" {
		return dto.InvitationResponse{}, apperrors.New(http.StatusForbidden, "FORBIDDEN", errNotWorkspaceOwner)
	}

	// Normalize email
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Check for existing pending invitation to this workspace
	existingInv, err := s.invitationRepo.GetInvitationByEmailAndWorkspace(ctx, email, workspaceID)
	if err == nil && existingInv.Status == "pending" && existingInv.WorkspaceID == workspaceID {
		return dto.InvitationResponse{}, apperrors.New(http.StatusConflict, "ALREADY_INVITED", errEmailAlreadyInvited)
	}

	// Generate token
	token := uuid.New().String() + "-" + uuid.New().String()

	// Create invitation
	invitation, err := s.invitationRepo.CreateInvitation(ctx, db.CreateWorkspaceInvitationParams{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Lower:       email,
		Token:       token,
	})
	if err != nil {
		// Check for unique constraint violation
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

	if err := s.emailService.SendHTML(ctx, email, "You're invited to join a workspace!", emailBody); err != nil {
		// Don't fail the invitation if email fails, but log it
	}

	return toInvitationResponse(invitation), nil
}

// GetInvitation gets a single invitation by ID
func (s *invitationService) GetInvitation(ctx context.Context, requesterID uuid.UUID, invitationID uuid.UUID) (dto.InvitationResponse, error) {
	invitation, err := s.invitationRepo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return dto.InvitationResponse{}, mapInvitationError(err, errInvitationNotFound)
	}

	// Check requester has access to this workspace
	_, err = s.memberRepo.GetMember(ctx, invitation.WorkspaceID, requesterID)
	if err != nil {
		return dto.InvitationResponse{}, apperrors.New(http.StatusForbidden, "FORBIDDEN", "you don't have access to this invitation")
	}

	return toInvitationResponse(invitation), nil
}

// ListInvitations lists all invitations for a workspace
func (s *invitationService) ListInvitations(ctx context.Context, requesterID uuid.UUID, workspaceID uuid.UUID) ([]dto.InvitationResponse, error) {
	// Check requester is a member
	_, err := s.memberRepo.GetMember(ctx, workspaceID, requesterID)
	if err != nil {
		return nil, mapInvitationError(err, "you don't have access to this workspace")
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

// AcceptInvitation accepts an invitation using a token
func (s *invitationService) AcceptInvitation(ctx context.Context, token string, userID uuid.UUID) (dto.InvitationResponse, error) {
	// Get invitation by token
	invitation, err := s.invitationRepo.GetInvitationByToken(ctx, token)
	if err != nil {
		return dto.InvitationResponse{}, mapInvitationError(err, errInvitationNotFound)
	}

	// Check status
	if invitation.Status != "pending" {
		return dto.InvitationResponse{}, apperrors.New(http.StatusBadRequest, "INVALID_STATUS", errInvitationAlreadyUsed)
	}

	// Check if user is already a member
	_, err = s.memberRepo.GetMember(ctx, invitation.WorkspaceID, userID)
	if err == nil {
		return dto.InvitationResponse{}, apperrors.New(http.StatusConflict, "ALREADY_MEMBER", errUserAlreadyMember)
	}

	// Accept invitation - add to workspace members
	_, err = s.memberRepo.AddMember(ctx, db.AddWorkspaceMemberParams{
		ID:          uuid.New(),
		WorkspaceID: invitation.WorkspaceID,
		UserID:      userID,
		Role:        "MEMBER",
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return dto.InvitationResponse{}, apperrors.New(http.StatusConflict, "ALREADY_MEMBER", errUserAlreadyMember)
		}
		return dto.InvitationResponse{}, mapInvitationError(err, "failed to add member")
	}

	// Mark invitation as accepted
	accepted, err := s.invitationRepo.AcceptInvitation(ctx, invitation.ID)
	if err != nil {
		return dto.InvitationResponse{}, mapInvitationError(err, "failed to accept invitation")
	}

	return toInvitationResponse(accepted), nil
}

// DeclineInvitation declines an invitation using a token
func (s *invitationService) DeclineInvitation(ctx context.Context, token string, userID uuid.UUID) error {
	// Get invitation by token
	invitation, err := s.invitationRepo.GetInvitationByToken(ctx, token)
	if err != nil {
		return mapInvitationError(err, errInvitationNotFound)
	}

	// Check status
	if invitation.Status != "pending" {
		return apperrors.New(http.StatusBadRequest, "INVALID_STATUS", errInvitationAlreadyUsed)
	}

	// Mark as declined
	_, err = s.invitationRepo.DeclineInvitation(ctx, invitation.ID)
	if err != nil {
		return mapInvitationError(err, "failed to decline invitation")
	}

	return nil
}

// CancelInvitation cancels a pending invitation
func (s *invitationService) CancelInvitation(ctx context.Context, requesterID uuid.UUID, invitationID uuid.UUID) error {
	invitation, err := s.invitationRepo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return mapInvitationError(err, errInvitationNotFound)
	}

	// Check requester is OWNER of the workspace
	requester, err := s.memberRepo.GetMember(ctx, invitation.WorkspaceID, requesterID)
	if err != nil {
		return apperrors.New(http.StatusForbidden, "FORBIDDEN", errNotWorkspaceOwner)
	}
	if requester.Role != "OWNER" {
		return apperrors.New(http.StatusForbidden, "FORBIDDEN", errNotWorkspaceOwner)
	}

	// Can only cancel pending invitations
	if invitation.Status != "pending" {
		return apperrors.New(http.StatusBadRequest, "INVALID_STATUS", "can only cancel pending invitations")
	}

	// Cancel invitation
	_, err = s.invitationRepo.CancelInvitation(ctx, invitationID)
	if err != nil {
		return mapInvitationError(err, "failed to cancel invitation")
	}

	return nil
}

// ResendInvitation resends the invitation email
func (s *invitationService) ResendInvitation(ctx context.Context, requesterID uuid.UUID, invitationID uuid.UUID) error {
	invitation, err := s.invitationRepo.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return mapInvitationError(err, errInvitationNotFound)
	}

	// Check requester is OWNER of the workspace
	requester, err := s.memberRepo.GetMember(ctx, invitation.WorkspaceID, requesterID)
	if err != nil {
		return apperrors.New(http.StatusForbidden, "FORBIDDEN", errNotWorkspaceOwner)
	}
	if requester.Role != "OWNER" {
		return apperrors.New(http.StatusForbidden, "FORBIDDEN", errNotWorkspaceOwner)
	}

	// Can only resend pending invitations
	if invitation.Status != "pending" {
		return apperrors.New(http.StatusBadRequest, "INVALID_STATUS", "can only resend pending invitations")
	}

	// Get workspace info
	workspace, err := s.workspaceRepo.GetWorkspaceByID(ctx, invitation.WorkspaceID)
	if err != nil {
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get workspace info")
	}

	// Send email
	acceptLink := fmt.Sprintf("%s?token=%s", s.frontendURL, invitation.Token)
	emailBody := s.buildInvitationEmail(workspace.Name, acceptLink)

	if err := s.emailService.SendHTML(ctx, invitation.Email, "Reminder: You're invited to join a workspace!", emailBody); err != nil {
		return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "failed to send email")
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
	resp := dto.InvitationResponse{
		ID:          inv.ID.String(),
		WorkspaceID: inv.WorkspaceID.String(),
		Email:       inv.Email,
		Status:      inv.Status,
		CreatedAt:   inv.CreatedAt.Time,
		UpdatedAt:   inv.UpdatedAt.Time,
	}

	if inv.AcceptedAt.Valid {
		resp.AcceptedAt = &inv.AcceptedAt.Time
	}
	if inv.DeclinedAt.Valid {
		resp.DeclinedAt = &inv.DeclinedAt.Time
	}

	return resp
}

func mapInvitationError(err error, defaultMsg string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.New(http.StatusNotFound, "NOT_FOUND", defaultMsg)
	}
	return apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred")
}
