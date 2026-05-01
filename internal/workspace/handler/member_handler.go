package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"worksphere-api/internal/middleware"
	"worksphere-api/internal/router"
	"worksphere-api/internal/workspace"
	"worksphere-api/internal/workspace/dto"
	"worksphere-api/internal/workspace/service"
	"worksphere-api/pkg/response"
	"worksphere-api/pkg/validation"
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
		response.Error(c, workspace.ErrInvalidWorkspaceID)
		return
	}

	var req dto.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.HandleValidationError(c, err)
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
		response.Error(c, workspace.ErrInvalidWorkspaceID)
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
		response.Error(c, workspace.ErrInvalidWorkspaceID)
		return
	}

	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.Error(c, workspace.ErrInvalidUserID)
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
		response.Error(c, workspace.ErrInvalidWorkspaceID)
		return
	}

	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.Error(c, workspace.ErrInvalidUserID)
		return
	}

	var req dto.UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.HandleValidationError(c, err)
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
		response.Error(c, workspace.ErrInvalidWorkspaceID)
		return
	}

	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.Error(c, workspace.ErrInvalidUserID)
		return
	}

	if err := h.service.RemoveMember(c.Request.Context(), userID, workspaceID, targetUserID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, nil, "member removed successfully")
}

// RegisterRoutes đăng ký các route cho workspace member module.
func (h *MemberHandler) RegisterRoutes(groups router.Groups, _ ...gin.HandlerFunc) {
	members := groups.Protected.Group("/workspaces/:id/members")
	members.POST("", h.AddMember)
	members.GET("", h.ListMembers)
	members.GET("/:userId", h.GetMember)
	members.PATCH("/:userId", h.UpdateMemberRole)
	members.DELETE("/:userId", h.RemoveMember)
}
