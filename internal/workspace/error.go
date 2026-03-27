package workspace

import (
	"net/http"

	apperrors "worksphere-api/pkg/errors"
)

var (
	// Workspace errors
	ErrWorkspaceNotFound     = apperrors.New(http.StatusNotFound, "WORKSPACE_NOT_FOUND", "workspace not found")
	ErrWorkspaceNameRequired = apperrors.New(http.StatusBadRequest, "WORKSPACE_NAME_REQUIRED", "workspace name is required")
	ErrWorkspaceNameTooLong  = apperrors.New(http.StatusBadRequest, "WORKSPACE_NAME_TOO_LONG", "workspace name cannot exceed 150 characters")
	ErrInvalidWorkspaceName  = apperrors.New(http.StatusBadRequest, "INVALID_WORKSPACE_NAME", "workspace name contains invalid characters")
	ErrSlugAlreadyExists     = apperrors.New(http.StatusConflict, "SLUG_ALREADY_EXISTS", "workspace slug already exists")
	ErrInvalidWorkspaceID    = apperrors.New(http.StatusBadRequest, "INVALID_WORKSPACE_ID", "invalid workspace ID")
	ErrInvalidInvitationID   = apperrors.New(http.StatusBadRequest, "INVALID_INVITATION_ID", "invalid invitation ID")

	// Member errors
	ErrNotAMember            = apperrors.New(http.StatusForbidden, "NOT_A_MEMBER", "you are not a member of this workspace")
	ErrAlreadyMember         = apperrors.New(http.StatusConflict, "ALREADY_MEMBER", "user is already a member of this workspace")
	ErrMemberNotFound        = apperrors.New(http.StatusNotFound, "MEMBER_NOT_FOUND", "member not found")
	ErrInvalidRole           = apperrors.New(http.StatusBadRequest, "INVALID_ROLE", "invalid role")
	ErrInvalidUserID         = apperrors.New(http.StatusBadRequest, "INVALID_USER_ID", "invalid user ID")
	ErrCannotRemoveLastOwner = apperrors.New(http.StatusBadRequest, "CANNOT_REMOVE_LAST_OWNER", "cannot remove the last owner from workspace")
	ErrCannotDemoteLastOwner = apperrors.New(http.StatusBadRequest, "CANNOT_DEMOTE_LAST_OWNER", "cannot demote the last owner")

	// Invitation errors
	ErrInvitationNotFound    = apperrors.New(http.StatusNotFound, "INVITATION_NOT_FOUND", "invitation not found")
	ErrInvitationExpired     = apperrors.New(http.StatusBadRequest, "INVITATION_EXPIRED", "invitation has expired")
	ErrAlreadyInvited        = apperrors.New(http.StatusConflict, "ALREADY_INVITED", "user has already been invited")
	ErrInvalidInvitationToken = apperrors.New(http.StatusBadRequest, "INVALID_INVITATION_TOKEN", "invalid invitation token")
	ErrNotWorkspaceOwner     = apperrors.New(http.StatusForbidden, "NOT_WORKSPACE_OWNER", "only workspace owners can send invitations")
	ErrInvitationSendFailed  = apperrors.New(http.StatusInternalServerError, "INVITATION_SEND_FAILED", "failed to send invitation")

	// Permission errors
	ErrForbiddenAccess       = apperrors.New(http.StatusForbidden, "FORBIDDEN_ACCESS", "you don't have access to this resource")
	ErrInsufficientPermissions = apperrors.New(http.StatusForbidden, "INSUFFICIENT_PERMISSIONS", "you don't have permission to perform this action")
	ErrOnlyOwnersCanUpdate   = apperrors.New(http.StatusForbidden, "ONLY_OWNERS_CAN_UPDATE", "only workspace owners can update this workspace")

	// Generic errors
	ErrInvalidInput          = apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "invalid input")
	ErrInvalidRequest        = apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	ErrInternalServer        = apperrors.New(http.StatusInternalServerError, "INTERNAL_ERROR", "something went wrong")
)
