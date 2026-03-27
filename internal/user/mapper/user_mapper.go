package mapper

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"worksphere-api/internal/user"
	"worksphere-api/internal/user/dto"
	"worksphere-api/pkg/mapper"
)

// ToUser converts database row fields to User domain model.
// Using individual fields to support various SQLC Row types (CreateUserRow, GetUserRow, etc.)
func ToUser(
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

func ToUserResponse(user user.User) dto.UserResponse {
	return dto.UserResponse{
		ID:                user.ID,
		Email:             user.Email,
		FullName:          user.FullName,
		IsVerified:        user.IsVerified,
		Status:            user.Status,
		PasswordChangedAt: user.PasswordChangedAt,
		CreatedAt:         user.CreatedAt,
		UpdatedAt:         user.UpdatedAt,
	}
}

func ToUserListResponse(users []user.User) []dto.UserResponse {
	items := make([]dto.UserResponse, 0, len(users))
	for _, u := range users {
		items = append(items, ToUserResponse(u))
	}

	return items
}
