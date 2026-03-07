package dto

import (
	"time"

	"worksphere-api/internal/user"
)

type AuthResponse struct {
	AccessToken string       `json:"access_token"`
	User        AuthUserData `json:"user"`
}

type AuthUserData struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewAuthResponse(accessToken string, user user.User) AuthResponse {
	return AuthResponse{
		AccessToken: accessToken,
		User:        NewAuthUserData(user),
	}
}

func NewAuthUserData(user user.User) AuthUserData {
	return AuthUserData{
		ID:        user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
