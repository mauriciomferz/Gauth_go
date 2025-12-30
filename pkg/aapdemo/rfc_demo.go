package aapdemo

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/agentauth"
	"github.com/mauriciomferz/AgentAuth/pkg/token"
)

// ValidateAAP001Flow performs a minimal validation sequence on a agentauth Service.
// NOTE: The caller must have already issued a token to supply here; this helper focuses on
// validation + revocation semantics only. (Examples perform full grant/token issuance flows.)
func ValidateAAP001Flow(svc *agentauth.Service, issuedToken string) error {
	if svc == nil {
		return fmt.Errorf("service cannot be nil")
	}
	if issuedToken == "" {
		return fmt.Errorf("issued token required")
	}

	vr, err := svc.ValidateToken(issuedToken)
	if err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}
	if vr == nil || !vr.Valid {
		return fmt.Errorf("expected token to be valid")
	}

	if revokeErr := svc.InvalidateToken(issuedToken); revokeErr != nil {
		return fmt.Errorf("token revocation failed: %w", revokeErr)
	}

	vr, err = svc.ValidateToken(issuedToken)
	if err != nil {
		return fmt.Errorf("post-revocation validation failed: %w", err)
	}
	if vr != nil && vr.Valid {
		return fmt.Errorf("token should be invalid after revocation")
	}
	return nil
}

// TestAAP002Features exercises basic token store operations for AAP-002 style features.
func TestAAP002Features() error {
	store := token.NewMemoryStore()
	ctx := context.Background()

	t := &token.Token{
		ID:        "test-token-115",
		Value:     "test-value",
		ExpiresAt: time.Now().Add(time.Hour),
		Scopes:    []string{"read", "write"},
	}
	if err := store.Save(ctx, t.ID, t); err != nil {
		return fmt.Errorf("token save: %w", err)
	}
	got, err := store.Get(ctx, t.ID)
	if err != nil {
		return fmt.Errorf("token get: %w", err)
	}
	if got.ID != t.ID {
		return fmt.Errorf("retrieved token mismatch")
	}
	if err := store.Revoke(ctx, t.ID, "test-revocation"); err != nil {
		return fmt.Errorf("token revoke: %w", err)
	}
	return nil
}

// DemoAAP001PowerOfAttorney demonstrates an AAP-001 style issuance + revocation cycle.
func DemoAAP001PowerOfAttorney() error {
	fmt.Println("=== AAP-001 Power-of-Attorney Demonstration ===")

	cfg := agentauth.Config{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "AAP-demo-client",
		ClientSecret:      "AAP-demo-secret",
		Scopes:            []string{"transaction:execute", "read", "write"},
		AccessTokenExpiry: time.Hour,
	}
	svc, err := agentauth.New(cfg)
	if err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	fmt.Println("✅ AgentAuth service initialized")

	fmt.Println("\n📋 Step 1: Authorization Request")
	grant, err := svc.InitiateAuthorization(agentauth.AuthorizationRequest{ClientID: cfg.ClientID, Scopes: []string{"transaction:execute"}})
	if err != nil {
		return fmt.Errorf("authorization request: %w", err)
	}
	fmt.Printf("   ✅ Authorization granted - Grant ID: %s\n", grant.GrantID)
	fmt.Printf("   📜 Scopes: %v\n", grant.Scope)

	fmt.Println("\n🎫 Step 2: Token Issuance")
	tokenResp, err := svc.RequestToken(agentauth.TokenRequest{GrantID: grant.GrantID, Scope: grant.Scope, Restrictions: grant.Restrictions})
	if err != nil {
		return fmt.Errorf("token request: %w", err)
	}
	fmt.Printf("   ✅ Token issued successfully\n")
	if len(tokenResp.Token) > 20 {
		fmt.Printf("   🔑 Token: %s...\n", tokenResp.Token[:20])
	} else {
		fmt.Printf("   🔑 Token: %s\n", tokenResp.Token)
	}
	fmt.Printf("   ⏰ Expires: %s\n", tokenResp.ValidUntil.Format(time.RFC3339))

	fmt.Println("\n💳 Step 3: Transaction Processing (stub)")
	fmt.Printf("   ✅ Transaction processed successfully\n")
	fmt.Printf("   📊 Result: %s\n", "success")

	fmt.Println("\n🚫 Step 4: Token Revocation")
	if err := svc.InvalidateToken(tokenResp.Token); err != nil {
		return fmt.Errorf("token revocation: %w", err)
	}
	fmt.Printf("   ✅ Token revoked successfully\n")

	if vr, err := svc.ValidateToken(tokenResp.Token); err != nil {
		return fmt.Errorf("post-revocation validation: %w", err)
	} else if vr != nil && vr.Valid {
		return fmt.Errorf("token should be invalid after revocation")
	}
	fmt.Printf("   ✅ Token validation confirmed - token is invalid\n")

	fmt.Println("\n🎉 AAP-001 Power-of-Attorney flow completed successfully!")
	return nil
}
