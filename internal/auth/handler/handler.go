package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"worksphere-api/internal/auth/dto"
	"worksphere-api/internal/auth/service"
	"worksphere-api/internal/middleware"
	apperrors "worksphere-api/pkg/errors"
	"worksphere-api/pkg/response"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) RegisterPublicRoutes(group *gin.RouterGroup) {
	group.POST("/register", h.Register)
	group.POST("/login", h.Login)
}

func (h *AuthHandler) RegisterProtectedRoutes(group *gin.RouterGroup) {
	group.GET("/me", h.Me)
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}

	user, token, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusCreated, dto.NewAuthResponse(token, user), "success")
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.New(http.StatusBadRequest, "INVALID_REQUEST", "invalid request body"))
		return
	}

	user, token, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, dto.NewAuthResponse(token, user), "success")
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, err := middleware.GetCurrentUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	user, err := h.service.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, dto.NewMeResponse(user), "success")
}
