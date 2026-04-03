package task

import (
	"net/http"

	apperrors "worksphere-api/pkg/errors"
)

var (
	ErrTaskNotFound               = apperrors.New(http.StatusNotFound, "TASK_NOT_FOUND", "task not found")
	ErrInvalidTaskID              = apperrors.New(http.StatusBadRequest, "INVALID_TASK_ID", "invalid task ID")
	ErrTaskTitleRequired          = apperrors.New(http.StatusBadRequest, "TASK_TITLE_REQUIRED", "task title is required")
	ErrTaskTitleTooLong           = apperrors.New(http.StatusBadRequest, "TASK_TITLE_TOO_LONG", "task title cannot exceed 255 characters")
	ErrInvalidTaskStatus          = apperrors.New(http.StatusBadRequest, "INVALID_TASK_STATUS", "invalid task status")
	ErrInvalidTaskPriority        = apperrors.New(http.StatusBadRequest, "INVALID_TASK_PRIORITY", "invalid task priority")
	ErrInvalidAssigneeID          = apperrors.New(http.StatusBadRequest, "INVALID_ASSIGNEE_ID", "invalid assignee ID")
	ErrInvalidParentTaskID        = apperrors.New(http.StatusBadRequest, "INVALID_PARENT_TASK_ID", "invalid parent task ID")
	ErrInvalidTaskFilters         = apperrors.New(http.StatusBadRequest, "INVALID_TASK_FILTERS", "invalid task filters")
	ErrAssigneeNotWorkspaceMember = apperrors.New(http.StatusBadRequest, "ASSIGNEE_NOT_WORKSPACE_MEMBER", "assignee must be a workspace member")
	ErrParentTaskNotFound         = apperrors.New(http.StatusBadRequest, "PARENT_TASK_NOT_FOUND", "parent task not found in this workspace")
	ErrParentTaskSelfReference    = apperrors.New(http.StatusBadRequest, "PARENT_TASK_SELF_REFERENCE", "task cannot be its own parent")
	ErrInvalidSprintPoints        = apperrors.New(http.StatusBadRequest, "INVALID_SPRINT_POINTS", "sprint points must be zero or greater")
	ErrForbiddenAccess            = apperrors.New(http.StatusForbidden, "FORBIDDEN_ACCESS", "you don't have access to this resource")
	ErrInternalServer             = apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "something went wrong")
)
