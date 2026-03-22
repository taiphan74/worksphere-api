package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"worksphere-api/internal/middleware"
	"worksphere-api/internal/workspace/dto"
	"worksphere-api/internal/workspace/service"
	apperrors "worksphere-api/pkg/errors"
	"worksphere-api/pkg/response"
)

type MemberHandler struct {
	service service.MemberService
}

func NewMemberHandler(service service.MemberService) *MemberHandler {
	return &MemberHandler{service: service}
}

// AddMember handles POST /workspaces/:id/members
func (h *MemberHandler) AddMember(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_ID", "invalid workspace id"))
		return
	}

	var req dto.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "please provide a valid user_id and role (MEMBER)"))
		return
	}

	res, err := h.service.AddMember(c.Request.Context(), userID, workspaceID, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusCreated, res, "member added successfully")
}

// ListMembers handles GET /workspaces/:id/members
func (h *MemberHandler) ListMembers(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_ID", "invalid workspace id"))
		return
	}

	res, err := h.service.ListMembers(c.Request.Context(), userID, workspaceID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, res, "members retrieved successfully")
}

// GetMember handles GET /workspaces/:id/members/:userId
func (h *MemberHandler) GetMember(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_ID", "invalid workspace id"))
		return
	}

	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_USER_ID", "invalid user id"))
		return
	}

	res, err := h.service.GetMember(c.Request.Context(), userID, workspaceID, targetUserID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, res, "member retrieved successfully")
}

// UpdateMemberRole handles PATCH /workspaces/:id/members/:userId
func (h *MemberHandler) UpdateMemberRole(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_ID", "invalid workspace id"))
		return
	}

	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_USER_ID", "invalid user id"))
		return
	}

	var req dto.UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "please provide a valid role (OWNER, MEMBER)"))
		return
	}

	res, err := h.service.UpdateMemberRole(c.Request.Context(), userID, workspaceID, targetUserID, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, res, "member role updated successfully")
}

// RemoveMember handles DELETE /workspaces/:id/members/:userId
func (h *MemberHandler) RemoveMember(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	workspaceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_ID", "invalid workspace id"))
		return
	}

	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_USER_ID", "invalid user id"))
		return
	}

	if err := h.service.RemoveMember(c.Request.Context(), userID, workspaceID, targetUserID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, nil, "member removed successfully")
}
