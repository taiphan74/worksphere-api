package dto

import (
	"errors"
	"time"
)

var ErrInvalidSlug = errors.New("invalid slug: must contain at least one alphanumeric character")

type CreateWorkspaceRequest struct {
	Name string `json:"name" binding:"required,max=150"`
}

type UpdateWorkspaceRequest struct {
	Name *string `json:"name" binding:"omitempty,min=1,max=150"`
}

type WorkspaceResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
