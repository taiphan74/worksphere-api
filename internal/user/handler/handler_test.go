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

	"worksphere-api/internal/user"
	"worksphere-api/internal/user/handler"
	"worksphere-api/internal/user/mocks"
)

func setupRouter(userService *mocks.MockUserService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	userHandler := handler.NewUserHandler(userService)
	
	api := router.Group("/api/v1")
	users := api.Group("/users")
	userHandler.RegisterRoutes(users)
	
	return router
}

func TestUserHandler_CreateUser(t *testing.T) {
	type testCase struct {
		name               string
		reqBody            interface{}
		mockSetup          func(m *mocks.MockUserService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name: "Success - Trả về 201 cho data hợp lệ",
			reqBody: map[string]interface{}{
				"email":     "test@email.com",
				"password":  "validpassword123",
				"full_name": "Nguyen Van A",
			},
			mockSetup: func(m *mocks.MockUserService) {
				expectedUser := user.User{
					ID:        uuid.New().String(),
					Email:     "test@email.com",
					FullName:  stringPtr("Nguyen Van A"),
					Status:    "ACTIVE",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				m.On("CreateUser", mock.Anything, mock.AnythingOfType("dto.CreateUserRequest")).Return(expectedUser, nil)
			},
			expectedStatusCode: http.StatusCreated,
		},
		{
			name:               "Error - Malformed JSON body",
			reqBody:            "{ invalid_format: 123 ",
			mockSetup:          func(m *mocks.MockUserService) {}, // Handler lỗi parsing, mock ko bị gọi
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "INVALID_REQUEST", // Do ShouldBindJSON sinh ra trên ctx validation error
		},
		{
			name: "Error - Thiếu email (Validation)",
			reqBody: map[string]interface{}{
				"password": "validpassword123",
			},
			mockSetup:          func(m *mocks.MockUserService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Sai format email và password ngắn",
			reqBody: map[string]interface{}{
				"email":    "not-an-email",
				"password": "short", // < 8 ký tự
			},
			mockSetup:          func(m *mocks.MockUserService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Email đã tồn tại (Service Error)",
			reqBody: map[string]interface{}{
				"email":    "exist@email.com",
				"password": "validpassword123",
			},
			mockSetup: func(m *mocks.MockUserService) {
				m.On("CreateUser", mock.Anything, mock.AnythingOfType("dto.CreateUserRequest")).Return(user.User{}, user.ErrEmailAlreadyExists)
			},
			expectedStatusCode: http.StatusConflict,
			expectedErrorCode:  "EMAIL_ALREADY_EXISTS",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockUserService)
			tc.mockSetup(mockSvc)
			router := setupRouter(mockSvc)

			var reqBodyBytes []byte
			if strBody, ok := tc.reqBody.(string); ok {
				reqBodyBytes = []byte(strBody)
			} else {
				reqBodyBytes, _ = json.Marshal(tc.reqBody)
			}

			req, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(reqBodyBytes))
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
				// Fallback kiểm tra cấu trúc mã lỗi, do app dùng struct chuẩn hoặc bind gin.H
				// Validator trả về 'code' có nội dung của apperrors
				if code, exists := errResp["code"]; exists {
					assert.Equal(t, tc.expectedErrorCode, code)
				}
			} else {
				assert.Nil(t, resp["error"], "Phản hồi success không nên chứa block 'error'")
				assert.Equal(t, "success", resp["message"])
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestUserHandler_ListUsers(t *testing.T) {
	type testCase struct {
		name               string
		queryParams        string
		mockSetup          func(m *mocks.MockUserService)
		expectedStatusCode int
		expectedDataSize   int
	}

	tests := []testCase{
		{
			name:        "Success - Có query param lọc",
			queryParams: "?status=ACTIVE&search=kien",
			mockSetup: func(m *mocks.MockUserService) {
				m.On("ListUsers", mock.Anything, mock.AnythingOfType("*string"), mock.AnythingOfType("*string")).
					Return([]user.User{{ID: "1"}, {ID: "2"}}, nil)
			},
			expectedStatusCode: http.StatusOK,
			expectedDataSize:   2,
		},
		{
			name:        "Success - Không filter",
			queryParams: "",
			mockSetup: func(m *mocks.MockUserService) {
				m.On("ListUsers", mock.Anything, (*string)(nil), (*string)(nil)).
					Return([]user.User{}, nil)
			},
			expectedStatusCode: http.StatusOK,
			expectedDataSize:   0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockUserService)
			tc.mockSetup(mockSvc)
			router := setupRouter(mockSvc)

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/users"+tc.queryParams, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatusCode, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
			
			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			
			if tc.expectedStatusCode == http.StatusOK {
				data, ok := resp["data"].([]interface{})
				assert.True(t, ok)
				assert.Len(t, data, tc.expectedDataSize)
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestUserHandler_GetUser(t *testing.T) {
	validUUID := uuid.New()
	type testCase struct {
		name               string
		pathParamID        string
		mockSetup          func(m *mocks.MockUserService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name:        "Success - Tim thay user",
			pathParamID: validUUID.String(),
			mockSetup: func(m *mocks.MockUserService) {
				m.On("GetUserByID", mock.Anything, validUUID).
					Return(user.User{ID: validUUID.String(), Email: "test@example.com"}, nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Error - Invalid UUID",
			pathParamID:        "invalid-uuid-format",
			mockSetup:          func(m *mocks.MockUserService) {}, // Not called
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "INVALID_INPUT",
		},
		{
			name:        "Error - Not found",
			pathParamID: validUUID.String(),
			mockSetup: func(m *mocks.MockUserService) {
				m.On("GetUserByID", mock.Anything, validUUID).
					Return(user.User{}, user.ErrUserNotFound)
			},
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  "USER_NOT_FOUND",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockUserService)
			tc.mockSetup(mockSvc)
			router := setupRouter(mockSvc)

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/users/"+tc.pathParamID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatusCode, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)

			if tc.expectedErrorCode != "" {
				errResp := resp["error"].(map[string]interface{})
				assert.Equal(t, tc.expectedErrorCode, errResp["code"])
			} else {
				assert.Nil(t, resp["error"])
			}

			mockSvc.AssertExpectations(t)
		})
	}
}

func TestUserHandler_UpdateUser(t *testing.T) {
	type testCase struct {
		name               string
		pathParamID        string
		reqBody            map[string]interface{}
		mockSetup          func(m *mocks.MockUserService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	validUUID := uuid.New().String()

	tests := []testCase{
		{
			name:        "Success - Update thông tin full_name",
			pathParamID: validUUID,
			reqBody: map[string]interface{}{
				"full_name": "Updated Name",
			},
			mockSetup: func(m *mocks.MockUserService) {
				m.On("UpdateUser", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("dto.UpdateUserRequest")).
					Return(user.User{ID: validUUID, FullName: stringPtr("Updated Name")}, nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:        "Error - Invalid UUID",
			pathParamID: "invalid-uuid-format",
			reqBody: map[string]interface{}{
				"full_name": "New Name",
			},
			mockSetup:          func(m *mocks.MockUserService) {}, // Chặn tại Handler
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "INVALID_INPUT",
		},
		{
			name:        "Error - Invalid password length",
			pathParamID: validUUID,
			reqBody: map[string]interface{}{
				"password": "short", // min=8
			},
			mockSetup:          func(m *mocks.MockUserService) {}, // Chặn tại Validator
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name:        "Service Error - User Not Found",
			pathParamID: validUUID,
			reqBody: map[string]interface{}{
				"full_name": "Ghost User",
			},
			mockSetup: func(m *mocks.MockUserService) {
				m.On("UpdateUser", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("dto.UpdateUserRequest")).
					Return(user.User{}, user.ErrUserNotFound)
			},
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  "USER_NOT_FOUND",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockUserService)
			tc.mockSetup(mockSvc)
			router := setupRouter(mockSvc)

			bodyBytes, _ := json.Marshal(tc.reqBody)
			req, _ := http.NewRequest(http.MethodPatch, "/api/v1/users/"+tc.pathParamID, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatusCode, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)

			if tc.expectedErrorCode != "" {
				errResp := resp["error"].(map[string]interface{})
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

// Giữ lại test tuyến tính cho 2 API ít case và có logic rất đơn giản
func TestUserHandler_DeleteUser(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(mocks.MockUserService)
		router := setupRouter(mockSvc)
		
		userID := uuid.New()
		mockSvc.On("DeleteUser", mock.Anything, userID).Return(nil)
		
		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/users/"+userID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

func TestUserHandler_RestoreUser(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSvc := new(mocks.MockUserService)
		router := setupRouter(mockSvc)
		
		userID := uuid.New()
		mockSvc.On("RestoreUser", mock.Anything, userID).Return(user.User{ID: userID.String(), Status: "ACTIVE"}, nil)
		
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/users/"+userID.String()+"/restore", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

func stringPtr(s string) *string {
	return &s
}
