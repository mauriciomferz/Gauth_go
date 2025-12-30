package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OPAHTTPValidator validates scopes using OPA HTTP API (sidecar pattern)
type OPAHTTPValidator struct {
	baseURL    string
	httpClient *http.Client
}

// NewOPAHTTPValidator creates a validator that talks to OPA over HTTP
func NewOPAHTTPValidator(baseURL string) *OPAHTTPValidator {
	return &OPAHTTPValidator{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// OPARequest represents the request to OPA
type OPARequest struct {
	Input map[string]interface{} `json:"input"`
}

// OPAResponse represents the response from OPA
type OPAResponse struct {
	Result interface{} `json:"result"`
}

// ValidateScope checks if child scopes are covered by parent scopes
func (v *OPAHTTPValidator) ValidateScope(ctx context.Context, parent, child []string) error {
	// Build input
	input := map[string]interface{}{
		"action":        "validate_scope",
		"parent_scopes": parent,
		"child_scopes":  child,
	}

	// Create request
	reqBody := OPARequest{Input: input}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send to OPA
	url := fmt.Sprintf("%s/v1/data/agentauth/authz/allow", v.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("OPA request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OPA returned status %d: %s", resp.StatusCode, body)
	}

	// Parse response
	var opaResp OPAResponse
	if err := json.NewDecoder(resp.Body).Decode(&opaResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Check result
	allowed, ok := opaResp.Result.(bool)
	if !ok {
		return fmt.Errorf("unexpected result type: %T", opaResp.Result)
	}

	if !allowed {
		return fmt.Errorf("scope validation failed")
	}

	return nil
}

// ValidateWithDetails returns detailed decision information
func (v *OPAHTTPValidator) ValidateWithDetails(ctx context.Context, parent, child []string) (bool, map[string]interface{}, error) {
	input := map[string]interface{}{
		"action":        "validate_scope",
		"parent_scopes": parent,
		"child_scopes":  child,
	}

	reqBody := OPARequest{Input: input}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return false, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Query decision_details endpoint
	url := fmt.Sprintf("%s/v1/data/agentauth/authz/decision_details", v.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("OPA request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, nil, fmt.Errorf("OPA returned status %d: %s", resp.StatusCode, body)
	}

	var opaResp OPAResponse
	if err := json.NewDecoder(resp.Body).Decode(&opaResp); err != nil {
		return false, nil, fmt.Errorf("failed to decode response: %w", err)
	}

	details, ok := opaResp.Result.(map[string]interface{})
	if !ok {
		return false, nil, fmt.Errorf("unexpected result type: %T", opaResp.Result)
	}

	allowed := details["allow"].(bool)
	return allowed, details, nil
}

// HealthCheck verifies OPA is reachable
func (v *OPAHTTPValidator) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", v.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OPA health check failed: status %d", resp.StatusCode)
	}

	return nil
}

// Example: HTTP Client Usage
func exampleHTTPClient() {
	fmt.Println("=== Example: OPA HTTP Client (Sidecar) ===")

	// Connect to OPA sidecar (default: localhost:8181)
	validator := NewOPAHTTPValidator("http://localhost:8181")

	// Health check
	ctx := context.Background()
	if err := validator.HealthCheck(ctx); err != nil {
		fmt.Printf("⚠️  OPA not reachable: %v\n", err)
		fmt.Println("Start OPA with: docker run -p 8181:8181 openpolicyagent/opa:latest run --server")
		return
	}
	fmt.Println("✅ OPA sidecar is healthy")

	// Test validation
	parent := []string{"users:*"}
	child := []string{"users:read"}

	fmt.Printf("Parent: %v\n", parent)
	fmt.Printf("Child:  %v\n", child)

	err := validator.ValidateScope(ctx, parent, child)
	if err != nil {
		fmt.Printf("❌ Validation failed: %v\n", err)
	} else {
		fmt.Println("✅ Validation successful")
	}

	// Get detailed decision
	allowed, details, err := validator.ValidateWithDetails(ctx, parent, child)
	if err != nil {
		fmt.Printf("❌ Details query failed: %v\n", err)
		return
	}

	fmt.Printf("Allowed: %v\n", allowed)
	fmt.Println("Details:")
	prettyJSON, _ := json.MarshalIndent(details, "  ", "  ")
	fmt.Printf("  %s\n", prettyJSON)

	fmt.Println()
}
