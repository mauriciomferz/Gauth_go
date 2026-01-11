package agentauth

import (
	"testing"
	"time"
)

func TestHMACLegacyIssuanceValidation(t *testing.T) {
	t.Setenv("AGENTAUTH_TOKEN_SIG_MODE", "hmac")
	svc, err := New(Config{
		AuthServerURL: "http://localhost", ClientID: "legacy-client",
		ClientSecret: "supersecretlongmaterial1234567890", AccessTokenExpiry: time.Minute,
	})
	if err != nil {
		t.Fatalf("new error: %v", err)
	}
	resp, err := svc.RequestToken(TokenRequest{GrantID: "g1", Scope: []string{"read"}})
	if err != nil {
		t.Fatalf("issue error: %v", err)
	}
	if resp.Token == "" {
		t.Fatalf("empty token")
	}
	vr, err := svc.ValidateToken(resp.Token)
	if err != nil || !vr.Valid {
		t.Fatalf("validation failed: %v %+v", err, vr)
	}
	if vr.ClientID != "legacy-client" {
		t.Fatalf("unexpected client id %s", vr.ClientID)
	}
}

func TestHMACJWTLibPath(t *testing.T) {
	t.Setenv("AGENTAUTH_TOKEN_SIG_MODE", "hmac")
	t.Setenv("AGENTAUTH_USE_JWT_LIB", "1")
	t.Setenv("AGENTAUTH_JWT_ALG", "HS256")
	svc, err := New(Config{
		AuthServerURL: "http://localhost", ClientID: "jwt-client",
		ClientSecret: "supersecretlongmaterial1234567890", AccessTokenExpiry: time.Minute,
	})
	if err != nil {
		t.Fatalf("new error: %v", err)
	}
	resp, err := svc.RequestToken(TokenRequest{GrantID: "g1", Scope: []string{"read"}})
	if err != nil {
		t.Fatalf("issue error: %v", err)
	}
	vr, err := svc.ValidateToken(resp.Token)
	if err != nil || !vr.Valid {
		t.Fatalf("validation failed: %v %+v", err, vr)
	}
	if vr.ClientID != "jwt-client" {
		t.Fatalf("unexpected client id %s", vr.ClientID)
	}
}
