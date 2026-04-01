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
	"worksphere-api/internal/workspace"
	"worksphere-api/internal/workspace/dto"
	"worksphere-api/internal/workspace/handler"
	"worksphere-api/internal/workspace/mocks"
)

// testUserID dùng chung cho tất cả workspace test
var wsTestUserID = uuid.New()

// authMiddleware giả lập JWT auth cho workspace tests
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func setupWorkspaceRouter(wsSvc *mocks.MockWorkspaceService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	wsHandler := handler.NewWorkspaceHandler(wsSvc)

	workspaces := router.Group("/api/workspaces")
	workspaces.Use(authMiddleware())
	workspaces.POST("", wsHandler.CreateWorkspace)
	workspaces.GET("", wsHandler.ListWorkspaces)
	workspaces.GET("/:id", wsHandler.GetWorkspaceByID)
	workspaces.GET("/slug/:slug", wsHandler.GetWorkspaceBySlug)
	workspaces.PATCH("/:id", wsHandler.UpdateWorkspace)
	workspaces.DELETE("/:id", wsHandler.DeleteWorkspace)

	return router
}

func TestWorkspaceHandler_CreateWorkspace(t *testing.T) {
	type testCase struct {
		name               string
		reqBody            interface{}
		mockSetup          func(m *mocks.MockWorkspaceService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name: "Success - Tạo workspace thành công",
			reqBody: map[string]interface{}{
				"name": "My Workspace",
			},
			mockSetup: func(m *mocks.MockWorkspaceService) {
				m.On("CreateWorkspace", mock.Anything, wsTestUserID, mock.AnythingOfType("dto.CreateWorkspaceRequest")).
					Return(dto.WorkspaceResponse{
						ID:        uuid.New().String(),
						Name:      "My Workspace",
						Slug:      "my-workspace",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil)
			},
			expectedStatusCode: http.StatusCreated,
		},
		{
			name: "Error - Thiếu name",
			reqBody: map[string]interface{}{},
			mockSetup:          func(m *mocks.MockWorkspaceService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "VALIDATION_ERROR",
		},
		{
			name: "Error - Name chỉ có khoảng trắng",
			reqBody: map[string]interface{}{
				"name": "   ",
			},
			mockSetup:          func(m *mocks.MockWorkspaceService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "WORKSPACE_NAME_REQUIRED",
		},
		{
			name:               "Error - Malformed JSON",
			reqBody:            "{ invalid",
			mockSetup:          func(m *mocks.MockWorkspaceService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "INVALID_REQUEST",
		},
		{
			name: "Error - Slug đã tồn tại (service error)",
			reqBody: map[string]interface{}{
				"name": "Existing Workspace",
			},
			mockSetup: func(m *mocks.MockWorkspaceService) {
				m.On("CreateWorkspace", mock.Anything, wsTestUserID, mock.AnythingOfType("dto.CreateWorkspaceRequest")).
					Return(dto.WorkspaceResponse{}, workspace.ErrSlugAlreadyExists)
			},
			expectedStatusCode: http.StatusConflict,
			expectedErrorCode:  "SLUG_ALREADY_EXISTS",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockWorkspaceService)
			tc.mockSetup(mockSvc)
			router := setupWorkspaceRouter(mockSvc)

			var reqBodyBytes []byte
			if strBody, ok := tc.reqBody.(string); ok {
				reqBodyBytes = []byte(strBody)
			} else {
				reqBodyBytes, _ = json.Marshal(tc.reqBody)
			}

			req, _ := http.NewRequest(http.MethodPost, "/api/workspaces", bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Test-User-ID", wsTestUserID.String())
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

func TestWorkspaceHandler_ListWorkspaces(t *testing.T) {
	t.Run("Success - Lấy danh sách workspace", func(t *testing.T) {
		mockSvc := new(mocks.MockWorkspaceService)
		mockSvc.On("ListWorkspacesByUser", mock.Anything, wsTestUserID).
			Return([]dto.WorkspaceResponse{
				{ID: uuid.New().String(), Name: "Workspace 1", Slug: "workspace-1"},
				{ID: uuid.New().String(), Name: "Workspace 2", Slug: "workspace-2"},
			}, nil)
		router := setupWorkspaceRouter(mockSvc)

		req, _ := http.NewRequest(http.MethodGet, "/api/workspaces", nil)
		req.Header.Set("X-Test-User-ID", wsTestUserID.String())
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Nil(t, resp["error"])

		mockSvc.AssertExpectations(t)
	})
}

func TestWorkspaceHandler_GetWorkspaceByID(t *testing.T) {
	validWsID := uuid.New()

	type testCase struct {
		name               string
		pathParamID        string
		mockSetup          func(m *mocks.MockWorkspaceService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name:        "Success - Lấy workspace by ID",
			pathParamID: validWsID.String(),
			mockSetup: func(m *mocks.MockWorkspaceService) {
				m.On("GetWorkspaceByID", mock.Anything, wsTestUserID, validWsID).
					Return(dto.WorkspaceResponse{
						ID:   validWsID.String(),
						Name: "Test Workspace",
						Slug: "test-workspace",
					}, nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Error - Invalid UUID",
			pathParamID:        "invalid-uuid-format",
			mockSetup:          func(m *mocks.MockWorkspaceService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "INVALID_WORKSPACE_ID",
		},
		{
			name:        "Error - Workspace not found",
			pathParamID: validWsID.String(),
			mockSetup: func(m *mocks.MockWorkspaceService) {
				m.On("GetWorkspaceByID", mock.Anything, wsTestUserID, validWsID).
					Return(dto.WorkspaceResponse{}, workspace.ErrWorkspaceNotFound)
			},
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  "WORKSPACE_NOT_FOUND",
		},
		{
			name:        "Error - Not a member (Forbidden)",
			pathParamID: validWsID.String(),
			mockSetup: func(m *mocks.MockWorkspaceService) {
				m.On("GetWorkspaceByID", mock.Anything, wsTestUserID, validWsID).
					Return(dto.WorkspaceResponse{}, workspace.ErrForbiddenAccess)
			},
			expectedStatusCode: http.StatusForbidden,
			expectedErrorCode:  "FORBIDDEN_ACCESS",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockWorkspaceService)
			tc.mockSetup(mockSvc)
			router := setupWorkspaceRouter(mockSvc)

			req, _ := http.NewRequest(http.MethodGet, "/api/workspaces/"+tc.pathParamID, nil)
			req.Header.Set("X-Test-User-ID", wsTestUserID.String())
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatusCode, w.Code)

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

func TestWorkspaceHandler_GetWorkspaceBySlug(t *testing.T) {
	type testCase struct {
		name               string
		slug               string
		mockSetup          func(m *mocks.MockWorkspaceService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name: "Success - Lấy workspace by slug",
			slug: "my-workspace",
			mockSetup: func(m *mocks.MockWorkspaceService) {
				m.On("GetWorkspaceBySlug", mock.Anything, wsTestUserID, "my-workspace").
					Return(dto.WorkspaceResponse{
						ID:   uuid.New().String(),
						Name: "My Workspace",
						Slug: "my-workspace",
					}, nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "Error - Workspace not found",
			slug: "nonexistent",
			mockSetup: func(m *mocks.MockWorkspaceService) {
				m.On("GetWorkspaceBySlug", mock.Anything, wsTestUserID, "nonexistent").
					Return(dto.WorkspaceResponse{}, workspace.ErrWorkspaceNotFound)
			},
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  "WORKSPACE_NOT_FOUND",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockWorkspaceService)
			tc.mockSetup(mockSvc)
			router := setupWorkspaceRouter(mockSvc)

			req, _ := http.NewRequest(http.MethodGet, "/api/workspaces/slug/"+tc.slug, nil)
			req.Header.Set("X-Test-User-ID", wsTestUserID.String())
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatusCode, w.Code)

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

func TestWorkspaceHandler_UpdateWorkspace(t *testing.T) {
	validWsID := uuid.New()

	type testCase struct {
		name               string
		pathParamID        string
		reqBody            interface{}
		mockSetup          func(m *mocks.MockWorkspaceService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name:        "Success - Update workspace name",
			pathParamID: validWsID.String(),
			reqBody: map[string]interface{}{
				"name": "Updated Workspace",
			},
			mockSetup: func(m *mocks.MockWorkspaceService) {
				m.On("UpdateWorkspace", mock.Anything, wsTestUserID, validWsID, mock.AnythingOfType("dto.UpdateWorkspaceRequest")).
					Return(dto.WorkspaceResponse{
						ID:   validWsID.String(),
						Name: "Updated Workspace",
						Slug: "updated-workspace",
					}, nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:        "Error - Invalid workspace UUID",
			pathParamID: "invalid-uuid",
			reqBody: map[string]interface{}{
				"name": "Test",
			},
			mockSetup:          func(m *mocks.MockWorkspaceService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "INVALID_WORKSPACE_ID",
		},
		{
			name:        "Error - Name chỉ có khoảng trắng",
			pathParamID: validWsID.String(),
			reqBody: map[string]interface{}{
				"name": "   ",
			},
			mockSetup:          func(m *mocks.MockWorkspaceService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "WORKSPACE_NAME_REQUIRED",
		},
		{
			name:        "Error - Insufficient permissions",
			pathParamID: validWsID.String(),
			reqBody: map[string]interface{}{
				"name": "New Name",
			},
			mockSetup: func(m *mocks.MockWorkspaceService) {
				m.On("UpdateWorkspace", mock.Anything, wsTestUserID, validWsID, mock.AnythingOfType("dto.UpdateWorkspaceRequest")).
					Return(dto.WorkspaceResponse{}, workspace.ErrInsufficientPermissions)
			},
			expectedStatusCode: http.StatusForbidden,
			expectedErrorCode:  "INSUFFICIENT_PERMISSIONS",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockWorkspaceService)
			tc.mockSetup(mockSvc)
			router := setupWorkspaceRouter(mockSvc)

			var reqBodyBytes []byte
			if strBody, ok := tc.reqBody.(string); ok {
				reqBodyBytes = []byte(strBody)
			} else {
				reqBodyBytes, _ = json.Marshal(tc.reqBody)
			}

			req, _ := http.NewRequest(http.MethodPatch, "/api/workspaces/"+tc.pathParamID, bytes.NewBuffer(reqBodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Test-User-ID", wsTestUserID.String())
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatusCode, w.Code)

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

func TestWorkspaceHandler_DeleteWorkspace(t *testing.T) {
	validWsID := uuid.New()

	type testCase struct {
		name               string
		pathParamID        string
		mockSetup          func(m *mocks.MockWorkspaceService)
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name:        "Success - Xóa workspace thành công",
			pathParamID: validWsID.String(),
			mockSetup: func(m *mocks.MockWorkspaceService) {
				m.On("DeleteWorkspace", mock.Anything, wsTestUserID, validWsID).
					Return(nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Error - Invalid UUID",
			pathParamID:        "invalid-uuid",
			mockSetup:          func(m *mocks.MockWorkspaceService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  "INVALID_WORKSPACE_ID",
		},
		{
			name:        "Error - Forbidden",
			pathParamID: validWsID.String(),
			mockSetup: func(m *mocks.MockWorkspaceService) {
				m.On("DeleteWorkspace", mock.Anything, wsTestUserID, validWsID).
					Return(workspace.ErrForbiddenAccess)
			},
			expectedStatusCode: http.StatusForbidden,
			expectedErrorCode:  "FORBIDDEN_ACCESS",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockSvc := new(mocks.MockWorkspaceService)
			tc.mockSetup(mockSvc)
			router := setupWorkspaceRouter(mockSvc)

			req, _ := http.NewRequest(http.MethodDelete, "/api/workspaces/"+tc.pathParamID, nil)
			req.Header.Set("X-Test-User-ID", wsTestUserID.String())
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatusCode, w.Code)

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
