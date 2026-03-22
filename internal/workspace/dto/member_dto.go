package dto

import "time"

// ── Requests ──

type AddMemberRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
	Role   string `json:"role" binding:"required,oneof=MEMBER"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=OWNER MEMBER"`
}

// ── Responses ──

type MemberResponse struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id"`
	Role        string    `json:"role"`
	Email       string    `json:"email"`
	FullName    *string   `json:"full_name,omitempty"`
	AvatarKey   *string   `json:"avatar_key,omitempty"`
	UserStatus  string    `json:"user_status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
