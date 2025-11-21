package rfc0111

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz"
)

// embeddingTestService constructs a fresh RFC0111 service for embedding tests
func embeddingTestService() *Service {
	memAuthz := authz.NewMemoryAuthorizer()
	memAuthz.AddPolicy(authz.Policy{ID: "allow-all-alice", Subject: "alice", Resource: "*", Actions: []string{"*"}, Effect: authz.Allow})
	return NewService(audit.NewMemoryLogger(nil), memAuthz)
}

// TestEmbeddingRoundTrip tests the complete embedding workflow
func TestEmbeddingRoundTrip(t *testing.T) {
	os.Setenv("GAUTH_EMBED_FULL_POA", "1")
	os.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
	defer os.Unsetenv("GAUTH_EMBED_FULL_POA")
	defer os.Unsetenv("GAUTH_POA_ENVELOPE_V2")

	svc := embeddingTestService()
	ctx := context.Background()

	req := DelegationRequest{
		Grantor:  "alice",
		Grantee:  "bob",
		Scope:    []string{"finance:read", "finance:write"},
		Duration: 24 * time.Hour,
		Restrictions: map[string]string{
			"max_amount": "5000",
			"currency":   "USD",
		},
	}
	resp, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	result, err := svc.VerifyToken(ctx, resp.AuthToken)
	if err != nil {
		t.Fatalf("VerifyToken failed: %v", err)
	}

	if result.RawPOA == "" {
		t.Fatal("RawPOA not embedded despite GAUTH_EMBED_FULL_POA=1")
	}

	extracted, err := ExtractEmbeddedPoA(result)
	if err != nil {
		t.Fatalf("ExtractEmbeddedPoA failed: %v", err)
	}

	if extracted.ID != resp.POA.ID {
		t.Errorf("ID mismatch: expected %s, got %s", resp.POA.ID, extracted.ID)
	}
	if extracted.Grantor != resp.POA.Grantor {
		t.Errorf("Grantor mismatch: expected %s, got %s", resp.POA.Grantor, extracted.Grantor)
	}
	if extracted.Grantee != resp.POA.Grantee {
		t.Errorf("Grantee mismatch: expected %s, got %s", resp.POA.Grantee, extracted.Grantee)
	}
}

// TestEmbeddingSizeLimit tests that PoA exceeding size limit is not embedded
func TestEmbeddingSizeLimit(t *testing.T) {
	os.Setenv("GAUTH_EMBED_FULL_POA", "1")
	os.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
	os.Setenv("GAUTH_MAX_RAW_POA_BYTES", "100")
	defer os.Unsetenv("GAUTH_EMBED_FULL_POA")
	defer os.Unsetenv("GAUTH_POA_ENVELOPE_V2")
	defer os.Unsetenv("GAUTH_MAX_RAW_POA_BYTES")

	svc := embeddingTestService()
	ctx := context.Background()

	// Create large scope (within validation limits but exceeds embedding size)
	scope := make([]string, 30) // Under limit of 32
	for i := 0; i < 30; i++ {
		// Create unique scopes to avoid duplicate validation errors (VULN-01 security fix)
		scope[i] = "scope:" + strings.Repeat("a", 90) + ":item" + string(rune('a'+i))
	}

	req := DelegationRequest{
		Grantor:  "alice",
		Grantee:  "bob",
		Scope:    scope,
		Duration: 24 * time.Hour,
	}
	resp, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	result, err := svc.VerifyToken(ctx, resp.AuthToken)
	if err != nil {
		t.Fatalf("VerifyToken failed: %v", err)
	}

	if result.RawPOA != "" {
		t.Error("RawPOA embedded despite exceeding GAUTH_MAX_RAW_POA_BYTES")
	}

	_, err = ExtractEmbeddedPoA(result)
	if err == nil {
		t.Error("ExtractEmbeddedPoA should fail when RawPOA is empty")
	}
}

// TestOfflineVerification tests GAUTH_OFFLINE_VERIFICATION=1 mode
func TestOfflineVerification(t *testing.T) {
	os.Setenv("GAUTH_EMBED_FULL_POA", "1")
	os.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
	os.Setenv("GAUTH_OFFLINE_VERIFICATION", "1")
	defer os.Unsetenv("GAUTH_EMBED_FULL_POA")
	defer os.Unsetenv("GAUTH_POA_ENVELOPE_V2")
	defer os.Unsetenv("GAUTH_OFFLINE_VERIFICATION")

	svc := embeddingTestService()
	ctx := context.Background()

	req := DelegationRequest{
		Grantor:  "alice",
		Grantee:  "bob",
		Scope:    []string{"finance:read"},
		Duration: 24 * time.Hour,
	}
	resp, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	result, err := svc.VerifyToken(ctx, resp.AuthToken)
	if err != nil {
		t.Fatalf("VerifyToken failed: %v", err)
	}

	extracted, err := ExtractEmbeddedPoA(result)
	if err != nil {
		t.Fatalf("ExtractEmbeddedPoA failed: %v", err)
	}
	if extracted.ID != resp.POA.ID {
		t.Errorf("Offline verification used wrong PoA: expected %s, got %s", resp.POA.ID, extracted.ID)
	}
}

// TestBackwardCompatibility tests tokens without RawPOA still work
func TestBackwardCompatibility(t *testing.T) {
	os.Unsetenv("GAUTH_EMBED_FULL_POA")
	os.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
	defer os.Unsetenv("GAUTH_POA_ENVELOPE_V2")

	svc := embeddingTestService()
	ctx := context.Background()

	req := DelegationRequest{
		Grantor:  "alice",
		Grantee:  "bob",
		Scope:    []string{"finance:read"},
		Duration: 24 * time.Hour,
	}
	resp, err := svc.CreateDelegationCtx(ctx, req)
	if err != nil {
		t.Fatalf("CreateDelegation failed: %v", err)
	}

	result, err := svc.VerifyToken(ctx, resp.AuthToken)
	if err != nil {
		t.Fatalf("VerifyToken failed: %v", err)
	}

	if result.RawPOA != "" {
		t.Error("RawPOA embedded despite GAUTH_EMBED_FULL_POA not set")
	}

	_, err = ExtractEmbeddedPoA(result)
	if err == nil {
		t.Error("ExtractEmbeddedPoA should fail when RawPOA is empty")
	}
}

// TestExtractEmbeddedPoA_NilResult tests error handling
func TestExtractEmbeddedPoA_NilResult(t *testing.T) {
	_, err := ExtractEmbeddedPoA(nil)
	if err == nil {
		t.Error("ExtractEmbeddedPoA should fail with nil result")
	}
}

// TestExtractEmbeddedPoA_IDMismatch tests ID validation
func TestExtractEmbeddedPoA_IDMismatch(t *testing.T) {
	poa := PowerOfAttorney{
		ID:         "embedded-id",
		Grantor:    "alice",
		Grantee:    "bob",
		Scope:      []string{"finance:read"},
		ValidUntil: time.Now().Add(24 * time.Hour),
		Status:     POAStatusActive,
	}
	rawPOA, _ := json.Marshal(poa)
	result := &TokenVerificationResult{
		DelegationID: "envelope-id",
		RawPOA:       string(rawPOA),
	}
	_, err := ExtractEmbeddedPoA(result)
	if err == nil {
		t.Error("ExtractEmbeddedPoA should fail with ID mismatch")
	}
	if !strings.Contains(err.Error(), "id mismatch") {
		t.Errorf("unexpected error message: %v", err)
	}
}
