package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"worksphere-api/internal/middleware"
	"worksphere-api/internal/router"
	"worksphere-api/internal/task"
	"worksphere-api/internal/task/dto"
	"worksphere-api/internal/task/service"
	"worksphere-api/internal/workspace"
	"worksphere-api/pkg/response"
	"worksphere-api/pkg/validation"
)

type TaskHandler struct {
	service service.TaskService
}

func NewTaskHandler(service service.TaskService) *TaskHandler {
	return &TaskHandler{service: service}
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
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

	var req dto.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	req.Title = strings.TrimSpace(req.Title)

	res, err := h.service.CreateTask(c.Request.Context(), userID, workspaceID, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusCreated, res, "task created successfully")
}

func (h *TaskHandler) ListTasks(c *gin.Context) {
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

	var req dto.ListTasksRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, task.ErrInvalidTaskFilters)
		return
	}

	res, err := h.service.ListTasks(c.Request.Context(), userID, workspaceID, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, res, "tasks retrieved successfully")
}

func (h *TaskHandler) GetTask(c *gin.Context) {
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

	taskID, err := uuid.Parse(c.Param("taskId"))
	if err != nil {
		response.Error(c, task.ErrInvalidTaskID)
		return
	}

	res, err := h.service.GetTaskByID(c.Request.Context(), userID, workspaceID, taskID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, res, "task retrieved successfully")
}

func (h *TaskHandler) UpdateTask(c *gin.Context) {
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

	taskID, err := uuid.Parse(c.Param("taskId"))
	if err != nil {
		response.Error(c, task.ErrInvalidTaskID)
		return
	}

	var req dto.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	res, err := h.service.UpdateTask(c.Request.Context(), userID, workspaceID, taskID, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, res, "task updated successfully")
}

func (h *TaskHandler) DeleteTask(c *gin.Context) {
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

	taskID, err := uuid.Parse(c.Param("taskId"))
	if err != nil {
		response.Error(c, task.ErrInvalidTaskID)
		return
	}

	if err := h.service.DeleteTask(c.Request.Context(), userID, workspaceID, taskID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, nil, "task deleted successfully")
}

// RegisterRoutes đăng ký các route cho task module.
func (h *TaskHandler) RegisterRoutes(groups router.Groups, _ ...gin.HandlerFunc) {
	tasks := groups.Protected.Group("/workspaces/:id/tasks")
	tasks.POST("", h.CreateTask)
	tasks.GET("", h.ListTasks)
	tasks.GET("/:taskId", h.GetTask)
	tasks.PATCH("/:taskId", h.UpdateTask)
	tasks.DELETE("/:taskId", h.DeleteTask)
}
