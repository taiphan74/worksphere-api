package dto

import (
	"time"

)

type UserResponse struct {
	ID                string     `json:"id"`
	Email             string     `json:"email"`
	FullName          *string    `json:"full_name"`
	IsVerified        bool       `json:"is_verified"`
	Status            string     `json:"status"`
	PasswordChangedAt *time.Time `json:"password_changed_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
