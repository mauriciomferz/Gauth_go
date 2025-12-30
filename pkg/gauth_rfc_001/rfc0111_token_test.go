package gauth_rfc_001

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
	"github.com/o1egl/paseto"
)

// TestGenerateAuthTokenClaims performs basic sanity checks on issued token & POA fields.
func TestGenerateAuthTokenClaims(t *testing.T) {
	memAuthz := authz.NewMemoryAuthorizer()
	memAuthz.AddPolicy(authz.Policy{ID: "test-allow-create", Subject: "alice", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(audit.NewMemoryLogger(nil), memAuthz)
	req := DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"transaction:execute"}, Duration: 5 * time.Minute, Restrictions: map[string]string{"max_amount": "100.00"}}
	dr, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("create delegation: %v", err)
	}
	token := dr.AuthToken
	if !strings.HasPrefix(token, "v2.local.") {
		t.Fatalf("token does not have paseto v2 local prefix: %s", token)
	}
	if dr.POA.ID == "" {
		t.Fatalf("delegation ID empty")
	}
	if dr.POA.Status != POAStatusActive {
		t.Fatalf("unexpected status: %s", dr.POA.Status)
	}
}

// TestValidationContextRestriction exercises ValidateDelegationRich with structured amount.
func TestValidationContextRestriction(t *testing.T) {
	memAuthz := authz.NewMemoryAuthorizer()
	memAuthz.AddPolicy(authz.Policy{ID: "test-allow-create", Subject: "alice", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(audit.NewMemoryLogger(nil), memAuthz)
	req := DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"transaction:execute"}, Duration: 5 * time.Minute, Restrictions: map[string]string{"max_amount": "25"}}
	dr, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("create delegation: %v", err)
	}
	tooHigh := 30.0
	vctx := ValidationContext{Action: "transaction:execute", RequestedAmount: &tooHigh}
	if err := svc.ValidateDelegationRich(context.Background(), dr.POA.ID, "bob", vctx); err == nil {
		t.Fatalf("expected restriction exceeded error")
	}
	// Within limit
	okAmt := 10.0
	vctx2 := ValidationContext{Action: "transaction:execute", RequestedAmount: &okAmt}
	if err := svc.ValidateDelegationRich(context.Background(), dr.POA.ID, "bob", vctx2); err != nil {
		t.Fatalf("unexpected error for allowed amount: %v", err)
	}
}

// TestTokenRoundTripEnvKey sets GAUTH_TOKEN_SYM_KEY to a fixed value, issues a token, decrypts it, and asserts fields.
// typedClaims mirrors the envelope JSON fields for decoding.
type typedClaims struct {
	Version      string            `json:"ver"`
	Kid          string            `json:"kid"`
	DelegationID string            `json:"delegation_id"`
	Grantor      string            `json:"grantor"`
	Grantee      string            `json:"grantee"`
	Scope        []string          `json:"scope"`
	Restrictions map[string]string `json:"restrictions"`
	Status       string            `json:"status"`
	IssuedAtRaw  string            `json:"iat"`
	ExpiresAtRaw string            `json:"exp"`
	IssChain     string            `json:"iss_chain_tip"`
	RevChain     string            `json:"rev_chain_tip"`
}

func TestTokenRoundTripEnvKey(t *testing.T) {
	// Fixed 32-byte key
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	os.Setenv("GAUTH_TOKEN_SYM_KEY", base64.StdEncoding.EncodeToString(key))
	defer os.Unsetenv("GAUTH_TOKEN_SYM_KEY")

	memAuthz := authz.NewMemoryAuthorizer()
	memAuthz.AddPolicy(authz.Policy{ID: "allow-create", Subject: "alice", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(audit.NewMemoryLogger(nil), memAuthz)
	req := DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"transaction:execute"}, Duration: 2 * time.Minute}
	dr, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !strings.HasPrefix(dr.AuthToken, "v2.local.") {
		t.Fatalf("missing paseto prefix")
	}

	// Attempt decrypt into typed struct
	var claims typedClaims
	if err2 := svc.decryptWithAnyKey(dr.AuthToken, &claims); err2 != nil {
		t.Fatalf("decrypt failed: %v", err2)
	}
	// Strict assertions
	if claims.DelegationID != dr.POA.ID {
		t.Fatalf("delegation_id mismatch: %s vs %s", claims.DelegationID, dr.POA.ID)
	}
	if claims.Grantor != dr.POA.Grantor {
		t.Fatalf("grantor mismatch: %s vs %s", claims.Grantor, dr.POA.Grantor)
	}
	if claims.Grantee != dr.POA.Grantee {
		t.Fatalf("grantee mismatch: %s vs %s", claims.Grantee, dr.POA.Grantee)
	}
	if len(claims.Scope) != len(dr.POA.Scope) {
		t.Fatalf("scope length mismatch: %d vs %d", len(claims.Scope), len(dr.POA.Scope))
	}
	if claims.Status != string(dr.POA.Status) {
		t.Fatalf("status mismatch: %s vs %s", claims.Status, dr.POA.Status)
	}
	if claims.Version == "" || !strings.HasPrefix(claims.Version, "gauth-aap001-env") {
		t.Fatalf("unexpected version field: %s", claims.Version)
	}
	if claims.Kid == "" {
		t.Fatalf("kid missing in token claims")
	}
	// Parse times and compare expiries within small tolerance
	issued, err := time.Parse(time.RFC3339Nano, claims.IssuedAtRaw)
	if err != nil {
		t.Fatalf("issued_at parse: %v", err)
	}
	exp, err := time.Parse(time.RFC3339Nano, claims.ExpiresAtRaw)
	if err != nil {
		t.Fatalf("expires_at parse: %v", err)
	}
	if exp.Sub(dr.POA.ValidUntil).Abs() > time.Millisecond*50 {
		t.Fatalf("expires_at drift >50ms: token=%s poa=%s", exp, dr.POA.ValidUntil)
	}
	if issued.After(exp) {
		t.Fatalf("issued_at after expires_at: %s > %s", issued, exp)
	}
}

