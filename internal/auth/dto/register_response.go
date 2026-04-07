package dto

import (
	"time"

	"worksphere-api/internal/user"
)

type RegisterResponse struct {
	VerificationEmailSent bool             `json:"verification_email_sent"`
	User                  RegisterUserData `json:"user"`
}

type RegisterUserData struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	IsVerified bool      `json:"is_verified"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func NewRegisterResponse(user user.User, verificationEmailSent bool) RegisterResponse {
	return RegisterResponse{
		VerificationEmailSent: verificationEmailSent,
		User: RegisterUserData{
			ID:         user.ID,
			Email:      user.Email,
			IsVerified: user.IsVerified,
			CreatedAt:  user.CreatedAt,
			UpdatedAt:  user.UpdatedAt,
		},
	}
}
