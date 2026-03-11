package user

import (
	"time"
)

type User struct {
	ID                string
	Email             string
	FullName          *string
	IsVerified        bool
	Status            string
	PasswordChangedAt *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}
