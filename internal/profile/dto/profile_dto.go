package dto

import "time"

type ProfileResponse struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	FullName   *string   `json:"full_name"`
	AvatarKey  *string   `json:"avatar_key,omitempty"`
	Phone      *string   `json:"phone,omitempty"`
	JobTitle   *string   `json:"job_title,omitempty"`
	IsVerified bool      `json:"is_verified"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type UpdateProfileRequest struct {
	FullName *string `json:"full_name" binding:"omitempty,max=150"`
	Phone    *string `json:"phone" binding:"omitempty,max=20"`
	JobTitle *string `json:"job_title" binding:"omitempty,max=100"`
}

type ChangePasswordRequest struct {
	CurrentPassword    string `json:"current_password" binding:"required"`
	NewPassword        string `json:"new_password" binding:"required,min=8"`
	ConfirmNewPassword string `json:"confirm_new_password" binding:"required,eqfield=NewPassword"`
}

type AvatarUploadURLRequest struct {
	FileName    string `json:"file_name" binding:"required"`
	ContentType string `json:"content_type" binding:"required"`
	Size        int64  `json:"size" binding:"required,gt=0"`
}

type AvatarUploadURLResponse struct {
	ObjectKey       string            `json:"object_key"`
	UploadURL       string            `json:"upload_url"`
	Method          string            `json:"method"`
	ExpiresIn       int               `json:"expires_in"`
	RequiredHeaders map[string]string `json:"required_headers"`
}

type AvatarConfirmRequest struct {
	ObjectKey string `json:"object_key" binding:"required"`
}

type AvatarViewURLResponse struct {
	ViewURL   string `json:"view_url"`
	ExpiresIn int    `json:"expires_in"`
}
