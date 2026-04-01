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

	"worksphere-api/internal/middleware"
	"worksphere-api/internal/profile/dto"
	"worksphere-api/internal/profile/handler"
	"worksphere-api/internal/profile/mocks"
	apperrors "worksphere-api/pkg/errors"
)

// testUserID dùng chung cho tất cả test cases cần authenticated user
var testUserID = uuid.New()

func setupProfileRouter(profileService *mocks.MockProfileService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	profileHandler := handler.NewProfileHandler(profileService)

	// Simulate JWT auth middleware
	profile := router.Group("/api/profile")
	profile.Use(func(c *gin.Context) {
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
	})

	profile.GET("", profileHandler.GetProfile)
	profile.PATCH("", profileHandler.UpdateProfile)
	profile.POST("/change-password", profileHandler.ChangePassword)

	avatar := profile.Group("/avatar")
	avatar.POST("/upload-url", profileHandler.GetAvatarUploadURL)
	avatar.POST("/confirm", profileHandler.ConfirmAvatarUpload)
	avatar.GET("/view-url", profileHandler.GetAvatarViewURL)

	return router
}

func TestProfileHandler_GetProfile(t *testing.T) {
	type testCase struct {
		name               string
		userIDHeader       string
		mockSetup          func(m *mocks.MockProfileService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name:         "Success - Lấy profile thành công",
			userIDHeader: testUserID.String(),
			mockSetup: func(m *mocks.MockProfileService) {
				m.On("GetProfile", mock.Anything, testUserID).
					Return(dto.ProfileResponse{
						ID:         testUserID.String(),
						Email:      "test@email.com",
						IsVerified: true,
						Status:     "ACTIVE",
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					}, nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Error - Unauthorized (thiếu header)",
			userIDHeader:       "",
			mockSetup:          func(m *mocks.MockProfileService) {},
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  "UNAUTHORIZED",
		},
		{
			name:         "Error - Profile not found",
			userIDHeader: testUserID.String(),
			mockSetup: func(m *mocks.MockProfileService) {
				m.On("GetProfile", mock.Anything, testUserID).
					Return(dto.ProfileResponse{}, apperrors.New(http.StatusNotFound, "RESOURCE_NOT_FOUND", "not found"))
			},
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  "RESOURCE_NOT_FOUND",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockProfileService)
			tc.mockSetup(mockSvc)
			router := setupProfileRouter(mockSvc)

			req, _ := http.NewRequest(http.MethodGet, "/api/profile", nil)
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

func TestProfileHandler_UpdateProfile(t *testing.T) {
	type testCase struct {
		name               string
		reqBody            interface{}
		mockSetup          func(m *mocks.MockProfileService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name: "Success - Update full_name",
			reqBody: map[string]interface{}{
				"full_name": "Nguyen Van A",
			},
			mockSetup: func(m *mocks.MockProfileService) {
				m.On("UpdateProfile", mock.Anything, testUserID, mock.AnythingOfType("dto.UpdateProfileRequest")).
					Return(dto.ProfileResponse{
						ID:    testUserID.String(),
						Email: "test@email.com",
					}, nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "Success - Update phone và job_title",
			reqBody: map[string]interface{}{
				"phone":     "+84912345678",
				"job_title": "Backend Developer",
			},
			mockSetup: func(m *mocks.MockProfileService) {
				m.On("UpdateProfile", mock.Anything, testUserID, mock.AnythingOfType("dto.UpdateProfileRequest")).
					Return(dto.ProfileResponse{
						ID:    testUserID.String(),
						Email: "test@email.com",
					}, nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "Error - full_name quá dài (>150 ký tự)",
			reqBody: map[string]interface{}{
				"full_name": string(make([]byte, 151)),
			},
			mockSetup:          func(m *mocks.MockProfileService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name:               "Error - Malformed JSON",
			reqBody:            "{ invalid",
			mockSetup:          func(m *mocks.MockProfileService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "INVALID_REQUEST",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockProfileService)
			tc.mockSetup(mockSvc)
			router := setupProfileRouter(mockSvc)

			var reqBodyBytes []byte
			if strBody, ok := tc.reqBody.(string); ok {
				reqBodyBytes = []byte(strBody)
			} else {
				reqBodyBytes, _ = json.Marshal(tc.reqBody)
			}

			req, _ := http.NewRequest(http.MethodPatch, "/api/profile", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Test-User-ID", testUserID.String())
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

func TestProfileHandler_ChangePassword(t *testing.T) {
	type testCase struct {
		name               string
		reqBody            interface{}
		mockSetup          func(m *mocks.MockProfileService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name: "Success - Đổi mật khẩu thành công",
			reqBody: map[string]interface{}{
				"current_password":     "oldpassword123",
				"new_password":         "newpassword123",
				"confirm_new_password": "newpassword123",
			},
			mockSetup: func(m *mocks.MockProfileService) {
				m.On("ChangePassword", mock.Anything, testUserID, mock.AnythingOfType("dto.ChangePasswordRequest")).
					Return(nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "Error - Thiếu current_password",
			reqBody: map[string]interface{}{
				"new_password":         "newpassword123",
				"confirm_new_password": "newpassword123",
			},
			mockSetup:          func(m *mocks.MockProfileService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Thiếu new_password",
			reqBody: map[string]interface{}{
				"current_password":     "oldpassword123",
				"confirm_new_password": "newpassword123",
			},
			mockSetup:          func(m *mocks.MockProfileService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Password mới ngắn hơn 8 ký tự",
			reqBody: map[string]interface{}{
				"current_password":     "oldpassword123",
				"new_password":         "short",
				"confirm_new_password": "short",
			},
			mockSetup:          func(m *mocks.MockProfileService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Confirm password không khớp",
			reqBody: map[string]interface{}{
				"current_password":     "oldpassword123",
				"new_password":         "newpassword123",
				"confirm_new_password": "differentpassword",
			},
			mockSetup:          func(m *mocks.MockProfileService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Current password sai (service error)",
			reqBody: map[string]interface{}{
				"current_password":     "wrongpassword",
				"new_password":         "newpassword123",
				"confirm_new_password": "newpassword123",
			},
			mockSetup: func(m *mocks.MockProfileService) {
				m.On("ChangePassword", mock.Anything, testUserID, mock.AnythingOfType("dto.ChangePasswordRequest")).
					Return(apperrors.New(http.StatusBadRequest, "INVALID_PASSWORD", "current password is incorrect"))
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "INVALID_PASSWORD",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockProfileService)
			tc.mockSetup(mockSvc)
			router := setupProfileRouter(mockSvc)

			reqBodyBytes, _ := json.Marshal(tc.reqBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/profile/change-password", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Test-User-ID", testUserID.String())
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

func TestProfileHandler_GetAvatarUploadURL(t *testing.T) {
	type testCase struct {
		name               string
		reqBody            interface{}
		mockSetup          func(m *mocks.MockProfileService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name: "Success - Tạo upload URL thành công",
			reqBody: map[string]interface{}{
				"file_name":    "avatar.jpg",
				"content_type": "image/jpeg",
				"size":         1024,
			},
			mockSetup: func(m *mocks.MockProfileService) {
				m.On("GetAvatarUploadURL", mock.Anything, testUserID, mock.AnythingOfType("dto.AvatarUploadURLRequest")).
					Return(dto.AvatarUploadURLResponse{
						ObjectKey: "profiles/" + testUserID.String() + "/avatar/test.jpg",
						UploadURL: "https://r2.example.com/upload",
						Method:    "PUT",
						ExpiresIn: 900,
					}, nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "Error - Thiếu file_name",
			reqBody: map[string]interface{}{
				"content_type": "image/jpeg",
				"size":         1024,
			},
			mockSetup:          func(m *mocks.MockProfileService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Thiếu content_type",
			reqBody: map[string]interface{}{
				"file_name": "avatar.jpg",
				"size":      1024,
			},
			mockSetup:          func(m *mocks.MockProfileService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Size bằng 0",
			reqBody: map[string]interface{}{
				"file_name":    "avatar.jpg",
				"content_type": "image/jpeg",
				"size":         0,
			},
			mockSetup:          func(m *mocks.MockProfileService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockProfileService)
			tc.mockSetup(mockSvc)
			router := setupProfileRouter(mockSvc)

			reqBodyBytes, _ := json.Marshal(tc.reqBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/profile/avatar/upload-url", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Test-User-ID", testUserID.String())
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

func TestProfileHandler_ConfirmAvatarUpload(t *testing.T) {
	type testCase struct {
		name               string
		reqBody            interface{}
		mockSetup          func(m *mocks.MockProfileService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name: "Success - Confirm upload thành công",
			reqBody: map[string]interface{}{
				"object_key": "profiles/" + testUserID.String() + "/avatar/test.jpg",
			},
			mockSetup: func(m *mocks.MockProfileService) {
				m.On("ConfirmAvatarUpload", mock.Anything, testUserID, mock.AnythingOfType("dto.AvatarConfirmRequest")).
					Return(dto.ProfileResponse{
						ID:    testUserID.String(),
						Email: "test@email.com",
					}, nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "Error - Thiếu object_key",
			reqBody: map[string]interface{}{},
			mockSetup:          func(m *mocks.MockProfileService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockProfileService)
			tc.mockSetup(mockSvc)
			router := setupProfileRouter(mockSvc)

			reqBodyBytes, _ := json.Marshal(tc.reqBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/profile/avatar/confirm", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Test-User-ID", testUserID.String())
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
			} else {
				assert.Nil(t, resp["error"])
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestProfileHandler_GetAvatarViewURL(t *testing.T) {
	type testCase struct {
		name               string
		mockSetup          func(m *mocks.MockProfileService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name: "Success - Lấy avatar view URL thành công",
			mockSetup: func(m *mocks.MockProfileService) {
				m.On("GetAvatarViewURL", mock.Anything, testUserID).
					Return(dto.AvatarViewURLResponse{
						ViewURL:   "https://r2.example.com/view/avatar.jpg",
						ExpiresIn: 600,
					}, nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "Error - Avatar not found",
			mockSetup: func(m *mocks.MockProfileService) {
				m.On("GetAvatarViewURL", mock.Anything, testUserID).
					Return(dto.AvatarViewURLResponse{}, apperrors.New(http.StatusNotFound, "AVATAR_NOT_FOUND", "user has no avatar"))
			},
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  "AVATAR_NOT_FOUND",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockProfileService)
			tc.mockSetup(mockSvc)
			router := setupProfileRouter(mockSvc)

			req, _ := http.NewRequest(http.MethodGet, "/api/profile/avatar/view-url", nil)
			req.Header.Set("X-Test-User-ID", testUserID.String())
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
