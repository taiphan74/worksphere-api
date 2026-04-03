package dto

import (
	"encoding/json"
	"time"
)

type CreateTaskRequest struct {
	Title        string     `json:"title" binding:"required,max=255"`
	Description  *string    `json:"description"`
	Status       string     `json:"status" binding:"omitempty,oneof=TODO IN_PROGRESS IN_REVIEW DONE"`
	Priority     string     `json:"priority" binding:"omitempty,oneof=LOW MEDIUM HIGH URGENT"`
	DueAt        *time.Time `json:"due_at"`
	SprintPoints *int32     `json:"sprint_points"`
	AssigneeID   *string    `json:"assignee_id"`
	ParentID     *string    `json:"parent_id"`
}

type ListTasksRequest struct {
	Status         string `form:"status"`
	AssigneeID     string `form:"assignee_id"`
	ParentID       string `form:"parent_id"`
	IncludeDeleted bool   `form:"include_deleted"`
	DueFrom        string `form:"due_from"`
	DueTo          string `form:"due_to"`
}

type UpdateTaskRequest struct {
	Title        *string    `json:"title"`
	Description  *string    `json:"description"`
	Status       *string    `json:"status"`
	Priority     *string    `json:"priority"`
	DueAt        *time.Time `json:"due_at"`
	SprintPoints *int32     `json:"sprint_points"`
	AssigneeID   *string    `json:"assignee_id"`
	ParentID     *string    `json:"parent_id"`

	HasTitle        bool `json:"-"`
	HasDescription  bool `json:"-"`
	HasStatus       bool `json:"-"`
	HasPriority     bool `json:"-"`
	HasDueAt        bool `json:"-"`
	HasSprintPoints bool `json:"-"`
	HasAssigneeID   bool `json:"-"`
	HasParentID     bool `json:"-"`
}

func (r *UpdateTaskRequest) UnmarshalJSON(data []byte) error {
	type Alias UpdateTaskRequest

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	*r = UpdateTaskRequest(alias)
	_, r.HasTitle = payload["title"]
	_, r.HasDescription = payload["description"]
	_, r.HasStatus = payload["status"]
	_, r.HasPriority = payload["priority"]
	_, r.HasDueAt = payload["due_at"]
	_, r.HasSprintPoints = payload["sprint_points"]
	_, r.HasAssigneeID = payload["assignee_id"]
	_, r.HasParentID = payload["parent_id"]

	return nil
}

func (r UpdateTaskRequest) HasChanges() bool {
	return r.HasTitle || r.HasDescription || r.HasStatus || r.HasPriority || r.HasDueAt || r.HasSprintPoints || r.HasAssigneeID || r.HasParentID
}

type TaskResponse struct {
	ID           string     `json:"id"`
	WorkspaceID  string     `json:"workspace_id"`
	CreatorID    string     `json:"creator_id"`
	AssigneeID   *string    `json:"assignee_id,omitempty"`
	Title        string     `json:"title"`
	Description  *string    `json:"description,omitempty"`
	Status       string     `json:"status"`
	Priority     string     `json:"priority"`
	DueAt        *time.Time `json:"due_at,omitempty"`
	SprintPoints *int32     `json:"sprint_points,omitempty"`
	ParentID     *string    `json:"parent_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}
