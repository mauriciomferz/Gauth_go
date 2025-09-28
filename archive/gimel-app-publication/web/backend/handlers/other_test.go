package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func setupOtherHandlersTestRouter(t *testing.T) (*gin.Engine, *AuditHandler, *RateHandler, *DemoHandler) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test logger
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce test output

	// Create mock GAuth service
	gauthService := createMockGAuthServiceForOther(t)

	// Create handlers
	auditHandler := NewAuditHandler(gauthService, logger)
	rateHandler := NewRateHandler(gauthService, logger)
	demoHandler := NewDemoHandler(gauthService, logger)

	// Setup router
	router := gin.New()

	// Audit endpoints
	audit := router.Group("/api/v1/audit")
	{
		audit.GET("/events", auditHandler.GetEvents)
		audit.GET("/events/:id", auditHandler.GetEvent)
		audit.GET("/compliance", auditHandler.GetComplianceReport)
		audit.GET("/trails/:entity", auditHandler.GetAuditTrail)
	}

	// Rate limiting endpoints
	rate := router.Group("/api/v1/rate")
	{
		rate.GET("/limits", rateHandler.GetLimits)
		rate.POST("/limits", rateHandler.SetLimits)
		rate.GET("/status/:client", rateHandler.GetStatus)
	}

	// Demo scenarios endpoints
	demo := router.Group("/api/v1/demo")
	{
		demo.GET("/scenarios", demoHandler.GetScenarios)
		demo.POST("/scenarios/:id/run", demoHandler.RunScenario)
		demo.GET("/scenarios/:id/status", demoHandler.GetScenarioStatus)
	}

	return router, auditHandler, rateHandler, demoHandler
}

func createMockGAuthServiceForOther(t *testing.T) *services.GAuthService {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	config := viper.New()
	config.SetDefault("redis.addr", "localhost:6379")
	config.SetDefault("redis.password", "")
	config.SetDefault("redis.db", 0)
	
	mockService, err := services.NewGAuthService(config, logger)
	if err != nil {
		logger.WithError(err).Warn("Failed to create GAuth service for tests")
	}
	
	return mockService
}

// Test AuditHandler

func TestAuditHandler_GetEvents(t *testing.T) {
	router, _, _, _ := setupOtherHandlersTestRouter(t)

	tests := []struct {
		name           string
		query          string
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name:           "Get Events with Default Pagination",
			query:          "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response, "events")
				assert.Contains(t, response, "limit")
				assert.Contains(t, response, "offset")
				assert.Equal(t, float64(10), response["limit"])
				assert.Equal(t, float64(0), response["offset"])
			},
		},
		{
			name:           "Get Events with Custom Pagination",
			query:          "?limit=20&offset=5",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, float64(20), response["limit"])
				assert.Equal(t, float64(5), response["offset"])
			},
		},
		{
			name:           "Get Events with Invalid Pagination Parameters",
			query:          "?limit=invalid&offset=invalid",
			expectedStatus: http.StatusOK, // Should use defaults
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, float64(10), response["limit"])
				assert.Equal(t, float64(0), response["offset"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events"+tt.query, nil)
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestAuditHandler_GetEvent(t *testing.T) {
	router, _, _, _ := setupOtherHandlersTestRouter(t)

	tests := []struct {
		name           string
		eventID        string
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name:           "Get Existing Event",
			eventID:        "event_123",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "event_123", response["id"])
				assert.Equal(t, "authorization_request", response["type"])
				assert.Equal(t, "demo_client", response["actor_id"])
				assert.Equal(t, "demo_user", response["resource_id"])
				assert.Equal(t, "authorize", response["action"])
				assert.Equal(t, "success", response["outcome"])
				assert.Contains(t, response, "timestamp")
				assert.Contains(t, response, "metadata")
			},
		},
		{
			name:           "Get Event with Empty ID",
			eventID:        "",
			expectedStatus: http.StatusMovedPermanently, // Router redirects when path doesn't match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/audit/events/" + tt.eventID
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				if tt.checkResponse != nil {
					tt.checkResponse(t, response)
				}
			}
		})
	}
}

