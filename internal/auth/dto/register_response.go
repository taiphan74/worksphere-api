package dto

import (
	"time"

	"worksphere-api/internal/user"
)

type RegisterResponse struct {
	AccessToken string           `json:"access_token"`
	User        RegisterUserData `json:"user"`
}

type RegisterUserData struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewRegisterResponse(accessToken string, user user.User) RegisterResponse {
	return RegisterResponse{
		AccessToken: accessToken,
		User: RegisterUserData{
			ID:        user.ID,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	}
}
