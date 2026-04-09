package dto

type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
	VerificationUrl string `json:"verificationUrl" binding:"omitempty,url"`
}
