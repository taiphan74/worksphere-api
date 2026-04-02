package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "worksphere-api/internal/database/sqlc"
)

type UserSystemRoleRepository interface {
	AssignRole(ctx context.Context, userID uuid.UUID, roleID int64, assignedBy *uuid.UUID) error
}

type userSystemRoleRepository struct {
	queries *db.Queries
}

func NewUserSystemRoleRepository(queries *db.Queries) UserSystemRoleRepository {
	return &userSystemRoleRepository{queries: queries}
}

func (r *userSystemRoleRepository) AssignRole(ctx context.Context, userID uuid.UUID, roleID int64, assignedBy *uuid.UUID) error {
	params := db.AssignSystemRoleToUserParams{
		UserID:     userID,
		RoleID:     roleID,
		AssignedBy: pgtype.UUID{Valid: false},
	}
	if assignedBy != nil {
		params.AssignedBy = pgtype.UUID{
			Bytes: *assignedBy,
			Valid: true,
		}
	}

	return r.queries.AssignSystemRoleToUser(ctx, params)
}
