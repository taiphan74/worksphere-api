package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"worksphere-api/internal/middleware"
)

func TestRequireRoles_UserAccessingAdminEndpoint(t *testing.T) {
	type testCase struct {
		name               string
		userRoles          []string
		requiredRoles      []string
		expectedStatusCode int
		expectedErrorCode  string
	}

	tests := []testCase{
		{
			name:               "USER trying to access ADMIN endpoint - should fail",
			userRoles:          []string{"USER"},
			requiredRoles:      []string{"ADMIN"},
			expectedStatusCode: http.StatusForbidden,
			expectedErrorCode:  "INSUFFICIENT_PERMISSIONS",
		},
		{
			name:               "ADMIN accessing ADMIN endpoint - should succeed",
			userRoles:          []string{"ADMIN"},
			requiredRoles:      []string{"ADMIN"},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "ADMIN with multiple roles - should succeed",
			userRoles:          []string{"USER", "ADMIN", "SUPER_ADMIN"},
			requiredRoles:      []string{"ADMIN", "SUPER_ADMIN"},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "USER with one role trying ADMIN or SUPER_ADMIN - should fail",
			userRoles:          []string{"USER"},
			requiredRoles:      []string{"ADMIN", "SUPER_ADMIN"},
			expectedStatusCode: http.StatusForbidden,
			expectedErrorCode:  "INSUFFICIENT_PERMISSIONS",
		},
		{
			name:               "SUPER_ADMIN accessing ADMIN endpoint - should succeed",
			userRoles:          []string{"SUPER_ADMIN"},
			requiredRoles:      []string{"ADMIN", "SUPER_ADMIN"},
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()

			// Simulate JWT auth with specific roles
			router.Use(func(c *gin.Context) {
				c.Set(middleware.CurrentUserIDKey, uuid.New())
				c.Set("user_roles", tc.userRoles)
				c.Next()
			})

			// Apply role middleware
			api := router.Group("/api")
			api.Use(middleware.RequireRoles(tc.requiredRoles...))
			{
				api.GET("/test", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"message": "success"})
				})
			}

			req, _ := http.NewRequest(http.MethodGet, "/api/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatusCode, w.Code)

			if tc.expectedErrorCode != "" {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)

				errResp, ok := resp["error"].(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, tc.expectedErrorCode, errResp["code"])
			}
		})
	}
}

func TestRequireRole_SingleRoleCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Set user with ADMIN role
	router.Use(func(c *gin.Context) {
		c.Set(middleware.CurrentUserIDKey, uuid.New())
		c.Set("user_roles", []string{"ADMIN"})
		c.Next()
	})

	// Apply single role middleware
	api := router.Group("/api")
	api.Use(middleware.RequireRole("ADMIN"))
	{
		api.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
	}

	req, _ := http.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireRoles_NoRolesInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// No roles set in context
	router.Use(func(c *gin.Context) {
		c.Set(middleware.CurrentUserIDKey, uuid.New())
		// Intentionally not setting user_roles
		c.Next()
	})

	api := router.Group("/api")
	api.Use(middleware.RequireRoles("ADMIN"))
	{
		api.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
	}

	req, _ := http.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "UNAUTHORIZED")
}

func TestRequireRoles_InvalidRolesTypeInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Invalid type for roles
	router.Use(func(c *gin.Context) {
		c.Set(middleware.CurrentUserIDKey, uuid.New())
		c.Set("user_roles", "invalid-type") // Should be []string
		c.Next()
	})

	api := router.Group("/api")
	api.Use(middleware.RequireRoles("ADMIN"))
	{
		api.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})
	}

	req, _ := http.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "INTERNAL_ERROR")
}
