package repository

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"worksphere-api/internal/user"
	"worksphere-api/pkg/mapper"
)

func toUser(
	id uuid.UUID,
	email string,
	fullName pgtype.Text,
	isVerified bool,
	status string,
	passwordChangedAt pgtype.Timestamptz,
	createdAt pgtype.Timestamptz,
	updatedAt pgtype.Timestamptz,
	deletedAt pgtype.Timestamptz,
) user.User {
	return user.User{
		ID:                id.String(),
		Email:             email,
		FullName:          mapper.TextPtr(fullName),
		IsVerified:        isVerified,
		Status:            status,
		PasswordChangedAt: mapper.TimestamptzPtr(passwordChangedAt),
		CreatedAt:         createdAt.Time,
		UpdatedAt:         updatedAt.Time,
		DeletedAt:         mapper.TimestamptzPtr(deletedAt),
	}
}
