package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Gimel-Foundation/gauth/gauth-demo-app/web/backend/services"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Mock logger
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Suppress logs during tests
	
	// Create mock config for the service
	config := viper.New()
	config.SetDefault("redis.addr", "localhost:6379")
	config.SetDefault("redis.password", "")
	config.SetDefault("redis.db", 0)
	
	// Create GAuth service (will handle Redis connection failures gracefully)
	mockService, err := services.NewGAuthService(config, logger)
	if err != nil {
		// In case of service creation failure, create a minimal handler anyway
		logger.WithError(err).Warn("Failed to create GAuth service for tests")
	}
	
	authHandler := &AuthHandler{
		service: mockService,
		logger:  logger,
	}
	
	// Register routes using the actual available methods
	v1 := router.Group("/api/v1")
	{
		v1.POST("/auth/authorize", authHandler.Authorize)
		v1.POST("/auth/token", authHandler.Token)
		v1.POST("/auth/revoke", authHandler.Revoke)
		v1.GET("/auth/validate", authHandler.Validate)
		v1.GET("/auth/userinfo", authHandler.UserInfo)
	}
	
	return router
}

func TestAuthHandler_Authorize(t *testing.T) {
	router := setupTestRouter()

	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid Authorization",
			payload: map[string]interface{}{
				"client_id":     "testclient",
				"response_type": "code",
				"scope":         "read write",
				"redirect_uri":  "https://example.com/callback",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Missing Client ID",
			payload: map[string]interface{}{
				"response_type": "code",
				"scope":         "read",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing Response Type",
			payload: map[string]interface{}{
				"client_id": "testclient",
				"scope":     "read",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty Payload",
			payload:        map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonPayload, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/auth/authorize", bytes.NewBuffer(jsonPayload))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				// The authorize endpoint returns an authorization code, not access tokens
				assert.Contains(t, response, "code")
				assert.Contains(t, response, "redirect_uri")
				assert.NotEmpty(t, response["code"])
				assert.Equal(t, tt.payload["redirect_uri"], response["redirect_uri"])
			} else if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				assert.Contains(t, response, "error")
				assert.Contains(t, response["error"].(string), tt.expectedError)
			}
		})
	}
}



func TestAuthHandler_Token(t *testing.T) {
	router := setupTestRouter()

	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
	}{
		{
			name: "Valid Token Request",
			payload: map[string]interface{}{
				"grant_type":    "authorization_code",
				"code":         "test_auth_code_123",
				"client_id":    "demo_client",
				"client_secret": "demo_secret",
				"redirect_uri": "http://localhost:3000/callback",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Refresh Token Request",
			payload: map[string]interface{}{
				"grant_type":    "refresh_token",
				"refresh_token": "test_refresh_token_123",
				"client_id":     "demo_client",
				"client_secret": "demo_secret",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Invalid Grant Type",
			payload: map[string]interface{}{
				"grant_type": "invalid_grant",
				"code":      "test_code",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonPayload, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/auth/token", bytes.NewBuffer(jsonPayload))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				assert.Contains(t, response, "access_token")
				assert.Contains(t, response, "token_type")
				assert.Contains(t, response, "expires_in")
			}
		})
	}
}

func TestAuthHandler_ValidateToken(t *testing.T) {
	router := setupTestRouter()

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "Valid Token",
			authHeader:     "Bearer valid_test_token_123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid Token Format",
			authHeader:     "InvalidFormat token123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing Authorization Header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Expired Token",
			authHeader:     "Bearer expired_token_123",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/auth/validate", nil)
			require.NoError(t, err)
			
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				assert.Contains(t, response, "valid")
				assert.True(t, response["valid"].(bool))
				assert.Contains(t, response, "user_id")
				assert.Contains(t, response, "client_id")
				assert.Contains(t, response, "scope")
			}
		})
	}
}

func TestAuthHandler_Revoke(t *testing.T) {
	router := setupTestRouter()

	tests := []struct {
		name           string
		formData       map[string]string
		expectedStatus int
	}{
		{
			name: "Valid Token Revocation",
			formData: map[string]string{
				"token": "test_token_123",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Revoke Refresh Token",
			formData: map[string]string{
				"token": "refresh_token_456",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing Token",
			formData:       map[string]string{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create form data
			formData := url.Values{}
			for key, value := range tt.formData {
				formData.Set(key, value)
			}

			req, err := http.NewRequest("POST", "/api/v1/auth/revoke", strings.NewReader(formData.Encode()))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				assert.Contains(t, response, "message")
				assert.Equal(t, "Token revoked successfully", response["message"])
			}
		})
	}
}
