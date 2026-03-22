package dto

import "time"

// ── Requests ──

type SendInvitationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type AcceptInvitationRequest struct {
	Token string `json:"token" binding:"required"`
}

// ── Responses ──

type InvitationResponse struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Email       string    `json:"email"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type InvitationWithWorkspaceResponse struct {
	InvitationResponse
	WorkspaceName string `json:"workspace_name"`
	WorkspaceSlug string `json:"workspace_slug"`
	InviterName   string `json:"inviter_name"`
}
