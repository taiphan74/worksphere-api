package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"worksphere-api/internal/user/dto"
	"worksphere-api/internal/user/service"
	apperrors "worksphere-api/pkg/errors"
	"worksphere-api/pkg/response"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("", h.CreateUser)
	group.GET("", h.ListUsers)
	group.GET("/:id", h.GetUser)
	group.PATCH("/:id", h.UpdateUser)
	group.DELETE("/:id", h.DeleteUser)
	group.PATCH("/:id/restore", h.RestoreUser)
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "invalid request body"))
		return
	}

	record, err := h.service.CreateUser(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusCreated, dto.NewUserResponse(record), "success")
}

func (h *UserHandler) GetUser(c *gin.Context) {
	record, err := h.service.GetUserByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, dto.NewUserResponse(record), "success")
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	var req dto.ListUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "invalid query parameters"))
		return
	}

	records, err := h.service.ListUsers(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, dto.NewUserListResponse(records), "success")
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "invalid request body"))
		return
	}

	record, err := h.service.UpdateUser(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, dto.NewUserResponse(record), "success")
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	if err := h.service.DeleteUser(c.Request.Context(), c.Param("id")); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, nil, "success")
}

func (h *UserHandler) RestoreUser(c *gin.Context) {
	record, err := h.service.RestoreUser(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, dto.NewUserResponse(record), "success")
}
