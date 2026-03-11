package dto

import (
	"time"

	"worksphere-api/internal/user"
)

type RegisterResponse struct {
	VerificationSent bool             `json:"verification_sent"`
	User             RegisterUserData `json:"user"`
}

type RegisterUserData struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	FullName   *string   `json:"full_name"`
	IsVerified bool      `json:"is_verified"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func NewRegisterResponse(user user.User) RegisterResponse {
	return RegisterResponse{
		VerificationSent: true,
		User: RegisterUserData{
			ID:         user.ID,
			Email:      user.Email,
			FullName:   user.FullName,
			IsVerified: user.IsVerified,
			CreatedAt:  user.CreatedAt,
			UpdatedAt:  user.UpdatedAt,
		},
	}
}
