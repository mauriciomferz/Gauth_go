// Example: Advanced Delegation & Attestation Flow (RFC111-style, using canonical GAuth API)
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
	"github.com/mauriciomferz/Gauth_go/pkg/token"
)

func main() {
	// Generate a test RSA key for signing
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)

	// Initialize GAuth service
       svc, err := gauth.New(&gauth.Config{
	       AuthServerURL:     "https://example-auth-server",
	       ClientID:          "test-client",
	       ClientSecret:      "supersecret",
	       Scopes:            []string{"sign_contract"},
	       AccessTokenExpiry: 24 * time.Hour,
	       TokenConfig:       &token.Config{SigningMethod: token.RS256, SigningKey: priv},
       }, audit.NewLogger(100))
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize GAuth: %v", err))
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
	tokenData, err := svc.ValidateToken(tokenResp.Token)
	fmt.Printf("Delegated token valid: %v, error: %v\n", err == nil && tokenData != nil, err)
}
