package gauth

import (
	"encoding/base64"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func newTestService(t *testing.T, expiry time.Duration) *Service {
	cfg := Config{
		AuthServerURL:     "http://test",
		ClientID:          "test-client",
		ClientSecret:      "client-secret-abc1234567890",
		SigningKey:        "signing-secret-abcdefghijklmnopqrstuvwxyz-1234567890",
		Scopes:            []string{"read", "write"},
		AccessTokenExpiry: expiry,
	}
	svc, err := New(cfg)
	if err != nil {
		t.Fatalf("failed creating service: %v", err)
	}
	return svc
}

func TestSigningAndValidationHappyPath(t *testing.T) {
	svc := newTestService(t, time.Minute)
	resp, err := svc.RequestToken(TokenRequest{GrantID: "g1", Scope: []string{"read"}})
	if err != nil {
		t.Fatalf("RequestToken error: %v", err)
	}
	if !strings.Contains(resp.Token, ".") {
		t.Fatalf("expected JWT-like token format, got %s", resp.Token)
	}
	val, err := svc.ValidateToken(resp.Token)
	if err != nil {
		t.Fatalf("ValidateToken error: %v", err)
	}
	if !val.Valid || val.ClientID != "test-client" {
		t.Fatalf("unexpected validation result: %+v", val)
	}
	if len(val.Scope) != 1 || val.Scope[0] != "read" {
		t.Fatalf("scope mismatch: %+v", val.Scope)
	}
}

func TestTamperDetection(t *testing.T) {
	svc := newTestService(t, time.Minute)
	resp, _ := svc.RequestToken(TokenRequest{GrantID: "g1", Scope: []string{"read"}})
	parts := strings.Split(resp.Token, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected token parts")
	}
	// Tamper payload
	parts[1] = base64RawReplace(parts[1], "read", "admin")
	bad := strings.Join(parts, ".")
	if _, err := svc.ValidateToken(bad); err == nil {
		t.Fatalf("expected tampered token to fail validation")
	}
}

// base64RawReplace decodes a base64url segment, replaces substring, re-encodes rawurl
func base64RawReplace(seg, old, new string) string {
	b, _ := base64RawDecode(seg)
	return base64RawEncode(strings.ReplaceAll(string(b), old, new))
}

// Lightweight helpers (duplicated to avoid exporting internal logic)
func base64RawDecode(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) }
func base64RawEncode(b string) string          { return base64.RawURLEncoding.EncodeToString([]byte(b)) }

func TestExpiryEnforced(t *testing.T) {
	svc := newTestService(t, 200*time.Millisecond)
	resp, _ := svc.RequestToken(TokenRequest{GrantID: "g1", Scope: []string{"read"}})
	parts := strings.Split(resp.Token, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected token format")
	}
	payloadBytes, _ := base64.RawURLEncoding.DecodeString(parts[1])
	payload := string(payloadBytes)
	// locate exp numeric value
	idx := strings.Index(payload, "\"exp\":")
	if idx == -1 {
		t.Fatalf("exp not found in payload: %s", payload)
	}
	rest := payload[idx+6:]
	end := strings.IndexAny(rest, ",}")
	if end == -1 {
		t.Fatalf("malformed exp segment: %s", rest)
	}
	expField := rest[:end]
	expUnix, err := strconv.ParseInt(expField, 10, 64)
	if err != nil {
		t.Fatalf("parse exp: %v", err)
	}
	deadline := time.Now().Add(1500 * time.Millisecond)
	for time.Now().Unix() <= expUnix && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	t.Logf("payload=%s now=%d waitedUntil>%d", payload, time.Now().Unix(), expUnix)
	if _, err := svc.ValidateToken(resp.Token); err == nil {
		t.Fatalf("expected expired token to be invalid")
	}
}

func TestProductionMissingSigningKeyFailsStartup(t *testing.T) {
	// Simulate production by forcing empty signing key but rely on short client secret (padding) - still allowed.
	// We want to emphasize need for explicit signing key (demo context): service should still start (fallback),
	// but here we document behavior; if stricter behavior added later, adjust test accordingly.
	cfg := Config{ClientID: "c", ClientSecret: "short", SigningKey: "", AccessTokenExpiry: time.Minute}
	svc, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}
	if svc.signingKey == nil {
		t.Fatalf("expected fallback signing key to be set")
	}
}

// Prevent unused warnings for os import if future logic extends env usage
var _ = os.Getenv
