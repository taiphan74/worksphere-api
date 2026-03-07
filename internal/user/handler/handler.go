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
	group.GET("", h.ListUsers)
	group.GET("/:id", h.GetUser)
	group.POST("", h.CreateUser)
	group.PATCH("/:id", h.UpdateUser)
	group.DELETE("/:id", h.DeleteUser)
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
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
	user, err := h.service.GetUserByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, dto.NewUserResponse(user), "success")
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	users, err := h.service.ListUsers(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, dto.NewUserListResponse(users), "success")
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}

	user, err := h.service.UpdateUser(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, dto.NewUserResponse(user), "success")
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	if err := h.service.DeleteUser(c.Request.Context(), c.Param("id")); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, nil, "success")
}
