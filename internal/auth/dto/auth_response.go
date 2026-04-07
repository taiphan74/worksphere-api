package dto

import (
	"time"

	"worksphere-api/internal/user"
)

type AuthResponse struct {
	User AuthUserData `json:"user"`
}

type AuthUserData struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	FullName   *string   `json:"full_name"`
	IsVerified bool      `json:"is_verified"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func NewAuthResponse(user user.User) AuthResponse {
	return AuthResponse{
		User: NewAuthUserData(user),
	}
}

func NewAuthUserData(user user.User) AuthUserData {
	return AuthUserData{
		ID:         user.ID,
		Email:      user.Email,
		FullName:   user.FullName,
		IsVerified: user.IsVerified,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
	}
}
