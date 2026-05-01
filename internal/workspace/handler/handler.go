package handler

import (
	"net/http"
	"strings"

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

type WorkspaceHandler struct {
	service service.WorkspaceService
}

func NewWorkspaceHandler(service service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{service: service}
}

func (h *WorkspaceHandler) CreateWorkspace(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		response.Error(c, workspace.ErrWorkspaceNameRequired)
		return
	}

	res, err := h.service.CreateWorkspace(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusCreated, res, "workspace created successfully")
}

func (h *WorkspaceHandler) ListWorkspaces(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	res, err := h.service.ListWorkspacesByUser(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, res, "workspaces retrieved successfully")
}

func (h *WorkspaceHandler) GetWorkspaceByID(c *gin.Context) {
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

	res, err := h.service.GetWorkspaceByID(c.Request.Context(), userID, workspaceID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, res, "workspace retrieved successfully")
}

func (h *WorkspaceHandler) GetWorkspaceBySlug(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	slug := strings.TrimSpace(c.Param("slug"))
	if slug == "" {
		response.Error(c, workspace.ErrWorkspaceNameRequired)
		return
	}

	res, err := h.service.GetWorkspaceBySlug(c.Request.Context(), userID, slug)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, res, "workspace retrieved successfully")
}

func (h *WorkspaceHandler) UpdateWorkspace(c *gin.Context) {
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

	var req dto.UpdateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			response.Error(c, workspace.ErrWorkspaceNameRequired)
			return
		}
		req.Name = &trimmed
	}

	res, err := h.service.UpdateWorkspace(c.Request.Context(), userID, workspaceID, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, res, "workspace updated successfully")
}

func (h *WorkspaceHandler) DeleteWorkspace(c *gin.Context) {
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

	if err := h.service.DeleteWorkspace(c.Request.Context(), userID, workspaceID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, nil, "workspace deleted successfully")
}

// RegisterRoutes đăng ký các route cho workspace module.
func (h *WorkspaceHandler) RegisterRoutes(groups router.Groups, _ ...gin.HandlerFunc) {
	workspaces := groups.Protected.Group("/workspaces")
	workspaces.POST("", h.CreateWorkspace)
	workspaces.GET("", h.ListWorkspaces)
	workspaces.GET("/:id", h.GetWorkspaceByID)
	workspaces.GET("/slug/:slug", h.GetWorkspaceBySlug)
	workspaces.PATCH("/:id", h.UpdateWorkspace)
	workspaces.DELETE("/:id", h.DeleteWorkspace)
}
