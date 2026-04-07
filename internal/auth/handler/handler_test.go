package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"worksphere-api/internal/auth"
	"worksphere-api/internal/auth/handler"
	"worksphere-api/internal/auth/mocks"
	"worksphere-api/internal/auth/service"
	"worksphere-api/internal/middleware"
	"worksphere-api/internal/user"
)

func setupAuthRouter(authService *mocks.MockAuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	authHandler := handler.NewAuthHandler(authService, nil, "development", 60, 7)

	api := router.Group("/api")

	// Public routes
	authPublic := api.Group("/auth")
	authPublic.POST("/register", authHandler.Register)
	authPublic.POST("/login", authHandler.Login)
	authPublic.GET("/verify-email", authHandler.VerifyEmail)
	authPublic.POST("/resend-verification", authHandler.ResendVerification)
	authPublic.POST("/forgot-password", authHandler.ForgotPassword)
	authPublic.POST("/reset-password", authHandler.ResetPassword)

	// Protected route (simulate middleware by manually setting user ID)
	authProtected := api.Group("/auth")
	authProtected.GET("/me", func(c *gin.Context) {
		// Simulate JWT middleware - set userID from header
		userIDStr := c.GetHeader("X-Test-User-ID")
		if userIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "authentication required"}})
			c.Abort()
			return
		}
		uid, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "INVALID_TOKEN", "message": "invalid token"}})
			c.Abort()
			return
		}
		c.Set(middleware.CurrentUserIDKey, uid)
		c.Next()
	}, authHandler.Me)

	return router
}

