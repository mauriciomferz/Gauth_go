package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Gimel-Foundation/gauth/gauth-demo-app/web/backend/services"
)

func setupTokenTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	// Create mock config for the service
	config := viper.New()
	config.SetDefault("redis.addr", "localhost:6379")
	config.SetDefault("redis.password", "")
	config.SetDefault("redis.db", 0)
	
	// Create GAuth service (will handle Redis connection failures gracefully)
	mockService, err := services.NewGAuthService(config, logger)
	if err != nil {
		logger.WithError(err).Warn("Failed to create GAuth service for tests")
	}
	
	tokenHandler := NewTokenHandler(mockService, logger)
	
	v1 := router.Group("/api/v1")
	{
		v1.POST("/tokens", tokenHandler.CreateToken)
		v1.POST("/tokens/validate", tokenHandler.ValidateToken)
		v1.POST("/tokens/refresh", tokenHandler.RefreshToken)
		v1.DELETE("/tokens/:id", tokenHandler.RevokeToken)
		v1.GET("/tokens", tokenHandler.ListTokens)
	}
	
	return router
}

func TestTokenHandler_CreateToken(t *testing.T) {
	router := setupTokenTestRouter()

	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
	}{
		{
			name: "Create JWT Token",
			payload: map[string]interface{}{
				"type":     "JWT",
				"subject":  "user123",
				"scopes":   []string{"read", "write"},
				"expires_in": 3600,
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Create PASETO Token",
			payload: map[string]interface{}{
				"type":       "PASETO",
				"subject":    "user456",
				"scopes":     []string{"transaction:execute"},
				"expires_in": 7200,
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Create Token with Custom Claims",
			payload: map[string]interface{}{
				"type":    "JWT",
				"subject": "admin",
				"scopes":  []string{"admin", "user:manage"},
				"claims": map[string]interface{}{
					"role":        "administrator",
					"department":  "IT",
					"permissions": []string{"create", "read", "update", "delete"},
				},
				"expires_in": 1800,
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Missing Subject",
			payload: map[string]interface{}{
				"type":   "JWT",
				"scopes": []string{"read"},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid Token Type",
			payload: map[string]interface{}{
				"type":    "INVALID",
				"subject": "user123",
				"scopes":  []string{"read"},
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonPayload, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/tokens", bytes.NewBuffer(jsonPayload))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				// The handler returns the token object directly, not wrapped in success structure
				assert.Contains(t, response, "token")
				assert.Contains(t, response, "expires_at")
				assert.NotEmpty(t, response["token"])
				
				// Safely check claims if they exist
				if claims, ok := response["claims"]; ok && claims != nil {
					// Claims exist and are not nil
					assert.NotNil(t, claims)
				}
			}
		})
	}
}

func TestTokenHandler_ValidateToken(t *testing.T) {
	router := setupTokenTestRouter()

	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
	}{
		{
			name: "Valid JWT Token",
			payload: map[string]interface{}{
				"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.signature",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Valid PASETO Token", 
			payload: map[string]interface{}{
				"token": "v2.local.test-paseto-token",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Invalid Token Format",
			payload: map[string]interface{}{
				"token": "invalid-token-format",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Expired Token",
			payload: map[string]interface{}{
				"token": "expired.token.here",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Missing Token",
			payload: map[string]interface{}{
				"type": "JWT",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonPayload, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/tokens/validate", bytes.NewBuffer(jsonPayload))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				assert.Contains(t, response, "valid")
				assert.Contains(t, response, "token_info")
				
				if tokenInfo, ok := response["token_info"].(map[string]interface{}); ok {
					assert.Contains(t, tokenInfo, "subject")
					assert.Contains(t, tokenInfo, "scopes")
					assert.Contains(t, tokenInfo, "issued_at")
					assert.Contains(t, tokenInfo, "expires_at")
				}
			}
		})
	}
}

func TestTokenHandler_RefreshToken(t *testing.T) {
	router := setupTokenTestRouter()

	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
	}{
		{
			name: "Valid Refresh Token",
			payload: map[string]interface{}{
				"refresh_token": "valid_refresh_token_123",
				"client_id":     "demo_client",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Invalid Refresh Token",
			payload: map[string]interface{}{
				"refresh_token": "invalid_token",
				"client_id":     "demo_client",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Missing Refresh Token",
			payload: map[string]interface{}{
				"client_id": "demo_client",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonPayload, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/api/v1/tokens/refresh", bytes.NewBuffer(jsonPayload))
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
				assert.Contains(t, response, "refresh_token")
				assert.Contains(t, response, "expires_in")
			}
		})
	}
}

func TestTokenHandler_RevokeToken(t *testing.T) {
	router := setupTokenTestRouter()

	tests := []struct {
		name           string
		tokenID        string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "Valid Token Revocation",
			tokenID:        "token_123",
			authHeader:     "Bearer valid_admin_token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Unauthorized Revocation",
			tokenID:        "token_456",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Token Not Found",
			tokenID:        "nonexistent",
			authHeader:     "Bearer valid_admin_token",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("DELETE", "/api/v1/tokens/"+tt.tokenID, nil)
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
				
				assert.Contains(t, response, "success")
				assert.True(t, response["success"].(bool))
				assert.Contains(t, response, "message")
			}
		})
	}
}

func TestTokenHandler_ListTokens(t *testing.T) {
	router := setupTokenTestRouter()

	tests := []struct {
		name           string
		queryParams    string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "List All Tokens",
			queryParams:    "",
			authHeader:     "Bearer valid_admin_token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "List Tokens with Pagination",
			queryParams:    "?page=1&limit=10",
			authHeader:     "Bearer valid_admin_token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Filter by Subject",
			queryParams:    "?subject=user123",
			authHeader:     "Bearer valid_admin_token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Filter by Token Type",
			queryParams:    "?type=JWT",
			authHeader:     "Bearer valid_admin_token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Unauthorized Access",
			queryParams:    "",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/api/v1/tokens"+tt.queryParams, nil)
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
				
				assert.Contains(t, response, "tokens")
				assert.Contains(t, response, "total")
				assert.Contains(t, response, "page")
				assert.Contains(t, response, "limit")
				
				if tokens, ok := response["tokens"].([]interface{}); ok {
					// Verify token structure
					for _, token := range tokens {
						if tokenData, ok := token.(map[string]interface{}); ok {
							assert.Contains(t, tokenData, "id")
							assert.Contains(t, tokenData, "subject")
							assert.Contains(t, tokenData, "type")
							assert.Contains(t, tokenData, "created_at")
							assert.Contains(t, tokenData, "expires_at")
						}
					}
				}
			}
		})
	}
}
