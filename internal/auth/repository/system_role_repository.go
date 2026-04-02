package repository

import (
	"context"

	db "worksphere-api/internal/database/sqlc"
)

type SystemRoleRepository interface {
	GetByCode(ctx context.Context, code string) (db.SystemRole, error)
}

type systemRoleRepository struct {
	queries *db.Queries
}

func NewSystemRoleRepository(queries *db.Queries) SystemRoleRepository {
	return &systemRoleRepository{queries: queries}
}

func (r *systemRoleRepository) GetByCode(ctx context.Context, code string) (db.SystemRole, error) {
	return r.queries.GetSystemRoleByCode(ctx, code)
}
