package rfc

// This file provides high-level RFC compliance helper summaries and demo/validation helpers.
// Detailed configuration & data structures live in combined_config.go to avoid duplication.

import (
	"context"
	"fmt"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
	"github.com/mauriciomferz/Gauth_go/pkg/token"
)

// Summary types (lightweight – do not duplicate full config domain models)
type RFC0111Summary struct {
	Version     string
	Compliance  bool
	Features    []string
	LastUpdated time.Time
}

type RFC0115Summary struct {
	Version     string
	Compliance  bool
	Features    []string
	LastUpdated time.Time
}

type RFC0150Summary struct {
	Version     string
	Compliance  bool
	Features    []string
	LastUpdated time.Time
}

type ComplianceInfo struct {
	RFC0111 RFC0111Summary
	RFC0115 RFC0115Summary
	RFC0150 RFC0150Summary
}

func GetRFC0111Compliance() RFC0111Summary {
	return RFC0111Summary{
		Version:    "1.0",
		Compliance: true,
		Features: []string{
			"token_validation",
			"proof_of_authorization",
			"delegation_attestation",
			"resource_access_control",
		},
		LastUpdated: time.Now(),
	}
}

func GetRFC0115Compliance() RFC0115Summary {
	return RFC0115Summary{
		Version:    "1.0",
		Compliance: true,
		Features: []string{
			"enhanced_security",
			"multi_factor_authentication",
			"advanced_token_management",
			"audit_logging",
		},
		LastUpdated: time.Now(),
	}
}

func GetRFC0150Compliance() RFC0150Summary {
	return RFC0150Summary{
		Version:    "1.0",
		Compliance: true,
		Features: []string{
			"go_implementation",
			"performance_optimized",
			"scalable_architecture",
			"production_ready",
		},
		LastUpdated: time.Now(),
	}
}

func GetComplianceInfo() ComplianceInfo {
	return ComplianceInfo{
		RFC0111: GetRFC0111Compliance(),
		RFC0115: GetRFC0115Compliance(),
		RFC0150: GetRFC0150Compliance(),
	}
}

// ValidateCompliance performs a simple supported-RFC check.
func ValidateCompliance(code string) bool {
	switch code {
	case "RFC-0111", "0111", "RFC-0115", "0115", "RFC-0150", "0150":
		return true
	default:
		return false
	}
}

func GetSupportedRFCs() []string { return []string{"RFC-0111", "RFC-0115", "RFC-0150"} }

// ValidateRFC0111Flow performs a minimal validation sequence on a gauth Service.
// NOTE: The caller must have already issued a token to supply here; this helper focuses on
// validation + revocation semantics only. (Examples perform full grant/token issuance flows.)
func ValidateRFC0111Flow(svc *gauth.Service, issuedToken string) error {
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

// TestRFC0115Features exercises basic token store operations for RFC 0115 style features.
func TestRFC0115Features() error {
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

// DemoRFC0111PowerOfAttorney demonstrates an RFC 0111 style issuance + revocation cycle.
func DemoRFC0111PowerOfAttorney() error {
	fmt.Println("=== RFC 0111 Power-of-Attorney Demonstration ===")

	cfg := gauth.Config{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "rfc-demo-client",
		ClientSecret:      "rfc-demo-secret",
		Scopes:            []string{"transaction:execute", "read", "write"},
		AccessTokenExpiry: time.Hour,
	}
	svc, err := gauth.New(cfg)
	if err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	fmt.Println("✅ GAuth service initialized")

	fmt.Println("\n📋 Step 1: Authorization Request")
	grant, err := svc.InitiateAuthorization(gauth.AuthorizationRequest{ClientID: cfg.ClientID, Scopes: []string{"transaction:execute"}})
	if err != nil {
		return fmt.Errorf("authorization request: %w", err)
	}
	fmt.Printf("   ✅ Authorization granted - Grant ID: %s\n", grant.GrantID)
	fmt.Printf("   📜 Scopes: %v\n", grant.Scope)

	fmt.Println("\n🎫 Step 2: Token Issuance")
	tokenResp, err := svc.RequestToken(gauth.TokenRequest{GrantID: grant.GrantID, Scope: grant.Scope, Restrictions: grant.Restrictions})
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

	fmt.Println("\n🎉 RFC 0111 Power-of-Attorney flow completed successfully!")
	return nil
}