func TestAuditHandler_GetComplianceReport(t *testing.T) {
	router, _, _, _ := setupOtherHandlersTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/compliance", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "2024-01", response["period"])
	assert.Equal(t, float64(1500), response["total_events"])
	assert.Equal(t, float64(1450), response["successful_authentications"])
	assert.Equal(t, float64(50), response["failed_authentications"])
	assert.Equal(t, 97.5, response["compliance_score"])
	assert.Contains(t, response, "violations")
	assert.Contains(t, response, "generated_at")

	// Check violations structure
	violations := response["violations"].([]interface{})
	assert.Greater(t, len(violations), 0)

	violation := violations[0].(map[string]interface{})
	assert.Equal(t, "rate_limit_exceeded", violation["type"])
	assert.Equal(t, float64(5), violation["count"])
	assert.Equal(t, "medium", violation["severity"])
}

func TestAuditHandler_GetAuditTrail(t *testing.T) {
	router, _, _, _ := setupOtherHandlersTestRouter(t)

	tests := []struct {
		name           string
		entityID       string
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name:           "Get Audit Trail for Entity",
			entityID:       "entity_123",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "entity_123", response["entity_id"])
				assert.Contains(t, response, "events")
				assert.Equal(t, float64(2), response["total_events"])

				events := response["events"].([]interface{})
				assert.Equal(t, 2, len(events))

				// Check first event
				event1 := events[0].(map[string]interface{})
				assert.Equal(t, "entity_creation", event1["action"])
				assert.Equal(t, "Legal entity created", event1["description"])
				assert.Equal(t, "system", event1["actor"])

				// Check second event
				event2 := events[1].(map[string]interface{})
				assert.Equal(t, "authorization_request", event2["action"])
				assert.Equal(t, "Authorization requested", event2["description"])
				assert.Equal(t, "demo_client", event2["actor"])
			},
		},
		{
			name:           "Get Audit Trail with Empty Entity ID",
			entityID:       "",
			expectedStatus: http.StatusNotFound, // Router will not match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/audit/trails/" + tt.entityID
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				if tt.checkResponse != nil {
					tt.checkResponse(t, response)
				}
			}
		})
	}
}

// Test RateHandler

func TestRateHandler_GetLimits(t *testing.T) {
	router, _, _, _ := setupOtherHandlersTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rate/limits", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "global")
	assert.Contains(t, response, "per_client")
	assert.Contains(t, response, "endpoints")

	// Check global limits
	global := response["global"].(map[string]interface{})
	assert.Equal(t, float64(1000), global["requests_per_minute"])
	assert.Equal(t, float64(10000), global["requests_per_hour"])
	assert.Equal(t, float64(100000), global["requests_per_day"])

	// Check per client limits
	perClient := response["per_client"].(map[string]interface{})
	assert.Equal(t, float64(60), perClient["requests_per_minute"])
	assert.Equal(t, float64(1000), perClient["requests_per_hour"])
	assert.Equal(t, float64(10000), perClient["requests_per_day"])

	// Check endpoint specific limits
	endpoints := response["endpoints"].(map[string]interface{})
	assert.Contains(t, endpoints, "/api/v1/auth/authorize")
	assert.Contains(t, endpoints, "/api/v1/auth/token")
}

