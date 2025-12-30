// Example: Advanced Delegation & Attestation Flow (RFC111-style, using canonical AgentAuth API)
package main

import (
	"fmt"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
)

func main() {
	// Initialize AgentAuth service
	svc, err := gauth.New(gauth.Config{
		AuthServerURL:     "https://example-auth-server",
		ClientID:          "test-client",
		ClientSecret:      "supersecret",
		Scopes:            []string{"sign_contract"},
		AccessTokenExpiry: 24 * time.Hour,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize AgentAuth: %v", err))
	}

	// Step 1: Initiate authorization (delegation)
	grant, err := svc.InitiateAuthorization(gauth.AuthorizationRequest{
		ClientID: "test-client",
		Scopes:   []string{"sign_contract"},
	})
	if err != nil {
		panic(err)
	}

	// Step 2: Request a token for the delegated grant
	tokenResp, err := svc.RequestToken(gauth.TokenRequest{
		GrantID: grant.GrantID,
		Scope:   grant.Scope,
	})
	if err != nil {
		panic(err)
	}

	// Step 3: Validate the issued token
	_, err = svc.ValidateToken(tokenResp.Token)
	isValid := err == nil
	fmt.Printf("Delegated token valid: %v, error: %v\n", isValid, err)
}
