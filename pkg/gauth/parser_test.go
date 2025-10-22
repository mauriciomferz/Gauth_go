package gauth

import "testing"

func TestParseClaimsBasic(t *testing.T) {
	js := []byte(`{"sub":"alice","exp":123456,"scope":["read"],"meta":{"tier":"gold"}}`)
	claims, err := ParseClaims(js)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if claims["sub"] != "alice" {
		t.Fatalf("sub mismatch")
	}
	if m, ok := claims["meta"].(map[string]any); !ok || m["tier"] != "gold" {
		t.Fatalf("meta tier mismatch")
	}
}

func TestParseClaimsDuplicate(t *testing.T) {
	js := []byte(`{"a":1,"a":2}`)
	_, err := ParseClaims(js)
	if err == nil {
		t.Fatalf("expected duplicate key error")
	}
}

func TestParseClaimsDepthExceeded(t *testing.T) {
	// create depth >10: 11 nested objects
	d := "{\"x\":"
	for i := 0; i < 11; i++ {
		d += "{"
	}
	d += "1"
	for i := 0; i < 11; i++ {
		d += "}"
	}
	d += "}"
	_, err := ParseClaims([]byte(d))
	if err == nil {
		t.Fatalf("expected depth error")
	}
}
