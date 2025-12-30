// Package main demonstrates GNAP client usage with AgentAuth.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

const baseURL = "http://localhost:8080"

func main() {
	// Step 1: Discover GNAP endpoints
	fmt.Println("=== Step 1: Discovery ===")
	discovery, err := discover()
	if err != nil {
		log.Fatalf("Discovery failed: %v", err)
	}
	fmt.Printf("Grant endpoint: %s\n\n", discovery["grant_request_endpoint"])

	// Step 2: Request a grant
	fmt.Println("=== Step 2: Grant Request ===")
	grantResp, err := requestGrant()
	if err != nil {
		log.Fatalf("Grant request failed: %v", err)
	}

	token := grantResp["access_token"].(map[string]interface{})
	tokenValue := token["value"].(string)
	fmt.Printf("Access Token: %s\n", tokenValue)
	fmt.Printf("Expires In: %.0f seconds\n\n", token["expires_in"].(float64))

	// Step 3: Use the token
	fmt.Println("=== Step 3: Use Token ===")
	fmt.Printf("Authorization: GNAP %s\n\n", tokenValue)

	// Step 4: Token management (optional)
	if cont, ok := grantResp["continue"].(map[string]interface{}); ok {
		fmt.Println("=== Step 4: Continuation Available ===")
		fmt.Printf("Continue URI: %s\n", cont["uri"])
		fmt.Printf("Continue Token: %s\n", cont["access_token"].(map[string]interface{})["value"])
	}

	fmt.Println("\n✓ GNAP flow complete!")
}

func discover() (map[string]interface{}, error) {
	resp, err := http.Get(baseURL + "/.well-known/gnap-as-rs")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func requestGrant() (map[string]interface{}, error) {
	// Build grant request
	grantReq := map[string]interface{}{
		"access_token": map[string]interface{}{
			"access": []map[string]interface{}{
				{
					"type":    "gauth-api",
					"actions": []string{"read", "write"},
				},
			},
		},
		"client": map[string]interface{}{
			"display": map[string]interface{}{
				"name": "GNAP Example Client",
			},
		},
	}

	body, _ := json.Marshal(grantReq)
	resp, err := http.Post(
		baseURL+"/gnap/tx",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: %s", body)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}