func TestAuthHandler_Register(t *testing.T) {
	type testCase struct {
		name               string
		reqBody            interface{}
		mockSetup          func(m *mocks.MockAuthService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name: "Success - Trả về 201 cho data hợp lệ",
			reqBody: map[string]interface{}{
				"email":    "newuser@email.com",
				"password": "validpassword123",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.On("Register", mock.Anything, mock.AnythingOfType("dto.RegisterRequest")).
					Return(service.RegisterResult{
						AccessToken:           "test-token",
						RefreshToken:          "test-refresh-token",
						User:                  user.User{ID: uuid.New().String(), Email: "newuser@email.com"},
						VerificationEmailSent: true,
					}, nil)
			},
			expectedStatusCode: http.StatusCreated,
		},
		{
			name:               "Error - Malformed JSON body",
			reqBody:            "{ invalid_json: 123 ",
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "INVALID_REQUEST",
		},
		{
			name: "Error - Thiếu email",
			reqBody: map[string]interface{}{
				"password": "validpassword123",
			},
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Thiếu password",
			reqBody: map[string]interface{}{
				"email": "test@email.com",
			},
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Password ngắn hơn 8 ký tự",
			reqBody: map[string]interface{}{
				"email":    "test@email.com",
				"password": "short",
			},
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Password dài hơn 72 ký tự",
			reqBody: map[string]interface{}{
				"email":    "long@email.com",
				"password": "a123456789b123456789c123456789d123456789e123456789f123456789g123456789h123",
			},
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Email đã tồn tại",
			reqBody: map[string]interface{}{
				"email":    "exist@email.com",
				"password": "validpassword123",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.On("Register", mock.Anything, mock.AnythingOfType("dto.RegisterRequest")).
					Return(service.RegisterResult{}, auth.ErrEmailAlreadyRegistered)
			},
			expectedStatusCode: http.StatusConflict,
			expectedErrorCode:  "EMAIL_ALREADY_REGISTERED",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockAuthService)
			tc.mockSetup(mockSvc)
			router := setupAuthRouter(mockSvc)

			var reqBodyBytes []byte
			if strBody, ok := tc.reqBody.(string); ok {
				reqBodyBytes = []byte(strBody)
			} else {
				reqBodyBytes, _ = json.Marshal(tc.reqBody)
			}

			req, _ := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatusCode, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)

			if tc.expectedErrorCode != "" {
				errResp, ok := resp["error"].(map[string]interface{})
				assert.True(t, ok, "Phản hồi lỗi thiếu block 'error'")
				if code, exists := errResp["code"]; exists {
					assert.Equal(t, tc.expectedErrorCode, code)
				}
			} else {
				assert.Nil(t, resp["error"], "Phản hồi success không nên chứa block 'error'")
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	type testCase struct {
		name               string
		reqBody            interface{}
		mockSetup          func(m *mocks.MockAuthService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name: "Success - Login thành công",
			reqBody: map[string]interface{}{
				"email":    "test@email.com",
				"password": "validpassword123",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.On("Login", mock.Anything, mock.AnythingOfType("dto.LoginRequest")).
					Return(user.User{ID: uuid.New().String(), Email: "test@email.com"}, "jwt-token", "refresh-token", nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "Error - Thiếu email",
			reqBody: map[string]interface{}{
				"password": "validpassword123",
			},
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Thiếu password",
			reqBody: map[string]interface{}{
				"email": "test@email.com",
			},
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Password dài hơn 72 ký tự",
			reqBody: map[string]interface{}{
				"email":    "test@email.com",
				"password": "a123456789b123456789c123456789d123456789e123456789f123456789g123456789h123",
			},
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Invalid credentials",
			reqBody: map[string]interface{}{
				"email":    "wrong@email.com",
				"password": "wrongpassword",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.On("Login", mock.Anything, mock.AnythingOfType("dto.LoginRequest")).
					Return(user.User{}, "", "", auth.ErrInvalidCredentials)
			},
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "INVALID_CREDENTIALS",
		},
		{
			name: "Error - User suspended",
			reqBody: map[string]interface{}{
				"email":    "suspended@email.com",
				"password": "validpassword123",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.On("Login", mock.Anything, mock.AnythingOfType("dto.LoginRequest")).
					Return(user.User{}, "", "", auth.ErrUserSuspended)
			},
			expectedStatusCode: http.StatusForbidden,
			expectedErrorCode:  "USER_SUSPENDED",
		},
		{
			name: "Error - Email chưa xác minh",
			reqBody: map[string]interface{}{
				"email":    "unverified@email.com",
				"password": "validpassword123",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.On("Login", mock.Anything, mock.AnythingOfType("dto.LoginRequest")).
					Return(user.User{}, "", "", auth.ErrEmailNotVerified)
			},
			expectedStatusCode: http.StatusForbidden,
			expectedErrorCode:  "EMAIL_NOT_VERIFIED",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockAuthService)
			tc.mockSetup(mockSvc)
			router := setupAuthRouter(mockSvc)

			var reqBodyBytes []byte
			if strBody, ok := tc.reqBody.(string); ok {
				reqBodyBytes = []byte(strBody)
			} else {
				reqBodyBytes, _ = json.Marshal(tc.reqBody)
			}

			req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatusCode, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)

			if tc.expectedErrorCode != "" {
				errResp, ok := resp["error"].(map[string]interface{})
				assert.True(t, ok, "Phản hồi lỗi thiếu block 'error'")
				if code, exists := errResp["code"]; exists {
					assert.Equal(t, tc.expectedErrorCode, code)
				}
			} else {
				assert.Nil(t, resp["error"])
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_VerifyEmail(t *testing.T) {
	type testCase struct {
		name               string
		queryParams        string
		mockSetup          func(m *mocks.MockAuthService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name:        "Success - Xác minh email thành công",
			queryParams: "?token=valid-token-123",
			mockSetup: func(m *mocks.MockAuthService) {
				m.On("VerifyEmail", mock.Anything, "valid-token-123").
					Return(user.User{ID: uuid.New().String(), IsVerified: true}, nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:        "Error - Token không hợp lệ",
			queryParams: "?token=invalid-token",
			mockSetup: func(m *mocks.MockAuthService) {
				m.On("VerifyEmail", mock.Anything, "invalid-token").
					Return(user.User{}, auth.ErrInvalidToken)
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "INVALID_TOKEN",
		},
		{
			name:        "Error - Không có token",
			queryParams: "",
			mockSetup: func(m *mocks.MockAuthService) {
				m.On("VerifyEmail", mock.Anything, "").
					Return(user.User{}, auth.ErrInvalidToken)
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "INVALID_TOKEN",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockAuthService)
			tc.mockSetup(mockSvc)
			router := setupAuthRouter(mockSvc)

			req, _ := http.NewRequest(http.MethodGet, "/api/auth/verify-email"+tc.queryParams, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatusCode, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)

			if tc.expectedErrorCode != "" {
				errResp, ok := resp["error"].(map[string]interface{})
				assert.True(t, ok)
				if code, exists := errResp["code"]; exists {
					assert.Equal(t, tc.expectedErrorCode, code)
				}
			} else {
				assert.Nil(t, resp["error"])
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_ResendVerification(t *testing.T) {
	type testCase struct {
		name               string
		reqBody            interface{}
		mockSetup          func(m *mocks.MockAuthService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name: "Success - Gửi lại email xác minh",
			reqBody: map[string]interface{}{
				"email": "test@email.com",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.On("ResendVerification", mock.Anything, mock.AnythingOfType("string")).
					Return(service.ResendVerificationResult{}, nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Error - Thiếu email",
			reqBody:            map[string]interface{}{},
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Email không hợp lệ",
			reqBody: map[string]interface{}{
				"email": "not-an-email",
			},
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockAuthService)
			tc.mockSetup(mockSvc)
			router := setupAuthRouter(mockSvc)

			reqBodyBytes, _ := json.Marshal(tc.reqBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/auth/resend-verification", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatusCode, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)

			if tc.expectedErrorCode != "" {
				errResp, ok := resp["error"].(map[string]interface{})
				assert.True(t, ok)
				if code, exists := errResp["code"]; exists {
					assert.Equal(t, tc.expectedErrorCode, code)
				}
			} else {
				assert.Nil(t, resp["error"])
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_ForgotPassword(t *testing.T) {
	type testCase struct {
		name               string
		reqBody            interface{}
		mockSetup          func(m *mocks.MockAuthService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name: "Success - Gửi email reset password",
			reqBody: map[string]interface{}{
				"email": "test@email.com",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.On("ForgotPassword", mock.Anything, mock.AnythingOfType("string")).
					Return(nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Error - Thiếu email",
			reqBody:            map[string]interface{}{},
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Email không hợp lệ",
			reqBody: map[string]interface{}{
				"email": "not-an-email",
			},
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockAuthService)
			tc.mockSetup(mockSvc)
			router := setupAuthRouter(mockSvc)

			reqBodyBytes, _ := json.Marshal(tc.reqBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/auth/forgot-password", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatusCode, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)

			if tc.expectedErrorCode != "" {
				errResp, ok := resp["error"].(map[string]interface{})
				assert.True(t, ok)
				if code, exists := errResp["code"]; exists {
					assert.Equal(t, tc.expectedErrorCode, code)
				}
			} else {
				assert.Nil(t, resp["error"])
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_ResetPassword(t *testing.T) {
	type testCase struct {
		name               string
		reqBody            interface{}
		mockSetup          func(m *mocks.MockAuthService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name: "Success - Reset password thành công",
			reqBody: map[string]interface{}{
				"token":        "valid-reset-token",
				"new_password": "newpassword123",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.On("ResetPassword", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "Error - Thiếu token",
			reqBody: map[string]interface{}{
				"new_password": "newpassword123",
			},
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Thiếu new_password",
			reqBody: map[string]interface{}{
				"token": "valid-reset-token",
			},
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Password ngắn hơn 8 ký tự",
			reqBody: map[string]interface{}{
				"token":        "valid-reset-token",
				"new_password": "short",
			},
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Password dài hơn 72 ký tự",
			reqBody: map[string]interface{}{
				"token":        "valid-reset-token",
				"new_password": "a123456789b123456789c123456789d123456789e123456789f123456789g123456789h123",
			},
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Token không hợp lệ (service error)",
			reqBody: map[string]interface{}{
				"token":        "expired-token",
				"new_password": "newpassword123",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.On("ResetPassword", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(auth.ErrInvalidResetToken)
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "INVALID_RESET_TOKEN",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockAuthService)
			tc.mockSetup(mockSvc)
			router := setupAuthRouter(mockSvc)

			reqBodyBytes, _ := json.Marshal(tc.reqBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/auth/reset-password", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatusCode, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)

			if tc.expectedErrorCode != "" {
				errResp, ok := resp["error"].(map[string]interface{})
				assert.True(t, ok)
				if code, exists := errResp["code"]; exists {
					assert.Equal(t, tc.expectedErrorCode, code)
				}
			} else {
				assert.Nil(t, resp["error"])
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_Me(t *testing.T) {
	validUserID := uuid.New()

	type testCase struct {
		name               string
		userIDHeader       string
		mockSetup          func(m *mocks.MockAuthService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name:         "Success - Lấy thông tin user hiện tại",
			userIDHeader: validUserID.String(),
			mockSetup: func(m *mocks.MockAuthService) {
				m.On("GetCurrentUser", mock.Anything, validUserID).
					Return(user.User{
						ID:         validUserID.String(),
						Email:      "me@email.com",
						IsVerified: true,
						Status:     "ACTIVE",
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					}, nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Error - Không có auth header (Unauthorized)",
			userIDHeader:       "",
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "UNAUTHORIZED",
		},
		{
			name:               "Error - Invalid user ID",
			userIDHeader:       "invalid-uuid",
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "INVALID_TOKEN",
		},
		{
			name:         "Error - User not found",
			userIDHeader: validUserID.String(),
			mockSetup: func(m *mocks.MockAuthService) {
				m.On("GetCurrentUser", mock.Anything, validUserID).
					Return(user.User{}, auth.ErrUserNotFound)
			},
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  "USER_NOT_FOUND",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockAuthService)
			tc.mockSetup(mockSvc)
			router := setupAuthRouter(mockSvc)

			req, _ := http.NewRequest(http.MethodGet, "/api/auth/me", nil)
			if tc.userIDHeader != "" {
				req.Header.Set("X-Test-User-ID", tc.userIDHeader)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatusCode, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)

			if tc.expectedErrorCode != "" {
				errResp, ok := resp["error"].(map[string]interface{})
				assert.True(t, ok)
				if code, exists := errResp["code"]; exists {
					assert.Equal(t, tc.expectedErrorCode, code)
				}
			} else {
				assert.Nil(t, resp["error"])
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_GoogleLogin(t *testing.T) {
	type testCase struct {
		name               string
		reqBody            interface{}
		mockSetup          func(m *mocks.MockAuthService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name: "Error - Thiếu email",
			reqBody: map[string]interface{}{
				"id_token": "some-google-token",
			},
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Thiếu id_token",
			reqBody: map[string]interface{}{
				"email": "test@email.com",
			},
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name:               "Error - Body trống",
			reqBody:            map[string]interface{}{},
			mockSetup:          func(m *mocks.MockAuthService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockAuthService)
			tc.mockSetup(mockSvc)

			gin.SetMode(gin.TestMode)
			router := gin.New()
			authHandler := handler.NewAuthHandler(mockSvc, nil, "development", 60, 7)
			router.POST("/api/auth/google", authHandler.GoogleLogin)

			reqBodyBytes, _ := json.Marshal(tc.reqBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/auth/google", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatusCode, w.Code)

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)

			if tc.expectedErrorCode != "" {
				errResp, ok := resp["error"].(map[string]interface{})
				assert.True(t, ok)
				if code, exists := errResp["code"]; exists {
					assert.Equal(t, tc.expectedErrorCode, code)
				}
			}

			mockSvc.AssertExpectations(t)
		})
	}
}
