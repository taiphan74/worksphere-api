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
	ID             string     `json:"id"`
	WorkspaceID    string     `json:"workspace_id"`
	Email          string     `json:"email"`
	Status         string     `json:"status"`
	AcceptedAt     *time.Time `json:"accepted_at,omitempty"`
	DeclinedAt     *time.Time `json:"declined_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type InvitationWithWorkspaceResponse struct {
	InvitationResponse
	WorkspaceName string `json:"workspace_name"`
	WorkspaceSlug string `json:"workspace_slug"`
	InviterName   string `json:"inviter_name"`
}
