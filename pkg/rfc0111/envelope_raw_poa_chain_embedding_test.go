package rfc0111

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"testing"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz"
	poaPkg "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/poa"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/token"
	"github.com/o1egl/paseto"
)

// TestRawPOAChainEmbeddingEnabled verifies RawPOAChain is populated when feature flags enabled.
func TestRawPOAChainEmbeddingEnabled(t *testing.T) {
	os.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
	os.Setenv("GAUTH_EMBED_FULL_POA", "1")
	os.Setenv("GAUTH_EMBED_RAW_POA_CHAIN", "1")
	defer func() {
		os.Unsetenv("GAUTH_POA_ENVELOPE_V2")
		os.Unsetenv("GAUTH_EMBED_FULL_POA")
		os.Unsetenv("GAUTH_EMBED_RAW_POA_CHAIN")
	}()
	mem := metrics.NewMemory()
	svc := NewService(newAuditMemory(), newAllowAllAuthorizer(), WithMetrics(mem))
	resp, err := svc.CreateDelegation(DelegationRequest{Grantor: "g1", Grantee: "u1", Scope: []string{"read"}, Duration: 3600_000_000_000})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if resp.AuthToken == "" {
		t.Fatalf("empty token")
	}
	// Decrypt token to inspect envelope
	holder, err := decryptEnvelopeV2ForTest(svc, resp.AuthToken)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if holder.RawPOAChain == "" {
		t.Fatalf("expected RawPOAChain populated")
	}
	chainBytes, decErr := base64.StdEncoding.DecodeString(holder.RawPOAChain)
	if decErr != nil {
		t.Fatalf("base64 decode: %v", decErr)
	}
	chain, cErr := poaPkg.DecodeRawPOAStreamWith(bytes.NewReader(chainBytes), poaPkg.DefaultStreamLimits, poaPkg.RawPOAHashSHA256, false)
	if cErr != nil {
		t.Fatalf("decode chain: %v", cErr)
	}
	if len(chain.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(chain.Items))
	}
	if chain.Items[0].Issuer != "g1" || chain.Items[0].Subject != "u1" {
		t.Fatalf("unexpected item fields")
	}
}

// TestRawPOAChainEmbeddingSizeLimit forces omission via tiny max bytes cap.
func TestRawPOAChainEmbeddingSizeLimit(t *testing.T) {
	os.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
	os.Setenv("GAUTH_EMBED_FULL_POA", "1")
	os.Setenv("GAUTH_EMBED_RAW_POA_CHAIN", "1")
	os.Setenv("GAUTH_MAX_RAW_POA_BYTES", "1") // force omission
	defer func() {
		os.Unsetenv("GAUTH_POA_ENVELOPE_V2")
		os.Unsetenv("GAUTH_EMBED_FULL_POA")
		os.Unsetenv("GAUTH_EMBED_RAW_POA_CHAIN")
		os.Unsetenv("GAUTH_MAX_RAW_POA_BYTES")
	}()
	mem := metrics.NewMemory()
	svc := NewService(newAuditMemory(), newAllowAllAuthorizer(), WithMetrics(mem))
	resp, err := svc.CreateDelegation(DelegationRequest{Grantor: "g2", Grantee: "u2", Scope: []string{"read"}, Duration: 3600_000_000_000})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	holder, err := decryptEnvelopeV2ForTest(svc, resp.AuthToken)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if holder.RawPOAChain != "" {
		t.Fatalf("expected RawPOAChain omitted due to size cap")
	}
	// Metrics counters are not asserted here because snapshot struct does not expose RawPOAChain embedding fields.
}

// Helper: minimal audit memory logger and authorizer references (reuse existing test utilities if present).
func newAuditMemory() *audit.MemoryLogger        { return audit.NewAuditLogger() }
func newAllowAllAuthorizer() *allowAllAuthorizer { return &allowAllAuthorizer{} }

type allowAllAuthorizer struct{}

func (a *allowAllAuthorizer) Authorize(ctx context.Context, req authz.Request) (authz.Decision, error) {
	return authz.Decision{Allow: true, Reason: "allow_all"}, nil
}

// decryptEnvelopeV2ForTest mirrors Verification logic to extract EnvelopeV2 for test assertions.
func decryptEnvelopeV2ForTest(s *Service, tok string) (*token.EnvelopeV2, error) {
	v2 := paseto.NewV2()
	var holder token.EnvelopeV2
	keys := [][]byte{s.tokenKey}
	if s.keyRing != nil && s.keyRing.Active() != nil {
		keys = append([][]byte{s.keyRing.Active().Material}, keys...)
	}
	var decErr error
	for _, k := range keys {
		if err := v2.Decrypt(tok, k, &holder, nil); err == nil {
			return &holder, nil
		} else {
			decErr = err
		}
	}
	return nil, decErr
}

// bytesReader wraps raw bytes with *bytes.Reader without new import noise in test file (use small inline helper).
// (helper removed; direct bytes.NewReader used)
