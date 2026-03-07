package dto

import (
	"time"

	db "worksphere-api/internal/database/sqlc"
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

func NewAuthResponse(accessToken string, user db.User) AuthResponse {
	return AuthResponse{
		AccessToken: accessToken,
		User:        NewAuthUserData(user),
	}
}

func NewAuthUserData(user db.User) AuthUserData {
	return AuthUserData{
		ID:        user.ID.String(),
		Email:     user.Email,
		FullName:  user.FullName,
		CreatedAt: user.CreatedAt.Time,
		UpdatedAt: user.UpdatedAt.Time,
	}
}
