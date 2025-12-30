package gauth_aap_001

import (
	"context"
	"encoding/base64"
	"os"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

func TestEnvelopeCBORCompaction(t *testing.T) {
	// Enable CBOR embedding before creating service
	os.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
	os.Setenv("GAUTH_EMBED_FULL_POA_CBOR", "1")
	defer os.Unsetenv("GAUTH_POA_ENVELOPE_V2")
	defer os.Unsetenv("GAUTH_EMBED_FULL_POA_CBOR")

	// Setup service with audit logger and authorizer
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{
		ID:       "allow-all",
		Subject:  "*",
		Resource: "*",
		Actions:  []string{"*"},
		Effect:   authz.Allow,
	})
	svc := NewService(auditLogger, authorizer)

	// Create a delegation via the service
	dur := time.Hour
	req := DelegationRequest{
		Grantor:  "grantor",
		Grantee:  "grantee",
		Scope:    []string{"read", "write"},
		Duration: dur,
	}

	res, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("failed to create delegation: %v", err)
	}

	// Verify token with required subject context
	ctx := WithSubject(context.Background(), "grantee")
	vRes, err := svc.VerifyToken(ctx, res.AuthToken)
	if err != nil {
		t.Fatalf("failed to verify token: %v", err)
	}

	// Check if PoAVersion is CBOR
	if vRes.PoAVersion != poaVersionV1CBOR {
		t.Errorf("expected PoAVersion %s, got %s", poaVersionV1CBOR, vRes.PoAVersion)
	}

	// Check if RawPOA is set (base64 CBOR)
	if vRes.RawPOA == "" {
		t.Fatalf("RawPOA is empty")
	}

	// Verify the RawPOA is valid base64
	cborBytes, err := base64.StdEncoding.DecodeString(vRes.RawPOA)
	if err != nil {
		t.Fatalf("RawPOA is not valid base64: %v", err)
	}

	t.Logf("Successfully embedded CBOR PoA (size: %d bytes)", len(cborBytes))

	// Basic CBOR validation: check it starts with map header
	if len(cborBytes) == 0 || (cborBytes[0]>>5) != 5 {
		t.Errorf("CBOR does not start with map header")
	}
}

func TestStreamingLargePoAChainV2(t *testing.T) {
	// Enable chain embedding before creating service
	os.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
	os.Setenv("GAUTH_EMBED_RAW_POA_CHAIN", "1")
	defer os.Unsetenv("GAUTH_POA_ENVELOPE_V2")
	defer os.Unsetenv("GAUTH_EMBED_RAW_POA_CHAIN")

	// Setup service with audit logger and authorizer
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{
		ID:       "allow-all",
		Subject:  "*",
		Resource: "*",
		Actions:  []string{"*"},
		Effect:   authz.Allow,
	})
	svc := NewService(auditLogger, authorizer)

	// Create a delegation
	dur := time.Hour
	res, err := svc.CreateDelegation(DelegationRequest{
		Grantor:  "grantor",
		Grantee:  "grantee",
		Scope:    []string{"read"},
		Duration: dur,
	})
	if err != nil {
		t.Fatalf("failed to create delegation: %v", err)
	}

	// Verify token with required subject context
	ctx := WithSubject(context.Background(), "grantee")
	vRes, err := svc.VerifyToken(ctx, res.AuthToken)
	if err != nil {
		t.Fatalf("failed to verify token: %v", err)
	}

	// Check if RawPOAChain is present
	if vRes.RawPOAChain == "" {
		t.Errorf("expected RawPOAChain to be populated")
	}
	if vRes.RawPOAChainAlgo == "" {
		t.Errorf("expected RawPOAChainAlgo to be populated")
	}

	t.Logf("RawPOAChain present with algorithm: %s", vRes.RawPOAChainAlgo)
}
