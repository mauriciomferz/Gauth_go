package httpsig

import (
	"crypto/ed25519"
	"crypto/rand"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSigner_Ed25519(t *testing.T) {
	// Generate Ed25519 key pair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	signer, err := NewSigner("test-key", privKey)
	if err != nil {
		t.Fatalf("NewSigner failed: %v", err)
	}

	if signer.Algorithm != "ed25519" {
		t.Errorf("Expected algorithm ed25519, got %s", signer.Algorithm)
	}

	// Create request
	req := httptest.NewRequest("POST", "http://example.com/gnap/tx", strings.NewReader(`{"test": true}`))
	req.Header.Set("Content-Type", "application/json")

	// Sign
	if err := signer.Sign(req); err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	// Check headers
	sigInput := req.Header.Get("Signature-Input")
	if sigInput == "" {
		t.Error("Signature-Input header missing")
	}

	sig := req.Header.Get("Signature")
	if sig == "" {
		t.Error("Signature header missing")
	}

	// Verify signature contains expected parts
	if !strings.Contains(sigInput, "keyid=\"test-key\"") {
		t.Error("Signature-Input should contain keyid")
	}
	if !strings.Contains(sigInput, "alg=\"ed25519\"") {
		t.Error("Signature-Input should contain alg")
	}

	// Create verifier
	verifier := NewVerifier(func(keyID string) (interface{}, error) {
		if keyID == "test-key" {
			return pubKey, nil
		}
		return nil, nil
	})

	// Verify
	if err := verifier.Verify(req); err != nil {
		t.Errorf("Verify failed: %v", err)
	}
}

func TestVerifier_MissingHeaders(t *testing.T) {
	verifier := NewVerifier(func(keyID string) (interface{}, error) {
		return nil, nil
	})

	req := httptest.NewRequest("GET", "http://example.com/test", nil)

	err := verifier.Verify(req)
	if err == nil {
		t.Error("Expected error for missing headers")
	}
}

func TestVerifier_ExpiredSignature(t *testing.T) {
	// Generate key
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)

	signer, _ := NewSigner("test-key", privKey)

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	_ = signer.Sign(req)

	// Manually modify to make it expired
	sigInput := req.Header.Get("Signature-Input")
	// Replace expires with a past timestamp
	expired := strings.Replace(sigInput, "expires=", "expires=1", 1)
	req.Header.Set("Signature-Input", expired)

	verifier := NewVerifier(func(keyID string) (interface{}, error) {
		return pubKey, nil
	})

	err := verifier.Verify(req)
	if err == nil {
		t.Error("Expected error for expired signature")
	}
}

func TestParseSignatureInput(t *testing.T) {
	input := `sig1=("@method" "@target-uri" "content-type");created=1618884475;expires=1618884775;keyid="test-key-ed25519";alg="ed25519";nonce="abc123"`

	params, err := parseSignatureInput(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if params.KeyID != "test-key-ed25519" {
		t.Errorf("Expected keyid test-key-ed25519, got %s", params.KeyID)
	}

	if params.Algorithm != "ed25519" {
		t.Errorf("Expected alg ed25519, got %s", params.Algorithm)
	}

	if params.Created != 1618884475 {
		t.Errorf("Expected created 1618884475, got %d", params.Created)
	}

	if len(params.Components) != 3 {
		t.Errorf("Expected 3 components, got %d", len(params.Components))
	}
}

func TestRoundTrip(t *testing.T) {
	// Full round-trip test: sign and verify
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)

	// Sign
	signer, _ := NewSigner("roundtrip-key", privKey)
	req := httptest.NewRequest("POST", "http://as.example.com/gnap/tx", strings.NewReader(`{"access_token":{}}`))
	req.Header.Set("Content-Type", "application/json")
	_ = signer.Sign(req)

	// Verify
	verifier := NewVerifier(func(keyID string) (interface{}, error) {
		if keyID == "roundtrip-key" {
			return pubKey, nil
		}
		return nil, nil
	})

	if err := verifier.Verify(req); err != nil {
		t.Errorf("Round-trip verification failed: %v", err)
	}
}
