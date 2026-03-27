package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"worksphere-api/internal/auth/dto"
	"worksphere-api/internal/auth/service"
	"worksphere-api/internal/middleware"
	"worksphere-api/pkg/response"
	"worksphere-api/pkg/validation"
)

type AuthHandler struct {
	service     service.AuthService
	rateLimiter resendVerificationRateLimiter
}

type resendVerificationRateLimiter interface {
	AllowResendVerificationIP(ctx context.Context, ip string) (bool, int, error)
}

func NewAuthHandler(service service.AuthService, rateLimiter resendVerificationRateLimiter) *AuthHandler {
	return &AuthHandler{
		service:     service,
		rateLimiter: rateLimiter,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	result, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusCreated, dto.NewRegisterResponse(result.AccessToken, result.User, result.VerificationEmailSent), "success")
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	user, token, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, dto.NewAuthResponse(token, user), "success")
}

func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	var req dto.GoogleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	user, token, err := h.service.LoginWithGoogle(c.Request.Context(), req)
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

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	user, err := h.service.VerifyEmail(c.Request.Context(), c.Query("token"))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, dto.VerifyEmailResponse{Verified: user.IsVerified}, "success")
}

func (h *AuthHandler) ResendVerification(c *gin.Context) {
	var req dto.ResendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	if h.rateLimiter != nil {
		allowed, retryAfterSeconds, err := h.rateLimiter.AllowResendVerificationIP(c.Request.Context(), c.ClientIP())
		if err == nil && !allowed {
			response.ErrorWithRetryAfter(c, http.StatusTooManyRequests, "RATE_LIMITED", "too many resend verification attempts", retryAfterSeconds)
			return
		}
	}

	result, err := h.service.ResendVerification(c.Request.Context(), req.Email)
	if err != nil {
		response.Error(c, err)
		return
	}

	if result.RateLimited {
		response.ErrorWithRetryAfter(c, http.StatusTooManyRequests, "RATE_LIMITED", "too many resend verification attempts", result.RetryAfterSeconds)
		return
	}

	response.Success(c, http.StatusOK, nil, "if the email exists, a verification email has been sent")
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, nil, "if the email exists, a reset link has been sent")
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.HandleValidationError(c, err)
		return
	}

	if err := h.service.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, http.StatusOK, nil, "password reset successfully")
}