func TestRateHandler_SetLimits(t *testing.T) {
	router, _, _, _ := setupOtherHandlersTestRouter(t)

	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name: "Set Valid Rate Limits",
			payload: map[string]interface{}{
				"client_id":           "test_client",
				"requests_per_minute": 100,
				"requests_per_hour":   2000,
				"requests_per_day":    20000,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "test_client", response["client_id"])
				assert.Equal(t, float64(100), response["requests_per_minute"])
				assert.Equal(t, float64(2000), response["requests_per_hour"])
				assert.Equal(t, float64(20000), response["requests_per_day"])
				assert.Contains(t, response, "updated_at")
			},
		},
		{
			name: "Set Rate Limits with Partial Data",
			payload: map[string]interface{}{
				"client_id":           "partial_client",
				"requests_per_minute": 50,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "partial_client", response["client_id"])
				assert.Equal(t, float64(50), response["requests_per_minute"])
			},
		},
		{
			name:           "Invalid JSON Payload",
			payload:        nil, // Will send invalid JSON
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Contains(t, response, "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var payloadBytes []byte
			var err error

			if tt.payload != nil {
				payloadBytes, err = json.Marshal(tt.payload)
				require.NoError(t, err)
			} else {
				payloadBytes = []byte("invalid json")
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/rate/limits", bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			
			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestRateHandler_GetStatus(t *testing.T) {
	router, _, _, _ := setupOtherHandlersTestRouter(t)

	tests := []struct {
		name           string
		clientID       string
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name:           "Get Rate Status for Client",
			clientID:       "test_client",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "test_client", response["client_id"])
				assert.Contains(t, response, "current_period")
				assert.Contains(t, response, "daily_stats")
				assert.Contains(t, response, "last_request")

				// Check current period
				currentPeriod := response["current_period"].(map[string]interface{})
				assert.Equal(t, float64(45), currentPeriod["requests_made"])
				assert.Equal(t, float64(60), currentPeriod["requests_limit"])
				assert.Equal(t, float64(15), currentPeriod["requests_remaining"])

				// Check daily stats
				dailyStats := response["daily_stats"].(map[string]interface{})
				assert.Equal(t, float64(1500), dailyStats["requests_made"])
				assert.Equal(t, float64(10000), dailyStats["requests_limit"])
				assert.Equal(t, float64(8500), dailyStats["requests_remaining"])
			},
		},
		{
			name:           "Get Rate Status with Empty Client ID",
			clientID:       "",
			expectedStatus: http.StatusNotFound, // Router will not match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/rate/status/" + tt.clientID
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				if tt.checkResponse != nil {
					tt.checkResponse(t, response)
				}
			}
		})
	}
}

// Test DemoHandler

func TestDemoHandler_GetScenarios(t *testing.T) {
	router, _, _, _ := setupOtherHandlersTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/demo/scenarios", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "scenarios")
}

func TestDemoHandler_RunScenario(t *testing.T) {
	router, _, _, _ := setupOtherHandlersTestRouter(t)

	tests := []struct {
		name           string
		scenarioID     string
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name:           "Run Valid Scenario",
			scenarioID:     "basic_auth",
			expectedStatus: http.StatusAccepted,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "basic_auth", response["scenario_id"])
				assert.Contains(t, response, "execution_id")
				assert.Equal(t, "running", response["status"])
				assert.Contains(t, response, "started_at")
				assert.Equal(t, float64(3), response["steps_total"])
				assert.Equal(t, float64(0), response["steps_completed"])

				// Check execution ID format
				executionID := response["execution_id"].(string)
				assert.Contains(t, executionID, "exec_basic_auth_")
			},
		},
		{
			name:           "Run Scenario with Empty ID",
			scenarioID:     "",
			expectedStatus: http.StatusBadRequest, // Handler checks for empty ID and returns bad request
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/demo/scenarios/" + tt.scenarioID + "/run"
			req := httptest.NewRequest(http.MethodPost, url, nil)
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus == http.StatusAccepted {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				if tt.checkResponse != nil {
					tt.checkResponse(t, response)
				}
			}
		})
	}
}

