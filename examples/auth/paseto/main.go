package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
)

func main() {
	// Prepare PASETO config (stub, not implemented)
	pasetoCfg := auth.PASETOConfig{
		Version:       "v2",
		Purpose:       "local",
		SymmetricKey:  []byte("32-byte-key-for-AES-256----------"),
		TokenValidity: time.Hour,
	}

	cfg := auth.Config{
		Type:        auth.TypePaseto,
		ExtraConfig: pasetoCfg,
	}
	authenticator, err := auth.NewAuthenticator(cfg)
	if err != nil {
		log.Fatalf("Failed to create PASETO authenticator: %v", err)
	}

	// Simulate token request (will fail, stub)
	tokenReq := auth.TokenRequest{
		GrantType: "paseto",
		Scopes:    []string{"read", "write"},
		Subject:   "user123",
	}
	tokenResp, err := authenticator.GenerateToken(context.Background(), tokenReq)
	if err != nil {
		log.Printf("PASETO token generation not implemented: %v", err)
		return
	}
	fmt.Printf("PASETO token: %s\n", tokenResp.Token)

	// Validate token (will fail, stub)
	tokenData, err := authenticator.ValidateToken(context.Background(), tokenResp.Token)
	if err != nil {
		log.Printf("PASETO token validation not implemented: %v", err)
		return
	}
	fmt.Printf("Token validated. Subject: %s, Scopes: %v\n", tokenData.Subject, tokenData.Scope)

}
