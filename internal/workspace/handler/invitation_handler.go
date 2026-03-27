package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"worksphere-api/internal/middleware"
	"worksphere-api/internal/workspace/dto"
	"worksphere-api/internal/workspace/service"
	"worksphere-api/internal/workspace"
	"worksphere-api/pkg/response"
	"worksphere-api/pkg/validation"
)

type InvitationHandler struct {
	service service.InvitationService
}

func NewInvitationHandler(service service.InvitationService) *InvitationHandler {
	return &InvitationHandler{service: service}
}

// SendInvitation handles POST /workspaces/:id/invitations
func (h *InvitationHandler) SendInvitation(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, workspace.ErrInvalidWorkspaceID)
		return
	}

	var req dto.SendInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	res, err := h.service.SendInvitation(c.Request.Context(), userID, workspaceID, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusCreated, res, "invitation sent successfully")
}

// GetInvitation handles GET /invitations/:invitationId
func (h *InvitationHandler) GetInvitation(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	invitationID, err := uuid.Parse(c.Param("invitationId"))
	if err != nil {
		response.Error(c, workspace.ErrInvalidInvitationID)
		return
	}

	res, err := h.service.GetInvitation(c.Request.Context(), userID, invitationID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, res, "invitation retrieved successfully")
}

// ListInvitations handles GET /workspaces/:id/invitations
func (h *InvitationHandler) ListInvitations(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, workspace.ErrInvalidWorkspaceID)
		return
	}

	res, err := h.service.ListInvitations(c.Request.Context(), userID, workspaceID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, res, "invitations retrieved successfully")
}

// AcceptInvitation handles POST /invitations/accept
func (h *InvitationHandler) AcceptInvitation(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.AcceptInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	if err := h.service.AcceptInvitation(c.Request.Context(), req.Token, userID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, nil, "invitation accepted successfully")
}

// DeclineInvitation handles POST /invitations/decline
func (h *InvitationHandler) DeclineInvitation(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	if err := h.service.DeclineInvitation(c.Request.Context(), req.Token, userID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, nil, "invitation declined successfully")
}

// CancelInvitation handles DELETE /invitations/:invitationId
func (h *InvitationHandler) CancelInvitation(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	invitationID, err := uuid.Parse(c.Param("invitationId"))
	if err != nil {
		response.Error(c, workspace.ErrInvalidInvitationID)
		return
	}

	if err := h.service.CancelInvitation(c.Request.Context(), userID, invitationID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, nil, "invitation cancelled successfully")
}

// ResendInvitation handles POST /invitations/:invitationId/resend
func (h *InvitationHandler) ResendInvitation(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	invitationID, err := uuid.Parse(c.Param("invitationId"))
	if err != nil {
		response.Error(c, workspace.ErrInvalidInvitationID)
		return
	}

	if err := h.service.ResendInvitation(c.Request.Context(), userID, invitationID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, nil, "invitation resent successfully")
}