func TestDemoHandler_GetScenarioStatus(t *testing.T) {
	router, _, _, _ := setupOtherHandlersTestRouter(t)

	tests := []struct {
		name           string
		scenarioID     string
		expectedStatus int
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name:           "Get Scenario Status",
			scenarioID:     "basic_auth",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "basic_auth", response["scenario_id"])
				assert.Contains(t, response, "execution_id")
				assert.Equal(t, "completed", response["status"])
				assert.Contains(t, response, "started_at")
				assert.Contains(t, response, "completed_at")
				assert.Equal(t, float64(3), response["steps_total"])
				assert.Equal(t, float64(3), response["steps_completed"])
				assert.Contains(t, response, "steps")

				// Check steps structure
				steps := response["steps"].([]interface{})
				assert.Equal(t, 3, len(steps))

				// Check first step
				step1 := steps[0].(map[string]interface{})
				assert.Equal(t, "step_1", step1["id"])
				assert.Equal(t, "Authorization Request", step1["name"])
				assert.Equal(t, "completed", step1["status"])
				assert.Contains(t, step1, "result")

				step1Result := step1["result"].(map[string]interface{})
				assert.Contains(t, step1Result, "code")

				// Check second step
				step2 := steps[1].(map[string]interface{})
				assert.Equal(t, "step_2", step2["id"])
				assert.Equal(t, "Token Exchange", step2["name"])
				
				step2Result := step2["result"].(map[string]interface{})
				assert.Contains(t, step2Result, "access_token")

				// Check third step
				step3 := steps[2].(map[string]interface{})
				assert.Equal(t, "step_3", step3["id"])
				assert.Equal(t, "User Info Retrieval", step3["name"])
				
				step3Result := step3["result"].(map[string]interface{})
				assert.Contains(t, step3Result, "user_id")
			},
		},
		{
			name:           "Get Scenario Status with Empty ID",
			scenarioID:     "",
			expectedStatus: http.StatusBadRequest, // Handler checks for empty ID and returns bad request
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/demo/scenarios/" + tt.scenarioID + "/status"
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				
				if tt.checkResponse != nil {
					tt.checkResponse(t, response)
				}
			}
		})
	}
}

// Test handler creation

func TestNewAuditHandler(t *testing.T) {
	logger := logrus.New()
	service := createMockGAuthServiceForOther(t)

	handler := NewAuditHandler(service, logger)

	assert.NotNil(t, handler)
	assert.Equal(t, service, handler.service)
	assert.Equal(t, logger, handler.logger)
}

func TestNewRateHandler(t *testing.T) {
	logger := logrus.New()
	service := createMockGAuthServiceForOther(t)

	handler := NewRateHandler(service, logger)

	assert.NotNil(t, handler)
	assert.Equal(t, service, handler.service)
	assert.Equal(t, logger, handler.logger)
}

func TestNewDemoHandler(t *testing.T) {
	logger := logrus.New()
	service := createMockGAuthServiceForOther(t)

	handler := NewDemoHandler(service, logger)

	assert.NotNil(t, handler)
	assert.Equal(t, service, handler.service)
	assert.Equal(t, logger, handler.logger)
}

// Integration test for all handlers
func TestOtherHandlers_Integration(t *testing.T) {
	router, _, _, _ := setupOtherHandlersTestRouter(t)

	// Test multiple endpoints to ensure they work together
	endpoints := []struct {
		method   string
		path     string
		expected int
	}{
		{http.MethodGet, "/api/v1/audit/events", http.StatusOK},
		{http.MethodGet, "/api/v1/audit/compliance", http.StatusOK},
		{http.MethodGet, "/api/v1/rate/limits", http.StatusOK},
		{http.MethodGet, "/api/v1/demo/scenarios", http.StatusOK},
	}

	for _, endpoint := range endpoints {
		t.Run(fmt.Sprintf("%s %s", endpoint.method, endpoint.path), func(t *testing.T) {
			req := httptest.NewRequest(endpoint.method, endpoint.path, nil)
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, endpoint.expected, w.Code)
		})
	}
}
