package gauth

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/crypto"
)

func TestEdDSATokenIssueAndValidate(t *testing.T) {
	os.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	svc, err := New(Config{AuthServerURL: "http://localhost", ClientID: "client-1", ClientSecret: "secret", AccessTokenExpiry: time.Minute})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	resp, err := svc.RequestToken(TokenRequest{GrantID: "g1", Scope: []string{"read"}})
	if err != nil {
		t.Fatalf("RequestToken error: %v", err)
	}
	parts := strings.Split(resp.Token, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected token format: %s", resp.Token)
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decode header: %v", err)
	}
	if !strings.Contains(string(headerJSON), "\"EdDSA\"") {
		t.Fatalf("expected alg EdDSA in header JSON: %s", string(headerJSON))
	}
	val, err := svc.ValidateToken(resp.Token)
	if err != nil {
		t.Fatalf("ValidateToken error: %v", err)
	}
	if !val.Valid || val.ClientID != "client-1" {
		t.Fatalf("unexpected validation result: %+v", val)
	}
}

func TestEdDSATokenTamper(t *testing.T) {
	os.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	svc, err := New(Config{AuthServerURL: "http://localhost", ClientID: "client-1", ClientSecret: "secret", AccessTokenExpiry: time.Minute})
	if err != nil {
		t.Fatalf("new error: %v", err)
	}
	resp, err := svc.RequestToken(TokenRequest{GrantID: "g1", Scope: []string{"read"}})
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	parts := strings.Split(resp.Token, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected token parts")
	}
	// Tamper payload
	parts[1] += "AAAA"
	bad := strings.Join(parts, ".")
	if _, err := svc.ValidateToken(bad); err == nil {
		t.Fatalf("expected validation failure for tampered token")
	}
}

func TestEdDSAUnknownKid(t *testing.T) {
	os.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	svc, err := New(Config{AuthServerURL: "http://localhost", ClientID: "client-1", ClientSecret: "secret", AccessTokenExpiry: time.Minute})
	if err != nil {
		t.Fatalf("new error: %v", err)
	}
	resp, err := svc.RequestToken(TokenRequest{GrantID: "g1", Scope: []string{"read"}})
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	parts := strings.Split(resp.Token, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected token format")
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("header decode: %v", err)
	}
	// Replace kid with arbitrary unknown value
	newHeader := strings.Replace(string(headerJSON), "\"kid\":\"", "\"kid\":\"UNKNOWN_", 1)
	newHeadEnc := base64.RawURLEncoding.EncodeToString([]byte(newHeader))
	mutated := newHeadEnc + "." + parts[1] + "." + parts[2]
	if _, err := svc.ValidateToken(mutated); err == nil {
		t.Fatalf("expected failure for unknown kid")
	}
}

func TestEdDSARotatedOldKeyStillValidUntilExpiry(t *testing.T) {
	os.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	os.Setenv("GAUTH_KEY_ROTATION_HOURS", "2")
	svc, err := New(Config{AuthServerURL: "http://localhost", ClientID: "client-1", ClientSecret: "secret", AccessTokenExpiry: time.Minute})
	if err != nil {
		t.Fatalf("new error: %v", err)
	}
	resp, err := svc.RequestToken(TokenRequest{GrantID: "g1", Scope: []string{"read"}})
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	km, ok := svc.keyProvider.(*crypto.Manager)
	if !ok {
		t.Fatalf("key provider is not a manager")
	}
	activeKid := km.Active().ID
	// Rotate key
	if _, err := km.Rotate(); err != nil {
		t.Fatalf("rotate error: %v", err)
	}
	if km.Active().ID == activeKid {
		t.Fatalf("rotation did not change active key")
	}
	// Token signed with previous key should still validate (history retention)
	if _, err := svc.ValidateToken(resp.Token); err != nil {
		t.Fatalf("expected old key token to validate: %v", err)
	}
}
