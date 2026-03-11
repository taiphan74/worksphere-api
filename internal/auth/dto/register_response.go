package dto

import (
	"time"

	"worksphere-api/internal/user"
)

type RegisterResponse struct {
	AccessToken           string           `json:"access_token"`
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

func NewRegisterResponse(accessToken string, user user.User, verificationEmailSent bool) RegisterResponse {
	return RegisterResponse{
		AccessToken:           accessToken,
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
