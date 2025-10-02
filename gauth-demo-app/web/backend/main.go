package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"github.com/Gimel-Foundation/gauth/pkg/rfc"
)

// DemoScenario represents a demo scenario for the web app
type DemoScenario struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	RFCType     string                 `json:"rfc_type"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	Success  bool                   `json:"success"`
	Token    string                 `json:"token,omitempty"`
	Message  string                 `json:"message"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	RFCType  string                 `json:"rfc_type"`
}

var demoScenarios = []DemoScenario{
	{
		ID:          "rfc0111-basic",
		Name:        "RFC-0111 Basic GAuth 1.0",
		Description: "Basic RFC-0111 GAuth 1.0 scenario with P*P Architecture",
		Config: map[string]interface{}{
			"p2p_enabled":     true,
			"exclusions":      []string{"resource1", "resource2"},
			"extended_tokens": true,
			"ai_client":       false,
		},
		RFCType: "RFC-0111",
	},
	{
		ID:          "rfc0111-ai",
		Name:        "RFC-0111 AI Client",
		Description: "RFC-0111 with AI client capabilities enabled",
		Config: map[string]interface{}{
			"p2p_enabled":     true,
			"exclusions":      []string{},
			"extended_tokens": true,
			"ai_client":       true,
		},
		RFCType: "RFC-0111",
	},
	{
		ID:          "rfc0115-basic",
		Name:        "RFC-0115 Basic PoA Definition",
		Description: "Basic RFC-0115 Power of Attorney definition scenario",
		Config: map[string]interface{}{
			"parties": map[string]interface{}{
				"grantor": "User A",
				"grantee": "User B",
				"witness": "System",
			},
			"authorization_type": "limited",
			"legal_framework":    "standard",
		},
		RFCType: "RFC-0115",
	},
	{
		ID:          "rfc0115-advanced",
		Name:        "RFC-0115 Advanced PoA",
		Description: "Advanced RFC-0115 with complex authorization requirements",
		Config: map[string]interface{}{
			"parties": map[string]interface{}{
				"grantor": "Corporation A",
				"grantee": "Agent B",
				"witness": "Legal System",
				"notary":  "Certified Notary",
			},
			"authorization_type": "full",
			"legal_framework":    "enterprise",
		},
		RFCType: "RFC-0115",
	},
	{
		ID:          "combined-demo",
		Name:        "Combined RFC Demo",
		Description: "Demonstration of combined RFC-0111 and RFC-0115 functionality",
		Config: map[string]interface{}{
			"rfc0111": map[string]interface{}{
				"p2p_enabled":     true,
				"exclusions":      []string{"restricted"},
				"extended_tokens": true,
				"ai_client":       true,
			},
			"rfc0115": map[string]interface{}{
				"parties": map[string]interface{}{
					"grantor": "System",
					"grantee": "Client",
				},
				"authorization_type": "limited",
			},
		},
		RFCType: "Combined",
	},
}

// HealthStatus represents the health status response
type HealthStatus struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Service   string    `json:"service"`
	Version   string    `json:"version"`
}

// healthHandler handles health check requests for Kubernetes liveness probes
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	response := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Service:   "gauth-demo-backend",
		Version:   "1.0.0",
	}
	
	json.NewEncoder(w).Encode(response)
}

// readyHandler handles readiness check requests for Kubernetes readiness probes
func readyHandler(w http.ResponseWriter, r *http.Request) {
	// In a real application, you would check if the service is ready to serve traffic
	// For example: database connections, external service availability, etc.
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	response := HealthStatus{
		Status:    "ready",
		Timestamp: time.Now(),
		Service:   "gauth-demo-backend",
		Version:   "1.0.0",
	}
	
	json.NewEncoder(w).Encode(response)
}

func main() {
	router := mux.NewRouter()

	// CORS configuration
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	// Health endpoints for Kubernetes
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/ready", readyHandler).Methods("GET")
	router.HandleFunc("/healthz", healthHandler).Methods("GET")
	router.HandleFunc("/readyz", readyHandler).Methods("GET")

	// Routes
	router.HandleFunc("/scenarios", getScenariosHandler).Methods("GET")
	router.HandleFunc("/authenticate", authenticateHandler).Methods("POST")
	router.HandleFunc("/validate", validateHandler).Methods("POST")
	router.HandleFunc("/rfc0111/config", rfc0111ConfigHandler).Methods("POST")
	router.HandleFunc("/rfc0115/poa", rfc0115PoAHandler).Methods("POST")
	router.HandleFunc("/combined/demo", combinedDemoHandler).Methods("POST")

	// Serve static files
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("../frontend/")))

	handler := c.Handler(router)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

func getScenariosHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(demoScenarios)
}

func authenticateHandler(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	scenarioID, ok := req["scenario_id"].(string)
	if !ok {
		http.Error(w, "Missing scenario_id", http.StatusBadRequest)
		return
	}

	// Find scenario
	var scenario *DemoScenario
	for i := range demoScenarios {
		if demoScenarios[i].ID == scenarioID {
			scenario = &demoScenarios[i]
			break
		}
	}

	if scenario == nil {
		http.Error(w, "Scenario not found", http.StatusNotFound)
		return
	}

	// Generate mock token
	token := generateMockToken()

	response := AuthResponse{
		Success: true,
		Token:   token,
		Message: "Authentication successful for " + scenario.Name,
		Metadata: map[string]interface{}{
			"scenario":  scenario.Name,
			"rfc_type":  scenario.RFCType,
			"timestamp": time.Now().Unix(),
			"config":    scenario.Config,
		},
		RFCType: scenario.RFCType,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	token, ok := req["token"].(string)
	if !ok {
		http.Error(w, "Missing token", http.StatusBadRequest)
		return
	}

	// Mock validation
	isValid := strings.HasPrefix(token, "gauth_") && len(token) > 20

	response := map[string]interface{}{
		"valid":     isValid,
		"message":   getValidationMessage(isValid),
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func rfc0111ConfigHandler(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Create RFC-0111 configuration using the actual structure
	config := rfc.CreateRFC0111Config()
	
	// Apply user settings to the created config
	if p2pEnabled := getBoolFromMap(req, "p2p_enabled", true); p2pEnabled {
		config.Status = "p2p_enabled"
	}
	
	response := map[string]interface{}{
		"success":     true,
		"message":     "RFC-0111 configuration created successfully",
		"config":      config,
		"rfc_version": "RFC-0111",
		"timestamp":   time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func rfc0115PoAHandler(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Create RFC-0115 PoA definition using factory function
	poaDef := rfc.CreateRFC0115PoADefinition()
	
	// Apply user configurations (simplified demo)
	if authType := getStringFromMap(req, "authorization_type", ""); authType != "" {
		// Note: The actual structure is complex, this is a demo representation
		poaDef.GAuthContext.AIGovernanceLevel = authType
	}

	response := map[string]interface{}{
		"success":        true,
		"message":        "RFC-0115 Power of Attorney definition created successfully",
		"poa_definition": poaDef,
		"rfc_version":    "RFC-0115",
		"timestamp":      time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func combinedDemoHandler(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Create combined configuration using factory function
	combinedConfig := rfc.CreateCombinedRFCConfig()
	
	// Apply user configurations (simplified for demo)
	rfc0111Map, _ := req["rfc0111"].(map[string]interface{})
	rfc0115Map, _ := req["rfc0115"].(map[string]interface{})
	
	if len(rfc0111Map) > 0 {
		combinedConfig.IntegrationLevel = "rfc0111_enabled"
	}
	if len(rfc0115Map) > 0 {
		combinedConfig.IntegrationLevel = "combined_rfc"
	}

	// Validate combined configuration  
	if err := rfc.ValidateCombinedRFCConfig(combinedConfig); err != nil {
		http.Error(w, "Invalid combined configuration: "+err.Error(), http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"success":         true,
		"message":         "Combined RFC configuration validated successfully",
		"combined_config": combinedConfig,
		"rfc_versions":    []string{"RFC-0111", "RFC-0115"},
		"timestamp":       time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper functions
func generateMockToken() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return "gauth_" + string(b)
}

func getValidationMessage(isValid bool) string {
	if isValid {
		return "Token is valid"
	}
	return "Token is invalid"
}

func getBoolFromMap(m map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return defaultValue
}

func getStringFromMap(m map[string]interface{}, key string, defaultValue string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return defaultValue
}

