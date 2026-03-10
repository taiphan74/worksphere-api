package dto

import (
	"time"

	"worksphere-api/internal/user"
)

type UserResponse struct {
	ID                string     `json:"id"`
	Email             string     `json:"email"`
	FullName          *string    `json:"full_name"`
	Status            string     `json:"status"`
	PasswordChangedAt *time.Time `json:"password_changed_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func NewUserResponse(user user.User) UserResponse {
	return UserResponse{
		ID:                user.ID,
		Email:             user.Email,
		FullName:          user.FullName,
		Status:            user.Status,
		PasswordChangedAt: user.PasswordChangedAt,
		CreatedAt:         user.CreatedAt,
		UpdatedAt:         user.UpdatedAt,
	}
}

func NewUserListResponse(users []user.User) []UserResponse {
	items := make([]UserResponse, 0, len(users))
	for _, user := range users {
		items = append(items, NewUserResponse(user))
	}

	return items
}
