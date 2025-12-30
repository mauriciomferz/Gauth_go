package gauth_aap_001

import (
	"os"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
	"github.com/mauriciomferz/AgentAuth/pkg/token"
	"github.com/o1egl/paseto"
)

// TestRawPOAEmbeddingEnabled verifies RawPOA & PoAVersion populated when AGENTAUTH_EMBED_FULL_POA=1 and within size limit.
func TestRawPOAEmbeddingEnabled(t *testing.T) {
	os.Setenv("AGENTAUTH_POA_ENVELOPE_V2", "1")
	os.Setenv("AGENTAUTH_EMBED_FULL_POA", "1")
	os.Unsetenv("AGENTAUTH_MAX_RAW_POA_BYTES") // use default 8192
	defer func() {
		os.Unsetenv("AGENTAUTH_POA_ENVELOPE_V2")
		os.Unsetenv("AGENTAUTH_EMBED_FULL_POA")
		os.Unsetenv("AGENTAUTH_MAX_RAW_POA_BYTES")
	}()
	authzMem := authz.NewMemoryAuthorizer()
	// Allow policy for create_delegation
	authzMem.AddPolicy(authz.Policy{ID: "allow_create", Subject: "g1", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(auditLoggerForTest(), authzMem)
	req := DelegationRequest{Grantor: "g1", Grantee: "g2", Scope: []string{"a"}, Restrictions: map[string]string{"x": "y"}, Duration: time.Hour}
	resp, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("CreateDelegation error: %v", err)
	}
	// Decrypt token and inspect envelope V2 fields.
	// We reuse VerifyToken to parse; it will look up POA but that's fine.
	ctx := WithSubject(testCtx(), "g2")
	vr, err := svc.VerifyToken(ctx, resp.AuthToken)
	if err != nil {
		t.Fatalf("VerifyToken error: %v", err)
	}
	if vr.DelegationID == "" {
		t.Fatalf("delegation id missing")
	}
	// RawPOA embedding not directly exposed via verification result; we need to decrypt manually.
	// Minimal manual decryption replicating logic from VerifyToken for inspection.
	v2 := paseto.NewV2()
	var holder token.EnvelopeV2
	// Try keyRing active then legacy tokenKey via service internals.
	key := svc.tokenKey
	if svc.keyRing != nil && svc.keyRing.Active() != nil {
		key = svc.keyRing.Active().Material
	}
	if err := v2.Decrypt(resp.AuthToken, key, &holder, nil); err != nil {
		t.Fatalf("decrypt v2 failed: %v", err)
	}
	if holder.RawPOA == "" {
		t.Fatalf("expected RawPOA populated")
	}
	if holder.PoAVersion != poaVersionV1 {
		t.Fatalf("expected PoAVersion poa/v1 got %s", holder.PoAVersion)
	}
	if holder.CanonicalDigest == "" {
		t.Fatalf("expected canonical digest")
	}
}

// TestRawPOAEmbeddingDisabled verifies RawPOA omitted when AGENTAUTH_EMBED_FULL_POA is not enabled.
func TestRawPOAEmbeddingDisabled(t *testing.T) {
	os.Setenv("AGENTAUTH_POA_ENVELOPE_V2", "1")
	os.Unsetenv("AGENTAUTH_EMBED_FULL_POA")
	defer func() { os.Unsetenv("AGENTAUTH_POA_ENVELOPE_V2") }()
	authzMem := authz.NewMemoryAuthorizer()
	authzMem.AddPolicy(authz.Policy{ID: "allow_create", Subject: "a", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(auditLoggerForTest(), authzMem)
	req := DelegationRequest{Grantor: "a", Grantee: "b", Scope: []string{"w"}, Duration: time.Hour}
	resp, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("CreateDelegation error: %v", err)
	}
	v2 := paseto.NewV2()
	var holder token.EnvelopeV2
	key := svc.tokenKey
	if svc.keyRing != nil && svc.keyRing.Active() != nil {
		key = svc.keyRing.Active().Material
	}
	if err := v2.Decrypt(resp.AuthToken, key, &holder, nil); err != nil {
		t.Fatalf("decrypt v2 failed: %v", err)
	}
	if holder.RawPOA != "" {
		t.Fatalf("expected RawPOA empty when embedding disabled")
	}
	if holder.PoAVersion != "" {
		t.Fatalf("expected PoAVersion empty when embedding disabled")
	}
}

// TestRawPOAEmbeddingSizeLimit verifies omission when canonical JSON exceeds configured size limit.
func TestRawPOAEmbeddingSizeLimit(t *testing.T) {
	os.Setenv("AGENTAUTH_POA_ENVELOPE_V2", "1")
	os.Setenv("AGENTAUTH_EMBED_FULL_POA", "1")
	// Set very small max size to force omission.
	os.Setenv("AGENTAUTH_MAX_RAW_POA_BYTES", "64")
	defer func() {
		os.Unsetenv("AGENTAUTH_POA_ENVELOPE_V2")
		os.Unsetenv("AGENTAUTH_EMBED_FULL_POA")
		os.Unsetenv("AGENTAUTH_MAX_RAW_POA_BYTES")
	}()
	authzMem := authz.NewMemoryAuthorizer()
	authzMem.AddPolicy(authz.Policy{ID: "allow_create", Subject: "g", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(auditLoggerForTest(), authzMem)
	// Create a POA with enough data to exceed 64 bytes canonical JSON (multiple scope items & restrictions)
	restrictions := map[string]string{"a": "b", "c": "d", "e": "f"}
	req := DelegationRequest{Grantor: "g", Grantee: "h", Scope: []string{"one", "two", "three"}, Restrictions: restrictions, Duration: time.Hour}
	resp, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("CreateDelegation error: %v", err)
	}
	v2 := paseto.NewV2()
	var holder token.EnvelopeV2
	key := svc.tokenKey
	if svc.keyRing != nil && svc.keyRing.Active() != nil {
		key = svc.keyRing.Active().Material
	}
	if err := v2.Decrypt(resp.AuthToken, key, &holder, nil); err != nil {
		t.Fatalf("decrypt v2 failed: %v", err)
	}
	if holder.RawPOA != "" {
		t.Fatalf("expected RawPOA omitted due to size limit")
	}
	if holder.PoAVersion != "" {
		t.Fatalf("expected PoAVersion omitted when embedding skipped")
	}
}

// test helpers (reuse minimal subset from existing tests if available)
func auditLoggerForTest() *audit.MemoryLogger { return audit.NewMemoryLogger(nil) }
func testCtx() *testingContext                { return &testingContext{} }

// Provide a minimal context implementation for VerifyToken path (without cancellation).
type testingContext struct{}

func (t *testingContext) Deadline() (time.Time, bool)       { return time.Time{}, false }
func (t *testingContext) Done() <-chan struct{}             { return nil }
func (t *testingContext) Err() error                        { return nil }
func (t *testingContext) Value(key interface{}) interface{} { return nil }
