package user

import "time"

type User struct {
	ID                string
	Email             string
	FullName          *string
	Username          *string
	AvatarURL         *string
	Phone             *string
	JobTitle          *string
	Status            string
	PasswordChangedAt *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}
