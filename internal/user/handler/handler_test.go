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
	"worksphere-api/internal/user/dto"
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

func TestCreateUser(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockService := new(mocks.MockUserService)
		router := setupRouter(mockService)
		
		reqBody := dto.CreateUserRequest{
			Email:    "test@example.com",
			Password: "password123",
			FullName: stringPtr("Test User"),
		}
		
		expectedUser := user.User{
			ID:         uuid.New().String(),
			Email:      reqBody.Email,
			FullName:   reqBody.FullName,
			IsVerified: false,
			Status:     "ACTIVE",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		
		mockService.On("CreateUser", mock.Anything, reqBody).Return(expectedUser, nil)
		
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		
		// Act
		router.ServeHTTP(w, req)
		
		// Assert
		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
		
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, expectedUser.Email, data["email"])
		assert.Equal(t, *expectedUser.FullName, data["full_name"])
		assert.Equal(t, "success", resp["message"])
		
		mockService.AssertExpectations(t)
	})

	t.Run("Validation Error - Missing Email", func(t *testing.T) {
		// Arrange
		mockService := new(mocks.MockUserService)
		router := setupRouter(mockService)
		
		reqBody := dto.CreateUserRequest{
			Password: "password123",
		}
		
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		
		// Act
		router.ServeHTTP(w, req)
		
		// Assert
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
		
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		
		errResp := resp["error"].(map[string]interface{})
		assert.Equal(t, "VALIDATION_ERROR", errResp["code"])
	})
}

func TestGetUser(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockService := new(mocks.MockUserService)
		router := setupRouter(mockService)
		
		userID := uuid.New()
		expectedUser := user.User{
			ID:        userID.String(),
			Email:     "test@example.com",
			Status:    "ACTIVE",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		mockService.On("GetUserByID", mock.Anything, userID).Return(expectedUser, nil)
		
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String(), nil)
		w := httptest.NewRecorder()
		
		// Act
		router.ServeHTTP(w, req)
		
		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
		
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, userID.String(), data["id"])
		assert.Equal(t, "test@example.com", data["email"])
		
		mockService.AssertExpectations(t)
	})

	t.Run("Not Found", func(t *testing.T) {
		// Arrange
		mockService := new(mocks.MockUserService)
		router := setupRouter(mockService)
		
		userID := uuid.New()
		mockService.On("GetUserByID", mock.Anything, userID).Return(user.User{}, user.ErrUserNotFound)
		
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String(), nil)
		w := httptest.NewRecorder()
		
		// Act
		router.ServeHTTP(w, req)
		
		// Assert
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
		
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		
		errResp := resp["error"].(map[string]interface{})
		assert.Equal(t, "USER_NOT_FOUND", errResp["code"])
	})

	t.Run("Invalid UUID", func(t *testing.T) {
		// Arrange
		mockService := new(mocks.MockUserService)
		router := setupRouter(mockService)
		
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/users/invalid-uuid", nil)
		w := httptest.NewRecorder()
		
		// Act
		router.ServeHTTP(w, req)
		
		// Assert
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
		
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		
		errResp := resp["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_INPUT", errResp["code"])
	})
}

func TestListUsers(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockService := new(mocks.MockUserService)
		router := setupRouter(mockService)
		
		expectedUsers := []user.User{
			{ID: uuid.New().String(), Email: "u1@example.com", Status: "ACTIVE"},
			{ID: uuid.New().String(), Email: "u2@example.com", Status: "INACTIVE"},
		}
		
		mockService.On("ListUsers", mock.Anything, (*string)(nil), (*string)(nil)).Return(expectedUsers, nil)
		
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/users", nil)
		w := httptest.NewRecorder()
		
		// Act
		router.ServeHTTP(w, req)
		
		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
		
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		
		data := resp["data"].([]interface{})
		assert.Equal(t, 2, len(data))
		
		mockService.AssertExpectations(t)
	})
}

func TestUpdateUser(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockService := new(mocks.MockUserService)
		router := setupRouter(mockService)
		
		userID := uuid.New()
		reqBody := dto.UpdateUserRequest{
			FullName: stringPtr("Updated Name"),
		}
		
		expectedUser := user.User{
			ID:       userID.String(),
			Email:    "test@example.com",
			FullName: reqBody.FullName,
			Status:   "ACTIVE",
		}
		
		mockService.On("UpdateUser", mock.Anything, userID, reqBody).Return(expectedUser, nil)
		
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/users/"+userID.String(), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		
		// Act
		router.ServeHTTP(w, req)
		
		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
		
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, *reqBody.FullName, data["full_name"])
		
		mockService.AssertExpectations(t)
	})
}

func TestDeleteUser(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockService := new(mocks.MockUserService)
		router := setupRouter(mockService)
		
		userID := uuid.New()
		mockService.On("DeleteUser", mock.Anything, userID).Return(nil)
		
		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/users/"+userID.String(), nil)
		w := httptest.NewRecorder()
		
		// Act
		router.ServeHTTP(w, req)
		
		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
		
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "success", resp["message"])
		
		mockService.AssertExpectations(t)
	})
}

func TestRestoreUser(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockService := new(mocks.MockUserService)
		router := setupRouter(mockService)
		
		userID := uuid.New()
		expectedUser := user.User{
			ID:     userID.String(),
			Email:  "test@example.com",
			Status: "ACTIVE",
		}
		
		mockService.On("RestoreUser", mock.Anything, userID).Return(expectedUser, nil)
		
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/users/"+userID.String()+"/restore", nil)
		w := httptest.NewRecorder()
		
		// Act
		router.ServeHTTP(w, req)
		
		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
		
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, userID.String(), data["id"])
		
		mockService.AssertExpectations(t)
	})
}

func stringPtr(s string) *string {
	return &s
}
