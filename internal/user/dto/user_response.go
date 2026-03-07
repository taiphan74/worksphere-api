package dto

import (
	"time"

	db "worksphere-api/internal/database/sqlc"
)

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	CreatedAt time.Time `json:"created_at"`
}

func NewUserResponse(user db.User) UserResponse {
	return UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		FullName:  user.FullName,
		CreatedAt: user.CreatedAt.Time,
	}
}

func NewUserListResponse(users []db.User) []UserResponse {
	items := make([]UserResponse, 0, len(users))
	for _, user := range users {
		items = append(items, NewUserResponse(user))
	}

	return items
}
