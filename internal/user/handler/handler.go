package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"worksphere-api/internal/user/dto"
	"worksphere-api/internal/user/service"
	apperrors "worksphere-api/pkg/errors"
	"worksphere-api/pkg/response"
)

type UserHandler struct {
	service service.UserService
}

func NewUserHandler(service service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("", h.CreateUser)
	group.GET("/:id", h.GetUser)
	group.GET("", h.ListUsers)
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

	user, err := h.service.CreateUser(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusCreated, dto.NewUserResponse(user), "success")
}

func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "invalid user id"))
		return
	}

	user, err := h.service.GetUserByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, dto.NewUserResponse(user), "success")
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

	response.Success(c, http.StatusOK, dto.NewUserListResponse(users), "success")
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "invalid user id"))
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "invalid request body"))
		return
	}

	user, err := h.service.UpdateUser(c.Request.Context(), id, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, dto.NewUserResponse(user), "success")
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "invalid user id"))
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
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_INPUT", "invalid user id"))
		return
	}

	user, err := h.service.RestoreUser(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, dto.NewUserResponse(user), "success")
}
