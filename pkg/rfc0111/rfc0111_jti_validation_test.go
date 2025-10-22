package rfc0111

import (
	"context"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/token"
	"github.com/o1egl/paseto"
)

// helper to create basic service with creation privilege
func newBasicService() *Service {
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "p", Subject: "grantor", Resource: "*", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	return NewService(auditLogger, authorizer)
}

// Issue a delegation and return token plus envelope (by decrypting).
func issueAndDecrypt(t *testing.T, svc *Service) (string, token.Envelope) {
	t.Helper()
	resp, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "grantor", Grantee: "grantee", Scope: []string{"read"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("create delegation: %v", err)
	}
	var env token.Envelope
	if err := svc.decryptWithAnyKey(resp.AuthToken, &env); err != nil {
		t.Fatalf("decrypt token: %v", err)
	}
	return resp.AuthToken, env
}

func TestVerifyTokenMalformedJTI(t *testing.T) {
	svc := newBasicService()
	_, env := issueAndDecrypt(t, svc)
	if env.JTI == "" {
		t.Fatalf("expected jti present in issued token")
	}
	// Malform JTI (not a UUID v4) e.g. "not-a-uuid"
	env.JTI = "not-a-uuid"
	// Re-encrypt with same key
	// Re-encrypt using legacy tokenKey (will fail decrypt if key mismatch); prefer active key if available.
	activeKey := svc.tokenKey
	if svc.keyRing != nil && svc.keyRing.Active() != nil {
		activeKey = svc.keyRing.Active().Material
	}
	badTok, err := paseto.NewV2().Encrypt(activeKey, env, nil)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	_, verr := svc.VerifyToken(context.Background(), badTok)
	if verr == nil {
		t.Fatalf("expected invalid_request error for malformed jti")
	}
}

func TestVerifyTokenMissingJTIWithReplayProtection(t *testing.T) {
	svc := NewService(audit.NewMemoryLogger(nil), authz.NewMemoryAuthorizer(), WithReplayProtection(100, time.Minute))
	// authorize creation
	if ma, ok := svc.authz.(*authz.MemoryAuthorizer); ok {
		ma.AddPolicy(authz.Policy{ID: "p", Subject: "grantor", Resource: "*", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	}
	tok, env := issueAndDecrypt(t, svc)
	// Remove JTI
	env.JTI = ""
	activeKey := svc.tokenKey
	if svc.keyRing != nil && svc.keyRing.Active() != nil {
		activeKey = svc.keyRing.Active().Material
	}
	mutatedTok, err := paseto.NewV2().Encrypt(activeKey, env, nil)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	_, verr := svc.VerifyToken(context.Background(), mutatedTok)
	if verr == nil {
		t.Fatalf("expected invalid_request for missing jti under replay protection")
	}
	// Control: original token should verify OK
	if _, ok := svc.repo.Get(env.DelegationID); !ok {
		t.Fatalf("delegation missing")
	}
	if _, cerr := svc.VerifyToken(context.Background(), tok); cerr != nil {
		t.Fatalf("expected original token to verify: %v", cerr)
	}
}

func TestVerifyTokenValidUUIDv4Passes(t *testing.T) {
	svc := newBasicService()
	tok, _ := issueAndDecrypt(t, svc)
	// Just verify original token passes (UUID v4 generated internally)
	if _, err := svc.VerifyToken(context.Background(), tok); err != nil {
		t.Fatalf("verify failed for valid token: %v", err)
	}
}
