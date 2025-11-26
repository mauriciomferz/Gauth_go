package auth_test

import (
	"testing"
	"time"

	auth "github.com/mauriciomferz/Gauth_go/pkg/auth"
)

func TestJWTServiceScopesRoundTrip(t *testing.T) {
	svc, err := auth.NewProperJWTService("issuer-demo", "aud-demo")
	if err != nil {
		t.Fatalf("svc err: %v", err)
	}
	scopes := []string{"read:profile", "write:data", "admin:users"}
	tok, err := svc.CreateToken("user123", scopes, time.Minute)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	claims, err := svc.ValidateToken(tok)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}

	for _, s := range scopes {
		if !claims.HasScope(s) {
			t.Fatalf("expected scope %s present; got %v", s, claims.Scopes)
		}
	}
	if claims.HasScope("nonexistent:scope") {
		t.Fatalf("unexpected scope found")
	}
}

func TestJWTServiceNoScopesBackwardCompat(t *testing.T) {
	svc, _ := auth.NewProperJWTService("issuer-demo", "aud-demo")
	tok, err := svc.CreateToken("user123", nil, time.Minute)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	claims, err := svc.ValidateToken(tok)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if len(claims.Scopes) != 0 {
		t.Fatalf("expected empty scopes slice, got %v", claims.Scopes)
	}
}
