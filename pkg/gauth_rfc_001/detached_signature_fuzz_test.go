//go:build go1.18

package gauth_rfc_001

// Fuzz harness for detached signature issuance + verification path stability.
// Ensures that for arbitrary (bounded) grantor/grantee/scope inputs the detached signature validates
// when feature flags are enabled and canonicalization remains deterministic.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
	cr "github.com/mauriciomferz/Gauth_go/pkg/crypto"
)

func FuzzDetachedSignatureIssueVerify(f *testing.F) {
	// Seeds (simple, valid).
	f.Add("alice", "bob", "read,write")
	f.Add("grantor", "grantee", "read")
	f.Add("g", "u", "read,custom")
	f.Fuzz(func(t *testing.T, grantor, grantee, scopeCSV string) {
		// Basic sanitation / bounding to avoid pathological memory or empty principals.
		if len(grantor) == 0 || len(grantor) > 32 || len(grantee) == 0 || len(grantee) > 32 {
			return
		}
		scopesRaw := strings.Split(scopeCSV, ",")
		scopes := make([]string, 0, len(scopesRaw))
		for _, s := range scopesRaw {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			if len(scopes) >= 5 {
				break
			} // cap
			// Restrict token length to keep canonical JSON small.
			if len(s) > 20 {
				s = s[:20]
			}
			scopes = append(scopes, s)
		}
		if len(scopes) == 0 {
			scopes = []string{"read"}
		}
		t.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
		t.Setenv("GAUTH_DETACHED_SIGNATURE", "1")
		kp, err := cr.NewInMemoryEd25519Provider()
		if err != nil {
			t.Fatalf("key provider: %v", err)
		}
		auditLogger := audit.NewMemoryLogger(nil)
		authorizer := authz.NewMemoryAuthorizer()
		authorizer.AddPolicy(authz.Policy{ID: "p", Subject: grantor, Resource: "*", Actions: []string{"create_delegation"}, Effect: authz.Allow})
		svc := NewService(auditLogger, authorizer, WithSignerProvider(kp.ActiveSigner), WithKeyProvider(kp))
		resp, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: grantor, Grantee: grantee, Scope: scopes, Duration: time.Minute})
		if err != nil {
			// Invalid inputs (e.g., validation) are not interesting for this fuzz objective.
			return
		}
		vres, verr := svc.VerifyToken(WithSubject(context.Background(), grantee), resp.AuthToken)
		if verr != nil {
			t.Fatalf("verify token: %v", verr)
		}
		if !vres.DetachedSignatureValid {
			t.Fatalf("detached signature failed to validate for generated case: grantor=%q grantee=%q scopes=%v", grantor, grantee, scopes)
		}
	})
}
