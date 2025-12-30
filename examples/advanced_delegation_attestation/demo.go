// Package demo provides reusable advanced delegation & attestation demonstration logic
package main

import (
	"fmt"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/agentauth"
)

// RunDemo runs the advanced delegation & attestation flow
func RunDemo() {
	// Initialize AgentAuth service with example config
	svc, err := agentauth.New(agentauth.Config{
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
	grant, err := svc.InitiateAuthorization(agentauth.AuthorizationRequest{
		ClientID: "test-client",
		Scopes:   []string{"sign_contract"},
	})
	if err != nil {
		panic(err)
	}

	// Step 2: Request a token for the delegated grant
	tokenResp, err := svc.RequestToken(agentauth.TokenRequest{
		GrantID: grant.GrantID,
		Scope:   grant.Scope,
	})
	if err != nil {
		panic(err)
	}

	// Step 3: Validate the issued token (attestation)
	tokenValid, err := svc.ValidateToken(tokenResp.Token)
	isValid := err == nil && tokenValid != nil && tokenValid.Valid
	fmt.Printf("Delegated token valid: %v, error: %v\n", isValid, err)
}
