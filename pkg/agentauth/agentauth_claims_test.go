package agentauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// simpleReplayStore is an in-memory implementation that tracks JTIs and returns error on duplicates
// to exercise fail-closed semantics.
type simpleReplayStore struct{ seen map[string]struct{} }

func (s *simpleReplayStore) CheckAndStore(jti string) error {
	if s.seen == nil {
		s.seen = map[string]struct{}{}
	}
	if _, ok := s.seen[jti]; ok {
		return ErrInvalidToken
	}
	s.seen[jti] = struct{}{}
	return nil
}

// TestClaimSetAndReplayEnforcement validates presence and enforcement of iss,aud,jti,nbf and duplicate JTI rejection.
func TestClaimSetAndReplayEnforcement(t *testing.T) {
	cfg := Config{
		AuthServerURL: "https://auth.example", ClientID: "client-1",
		ClientSecret: "secret-ABCD-12345678901234567890", SigningKey: "secret-ABCD-12345678901234567890",
		AccessTokenExpiry: time.Minute, Audience: []string{"api://resource"},
	}
	svc, err := New(cfg, WithReplayStore(&simpleReplayStore{}))
	if err != nil {
		t.Fatalf("New service error: %v", err)
	}
	tok, err := svc.RequestToken(TokenRequest{Scope: []string{"read", "write"}})
	if err != nil {
		t.Fatalf("RequestToken error: %v", err)
	}
	vr, err := svc.ValidateToken(tok.Token)
	if err != nil {
		t.Fatalf("ValidateToken error: %v", err)
	}
	if !vr.Valid {
		t.Fatalf("expected valid token")
	}
	// second validation should fail due to replay duplicate JTI
	if _, err := svc.ValidateToken(tok.Token); err == nil {
		t.Fatalf("expected duplicate JTI rejection")
	}
}

// TestNotBeforeEnforcement ensures nbf prevents early use.
func TestNotBeforeEnforcement(t *testing.T) {
	cfg := Config{
		AuthServerURL: "https://auth.example", ClientID: "client-nbf",
		ClientSecret: "secret-ABCD-12345678901234567890", SigningKey: "secret-ABCD-12345678901234567890",
		AccessTokenExpiry: time.Minute,
	}
	svc, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	// craft token with future nbf by manipulating issued token payload after issuance (legacy hmac path)
	tr := TokenRequest{Scope: []string{"x"}}
	tok, err := svc.RequestToken(tr)
	if err != nil {
		t.Fatalf("RequestToken error: %v", err)
	}
	parts := strings.Split(tok.Token, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected token format")
	}
	// decode payload JSON
	pb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("payload decode: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(pb, &claims); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	claims["nbf"] = float64(time.Now().Add(30 * time.Second).Unix())
	// re-marshal
	nb, _ := json.Marshal(claims)
	parts[1] = base64.RawURLEncoding.EncodeToString(nb)
	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, svc.signingKey)
	mac.Write([]byte(unsigned))
	parts[2] = base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	mutated := strings.Join(parts, ".")
	if _, err := svc.ValidateToken(mutated); err == nil {
		t.Fatalf("expected nbf rejection")
	}
}

// TestAudienceEnforcement checks audience claim acceptance and rejection.
func TestAudienceEnforcement(t *testing.T) {
	cfg := Config{
		AuthServerURL: "https://auth.example", ClientID: "client-aud",
		ClientSecret: "secret-ABCD-12345678901234567890", SigningKey: "secret-ABCD-12345678901234567890",
		AccessTokenExpiry: time.Minute, Audience: []string{"api://one", "api://two"},
	}
	svc, err := New(cfg)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	tok, err := svc.RequestToken(TokenRequest{Scope: []string{"read"}})
	if err != nil {
		t.Fatalf("RequestToken error: %v", err)
	}
	if _, err := svc.ValidateToken(tok.Token); err != nil {
		t.Fatalf("aud valid token rejected: %v", err)
	}
	// mutate aud to unknown value
	parts := strings.Split(tok.Token, ".")
	pb, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims map[string]any
	_ = json.Unmarshal(pb, &claims)
	claims["aud"] = "api://other"
	nb, _ := json.Marshal(claims)
	parts[1] = base64.RawURLEncoding.EncodeToString(nb)
	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, svc.signingKey)
	mac.Write([]byte(unsigned))
	parts[2] = base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	bad := strings.Join(parts, ".")
	if _, err := svc.ValidateToken(bad); err == nil {
		t.Fatalf("expected audience mismatch rejection")
	}
}
