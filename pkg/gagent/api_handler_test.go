package gagent

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestRouter() (*gin.Engine, *APIHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAPIHandler()
	handler.RegisterRoutes(router)
	return router, handler
}

func TestListAgents(t *testing.T) {
	router, handler := setupTestRouter()

	// Register test agents
	agent1 := NewAgent("agent-1", "Agent 1", "gpt-4", "openai", 0.8)
	agent2 := NewAgent("agent-2", "Agent 2", "claude-3", "anthropic", 0.7)
	handler.RegisterAgent(agent1)
	handler.RegisterAgent(agent2)

	// Make request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/g-agent/agents", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response["success"].(bool) {
		t.Error("Expected success=true")
	}

	if count := response["count"].(float64); count != 2 {
		t.Errorf("Expected 2 agents, got %.0f", count)
	}
}

func TestGetAgentInfo(t *testing.T) {
	router, handler := setupTestRouter()

	agent := NewAgent("agent-1", "Test Agent", "gpt-4", "openai", 0.8)
	handler.RegisterAgent(agent)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/g-agent/agents/agent-1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response["success"].(bool) {
		t.Error("Expected success=true")
	}

	agentData := response["agent"].(map[string]interface{})
	if agentData["id"].(string) != "agent-1" {
		t.Errorf("Expected agent ID 'agent-1', got %s", agentData["id"].(string))
	}
}

func TestGetAgentInfoNotFound(t *testing.T) {
	router, _ := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/g-agent/agents/nonexistent", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestEnableDisableAgent(t *testing.T) {
	router, handler := setupTestRouter()

	agent := NewAgent("agent-1", "Test Agent", "gpt-4", "openai", 0.8)
	handler.RegisterAgent(agent)

	// Disable agent
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/g-agent/agents/agent-1/disable", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if agent.IsEnabled() {
		t.Error("Agent should be disabled")
	}

	// Enable agent
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/g-agent/agents/agent-1/enable", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !agent.IsEnabled() {
		t.Error("Agent should be enabled")
	}
}

func TestGetAgentMetrics(t *testing.T) {
	router, handler := setupTestRouter()

	agent := NewAgent("agent-1", "Test Agent", "gpt-4", "openai", 0.8)
	handler.RegisterAgent(agent)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/g-agent/agents/agent-1/metrics", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response["success"].(bool) {
		t.Error("Expected success=true")
	}

	if metrics, ok := response["metrics"].(map[string]interface{}); ok {
		if totalEval, ok := metrics["total_evaluations"].(float64); ok && totalEval != 0 {
			t.Error("Expected 0 initial evaluations")
		}
	}
}

func TestAPIEvaluateEnforcement(t *testing.T) {
	router, handler := setupTestRouter()

	agent := NewAgent("agent-1", "Test Agent", "gpt-4", "openai", 0.8)
	handler.RegisterAgent(agent)

	requestBody := map[string]interface{}{
		"agent_id": "agent-1",
		"subject":  "user:alice",
		"resource": "file:document.pdf",
		"action":   "read",
		"context": map[string]interface{}{
			"ip_address": "192.168.1.1",
		},
	}

	body, _ := json.Marshal(requestBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/g-agent/evaluate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response["success"].(bool) {
		t.Error("Expected success=true")
	}

	recommendation := response["recommendation"].(map[string]interface{})
	if recommendation["agent_id"].(string) != "agent-1" {
		t.Error("Recommendation should have agent_id")
	}

	if _, ok := recommendation["confidence"]; !ok {
		t.Error("Recommendation should have confidence")
	}

	if _, ok := recommendation["suggestion"]; !ok {
		t.Error("Recommendation should have suggestion")
	}
}

func TestEvaluateEnforcementInvalidRequest(t *testing.T) {
	router, handler := setupTestRouter()

	agent := NewAgent("agent-1", "Test Agent", "gpt-4", "openai", 0.8)
	handler.RegisterAgent(agent)

	// Missing required fields
	requestBody := map[string]interface{}{
		"agent_id": "agent-1",
		"subject":  "user:alice",
		// Missing resource and action
	}

	body, _ := json.Marshal(requestBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/g-agent/evaluate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestEvaluateBatch(t *testing.T) {
	router, handler := setupTestRouter()

	agent := NewAgent("agent-1", "Test Agent", "gpt-4", "openai", 0.8)
	handler.RegisterAgent(agent)

	requestBody := map[string]interface{}{
		"agent_id": "agent-1",
		"requests": []map[string]interface{}{
			{
				"subject":  "user:alice",
				"resource": "file:doc1.pdf",
				"action":   "read",
			},
			{
				"subject":  "user:bob",
				"resource": "file:doc2.pdf",
				"action":   "write",
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/g-agent/evaluate/batch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response["success"].(bool) {
		t.Error("Expected success=true")
	}

	results := response["results"].([]interface{})
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestHealthCheck(t *testing.T) {
	router, handler := setupTestRouter()

	agent1 := NewAgent("agent-1", "Agent 1", "gpt-4", "openai", 0.8)
	agent2 := NewAgent("agent-2", "Agent 2", "claude-3", "anthropic", 0.7)
	agent2.Disable()
	handler.RegisterAgent(agent1)
	handler.RegisterAgent(agent2)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/g-agent/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"].(string) != "healthy" {
		t.Error("Expected status='healthy'")
	}

	if response["total_agents"].(float64) != 2 {
		t.Error("Expected 2 total agents")
	}

	if response["enabled_agents"].(float64) != 1 {
		t.Error("Expected 1 enabled agent")
	}
}

func TestGetCapabilities(t *testing.T) {
	router, _ := setupTestRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/g-agent/capabilities", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response["success"].(bool) {
		t.Error("Expected success=true")
	}

	capabilities := response["capabilities"].(map[string]interface{})
	if _, ok := capabilities["enforcement_evaluation"]; !ok {
		t.Error("Expected enforcement_evaluation capability")
	}

	features := response["features"].([]interface{})
	if len(features) == 0 {
		t.Error("Expected features to be populated")
	}
}
