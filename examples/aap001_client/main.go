package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const apiBase = "http://localhost:8080/api/v1/aap-001"

// Example demonstrating how to use the AAP-001 REST API programmatically
func main() {
	fmt.Println("=== AAP-001 API Go Client Example ===")

	// Step I: Create Subscription
	fmt.Println("Step I: Creating subscription...")
	subID, err := createSubscription()
	if err != nil {
		panic(err)
	}
	fmt.Printf("✓ Subscription created: %s\n\n", subID)

	// Step II: Owner's Authorizer Authorization Proof
	fmt.Println("Step II: Verifying authorizer authorization...")
	err = executeStepII(subID)
	if err != nil {
		panic(err)
	}
	fmt.Println("✓ Step II completed")

	// Step III: Client Owner Identity Proof
	fmt.Println("Step III: Verifying client owner identity...")
	err = executeStepIII(subID)
	if err != nil {
		panic(err)
	}
	fmt.Println("✓ Step III completed")

	// Check final status
	fmt.Println("Checking subscription status...")
	status, err := getSubscriptionStatus(subID)
	if err != nil {
		panic(err)
	}
	fmt.Printf("✓ Subscription status: %s\n\n", status)

	fmt.Println("=== Demo completed successfully! ===")
}

func createSubscription() (string, error) {
	payload := map[string]interface{}{
		"owners_authorizer_id": "auth-12345",
		"identity_proof_request": map[string]interface{}{
			"subject_id":     "auth-12345",
			"identity_type":  "natural_person",
			"proof_method":   "pvp_token",
			"required_level": "substantial",
			"proof_data": map[string]interface{}{
				"pvp_token": "mock-pvp-token",
				"timestamp": time.Now().Format(time.RFC3339),
			},
		},
	}

	resp, err := makeRequest("POST", apiBase+"/subscriptions", payload)
	if err != nil {
		return "", err
	}

	subscriptionID, ok := resp["subscription_id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response: missing subscription_id")
	}

	return subscriptionID, nil
}

func executeStepII(subscriptionID string) error {
	payload := map[string]interface{}{
		"commercial_register_ref": "CR-12345-ABC",
		"jurisdiction":            "AT",
	}

	url := fmt.Sprintf("%s/subscriptions/%s/step-ii", apiBase, subscriptionID)
	_, err := makeRequest("POST", url, payload)
	return err
}

func executeStepIII(subscriptionID string) error {
	payload := map[string]interface{}{
		"subject_id":     "client-owner-001",
		"identity_type":  "natural_person",
		"proof_method":   "pvp_token",
		"required_level": "substantial",
		"proof_data": map[string]interface{}{
			"pvp_token": "mock-client-owner-token",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	url := fmt.Sprintf("%s/subscriptions/%s/step-iii", apiBase, subscriptionID)
	_, err := makeRequest("POST", url, payload)
	return err
}

func getSubscriptionStatus(subscriptionID string) (string, error) {
	url := fmt.Sprintf("%s/subscriptions/%s", apiBase, subscriptionID)
	resp, err := makeRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	status, ok := resp["status"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response: missing status")
	}

	return status, nil
}

func makeRequest(method, url string, payload interface{}) (map[string]interface{}, error) {
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}
