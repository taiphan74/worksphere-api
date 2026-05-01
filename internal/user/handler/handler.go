package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"worksphere-api/internal/middleware"
	"worksphere-api/internal/router"
	"worksphere-api/internal/user"
	"worksphere-api/internal/user/dto"
	"worksphere-api/internal/user/mapper"
	"worksphere-api/internal/user/service"
	"worksphere-api/pkg/response"
	"worksphere-api/pkg/validation"
)

type UserHandler struct {
	service service.UserService
}

func NewUserHandler(service service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// RegisterRoutes đăng ký các route cho user module (admin-only).
func (h *UserHandler) RegisterRoutes(groups router.Groups, _ ...gin.HandlerFunc) {
	adminGroup := groups.Protected.Group("/users")
	adminGroup.Use(middleware.RequireRoles("ADMIN", "SUPER_ADMIN"))
	adminGroup.POST("", h.CreateUser)
	adminGroup.GET("/:id", h.GetUser)
	adminGroup.GET("", h.ListUsers)
	adminGroup.PATCH("/:id", h.UpdateUser)
	adminGroup.DELETE("/:id", h.DeleteUser)
	adminGroup.PATCH("/:id/restore", h.RestoreUser)
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	user, err := h.service.CreateUser(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusCreated, mapper.ToUserResponse(user), "success")
}

func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, user.ErrInvalidInput)
		return
	}

	user, err := h.service.GetUserByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, mapper.ToUserResponse(user), "success")
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	var status *string
	if s := c.Query("status"); s != "" {
		status = &s
	}

	var search *string
	if s := c.Query("search"); s != "" {
		search = &s
	}

	users, err := h.service.ListUsers(c.Request.Context(), status, search)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, mapper.ToUserListResponse(users), "success")
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, user.ErrInvalidInput)
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	user, err := h.service.UpdateUser(c.Request.Context(), id, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, mapper.ToUserResponse(user), "success")
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, user.ErrInvalidInput)
		return
	}

	err = h.service.DeleteUser(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, nil, "success")
}

func (h *UserHandler) RestoreUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, user.ErrInvalidInput)
		return
	}

	user, err := h.service.RestoreUser(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, mapper.ToUserResponse(user), "success")
}
