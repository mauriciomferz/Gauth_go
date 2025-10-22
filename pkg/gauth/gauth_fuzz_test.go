package gauth

import (
	"strings"
	"testing"
	"time"
)

// FuzzValidateToken exercises ValidateToken with malformed / random token structures
// ensuring it never panics. Focuses on legacy HMAC path for deterministic behavior.
func FuzzValidateToken(f *testing.F) {
	svc, err := New(Config{ClientID: "fuzz-client", ClientSecret: strings.Repeat("y", 40), AuthServerURL: "https://auth.local", AccessTokenExpiry: time.Hour})
	if err != nil {
		f.Fatalf("service init: %v", err)
	}
	// Seed corpus with a few well-formed tokens
	baseClaims := map[string]any{"sub": "fuzz-client", "scope": "read", "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix()}
	goodTok := buildManualToken(svc.signingKey, baseClaims)
	f.Add(goodTok)
	f.Add("")
	f.Add("abc.def")           // wrong segment count
	f.Add("abc.def.")          // empty signature
	f.Add("abc.def.ghi.extra") // too many segments
	f.Add("..")
	f.Add(strings.Repeat(".", 10))
	f.Add("not base64!.not base64!.not base64!")

	f.Fuzz(func(t *testing.T, token string) {
		// Attempt validation; only requirement is no panic.
		_, _ = svc.ValidateToken(token)
	})
}
