package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
)

func main() {
	// Prepare OAuth2 config
	oauth2Cfg := auth.OAuth2Config{
		TokenURL:     "https://example.com/oauth/token", // Replace with real endpoint
		AuthorizeURL: "https://example.com/oauth/authorize", // Replace with real endpoint
		RedirectURL:  "http://localhost:8080/callback",
		Endpoints:    map[string]string{"introspection": "https://example.com/oauth/introspect"},
	}

	cfg := auth.Config{
		Type:        auth.TypeOAuth2,
		ClientID:    "your-client-id",
		ClientSecret: "your-client-secret",
		ExtraConfig: oauth2Cfg,
	}
	authenticator, err := auth.NewAuthenticator(cfg)
	if err != nil {
		log.Fatalf("Failed to create OAuth2 authenticator: %v", err)
	}

	// Simulate password grant
	tokenReq := auth.TokenRequest{
		GrantType: auth.GrantTypePassword,
		Scopes:    []string{"profile", "email"},
		Metadata: map[string]interface{}{
			"credentials": map[string]string{
				"Username": "user123",
				"Password": "testpass",
			},
		},
	}
	tokenResp, err := authenticator.GenerateToken(context.Background(), tokenReq)
	if err != nil {
		log.Fatalf("Token request failed: %v", err)
	}
	fmt.Printf("OAuth2 token: %s\n", tokenResp.Token)

	// Validate token (simulate)
	tokenData, err := authenticator.ValidateToken(context.Background(), tokenResp.Token)
	if err != nil {
		log.Fatalf("Token validation failed: %v", err)
	}
	fmt.Printf("Token validated. Subject: %s, Scopes: %v\n", tokenData.Subject, tokenData.Scope)
}
