package rfc0111

import (
	"context"
	"encoding/base64"
	"os"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz"
)

// TestStrictAuthenticityMissingKey ensures missing public key transitions from soft skip to integrity failure when strict mode enabled.
func TestStrictAuthenticityMissingKey(t *testing.T) {
	// Soft mode service: explicitly disable strict authenticity env flag.
	os.Setenv("GAUTH_STRICT_AUTHENTICITY", "0")
	defer os.Unsetenv("GAUTH_STRICT_AUTHENTICITY")
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "p1", Subject: "alice", Resource: "*", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	mem := metrics.NewMemory()
	svcSoft := NewService(auditLogger, authorizer, WithMetrics(mem))

	resp, err := svcSoft.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"read"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("create delegation soft: %v", err)
	}
	// Inject synthetic signature with unknown key id
	st, ok := svcSoft.repo.Get(resp.POA.ID)
	if !ok {
		t.Fatalf("repo get")
	}
	dig, _, derr := CanonicalPOADigest(st)
	if derr != nil {
		t.Fatalf("digest: %v", derr)
	}
	st.Signature = &POASignature{Algorithm: algEd25519, KeyID: "unknown_kid", DigestHex: dig, SigBase64: base64.StdEncoding.EncodeToString(make([]byte, 64))}
	if err2 := svcSoft.ValidateDelegationCtx(context.Background(), resp.POA.ID, "bob", "read"); err2 != nil {
		t.Fatalf("soft mode should not fail on missing key: %v", err2)
	}
	// Strict mode service
	auditLogger2 := audit.NewMemoryLogger(nil)
	authorizer2 := authz.NewMemoryAuthorizer()
	authorizer2.AddPolicy(authz.Policy{ID: "p1", Subject: "alice", Resource: "*", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	mem2 := metrics.NewMemory()
	svcStrict := NewService(auditLogger2, authorizer2, WithMetrics(mem2), WithStrictAuthenticity())
	resp2, err := svcStrict.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"read"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("create delegation strict: %v", err)
	}
	st2, ok2 := svcStrict.repo.Get(resp2.POA.ID)
	if !ok2 {
		t.Fatalf("repo get strict")
	}
	dig2, _, derr2 := CanonicalPOADigest(st2)
	if derr2 != nil {
		t.Fatalf("digest strict: %v", derr2)
	}
	st2.Signature = &POASignature{Algorithm: algEd25519, KeyID: "unknown_kid", DigestHex: dig2, SigBase64: base64.StdEncoding.EncodeToString(make([]byte, 64))}
	if err := svcStrict.ValidateDelegationCtx(context.Background(), resp2.POA.ID, "bob", "read"); err == nil {
		t.Fatalf("strict mode expected integrity failure on missing key")
	}
}