// TestTokenTamperDetection mutates one byte of the token and expects decryption failure.
func TestTokenTamperDetection(t *testing.T) {
	memAuthz := authz.NewMemoryAuthorizer()
	memAuthz.AddPolicy(authz.Policy{ID: "allow-create", Subject: "alice", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(audit.NewMemoryLogger(nil), memAuthz)
	req := DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"transaction:execute"}, Duration: time.Minute}
	dr, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	tok := dr.AuthToken
	if !strings.HasPrefix(tok, "v2.local.") {
		t.Fatalf("expected v2.local prefix")
	}
	// Mutate a middle character (avoid header prefix for realistic tamper)
	b := []byte(tok)
	mid := len(b) / 2
	b[mid] = flipChar(b[mid])
	tampered := string(b)
	var claims typedClaims
	err = svc.decryptWithAnyKey(tampered, &claims)
	if err == nil {
		t.Fatalf("expected decrypt failure for tampered token")
	}
}

// flipChar returns a different ASCII character for tamper simulation.
func flipChar(c byte) byte {
	if c == 'A' {
		return 'B'
	}
	if c == '9' {
		return '8'
	}
	if c == '-' {
		return '_'
	}
	if c == '_' {
		return '-'
	}
	return c ^ 0x01 // minor bit flip for other bytes
}

// TestIssuanceChainIntegrity creates multiple delegations and verifies issuance chain length & integrity.
func TestIssuanceChainIntegrity(t *testing.T) {
	memAuthz := authz.NewMemoryAuthorizer()
	memAuthz.AddPolicy(authz.Policy{ID: "allow-create", Subject: "root", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(audit.NewMemoryLogger(nil), memAuthz)
	// Create several delegations
	total := 5
	for i := 0; i < total; i++ {
		req := DelegationRequest{Grantor: "root", Grantee: fmt.Sprintf("user%d", i), Scope: []string{"transaction:execute"}, Duration: time.Minute}
		if _, err := svc.CreateDelegation(req); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}
	if svc.issChain == nil {
		t.Fatalf("issuance chain nil")
	}
	// Verify length
	// Accessing private field directly for test; acceptable in same package.
	if len(svc.issChain.events) != total {
		t.Fatalf("expected %d issuance events got %d", total, len(svc.issChain.events))
	}
	if err := svc.issChain.Verify(); err != nil {
		t.Fatalf("issuance chain verify failed: %v", err)
	}
	// Tamper with one hash and verify detection
	svc.issChain.events[2].Hash = "0000" + svc.issChain.events[2].Hash[4:]
	if err := svc.issChain.Verify(); err == nil {
		t.Fatalf("expected verification failure after tamper")
	}
}

// TestKeyRotationIsolation verifies that tokens issued under an old key cannot be decrypted after rotation,
// and newly issued tokens require the new key.
func TestKeyRotationIsolation(t *testing.T) {
	memAuthz := authz.NewMemoryAuthorizer()
	memAuthz.AddPolicy(authz.Policy{ID: "allow-create", Subject: "alice", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(audit.NewMemoryLogger(nil), memAuthz)
	// Issue first token under initial key
	req := DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"transaction:execute"}, Duration: time.Minute}
	dr1, err := svc.CreateDelegation(req)
	if err != nil {
		t.Fatalf("create under old key: %v", err)
	}
	oldActive := append([]byte(nil), svc.keyRing.Active().Material...)
	oldID := svc.keyRing.Active().ID
	// Rotate using keyRing so previous key is retained
	svc.keyRing.Rotate()

	// Issue second token under new key
	req2 := DelegationRequest{Grantor: "alice", Grantee: "carol", Scope: []string{"transaction:execute"}, Duration: time.Minute}
	dr2, err := svc.CreateDelegation(req2)
	if err != nil {
		t.Fatalf("create under new key: %v", err)
	}

	// Decrypt first token after rotation (should succeed via previous key lookup)
	var c1 typedClaims
	if err := svc.decryptWithAnyKey(dr1.AuthToken, &c1); err != nil {
		t.Fatalf("old token decrypt after rotation failed: %v", err)
	}
	if c1.Kid != oldID {
		t.Fatalf("expected old kid %s got %s", oldID, c1.Kid)
	}
	// Direct decrypt with new active key should fail for old token
	if err := paseto.NewV2().Decrypt(dr1.AuthToken, svc.keyRing.Active().Material, &typedClaims{}, nil); err == nil {
		t.Fatalf("expected failure decrypting old token with new active key")
	}
	// Issue second token under new active key already done (dr2). Decrypt with active key
	var c2 typedClaims
	if err := paseto.NewV2().Decrypt(dr2.AuthToken, svc.keyRing.Active().Material, &c2, nil); err != nil {
		t.Fatalf("new token decrypt with active key failed: %v", err)
	}
	if c2.Kid == oldID {
		t.Fatalf("new token should not carry old key id")
	}
	// Ensure new token not decryptable with old key material directly
	if err := paseto.NewV2().Decrypt(dr2.AuthToken, oldActive, &typedClaims{}, nil); err == nil {
		t.Fatalf("expected failure decrypting new token with old key material")
	}
}

// Define struct matching subset of payload for decoding
