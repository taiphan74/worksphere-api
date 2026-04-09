package dto

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
	ResetUrl string `json:"resetUrl" binding:"omitempty,url"`
}
