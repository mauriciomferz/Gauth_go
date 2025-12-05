package crypto

import (
	"testing"
)

func TestECDSA_P256_SignVerify(t *testing.T) {
	key, err := GenerateECDSAKey("P-256")
	if err != nil {
		t.Fatalf("ECDSA P-256 keygen failed: %v", err)
	}
	msg := []byte("test message for ECDSA")
	r, s, err := SignECDSA(key, msg)
	if err != nil {
		t.Fatalf("ECDSA sign failed: %v", err)
	}
	if !VerifyECDSA(key, msg, r, s) {
		t.Fatalf("ECDSA verify failed")
	}
}

func TestECDSA_P384_SignVerify(t *testing.T) {
	key, err := GenerateECDSAKey("P-384")
	if err != nil {
		t.Fatalf("ECDSA P-384 keygen failed: %v", err)
	}
	msg := []byte("test message for ECDSA P-384")
	r, s, err := SignECDSA(key, msg)
	if err != nil {
		t.Fatalf("ECDSA sign failed: %v", err)
	}
	if !VerifyECDSA(key, msg, r, s) {
		t.Fatalf("ECDSA verify failed")
	}
}

func TestECDSA_P521_SignVerify(t *testing.T) {
	key, err := GenerateECDSAKey("P-521")
	if err != nil {
		t.Fatalf("ECDSA P-521 keygen failed: %v", err)
	}
	msg := []byte("test message for ECDSA P-521")
	r, s, err := SignECDSA(key, msg)
	if err != nil {
		t.Fatalf("ECDSA sign failed: %v", err)
	}
	if !VerifyECDSA(key, msg, r, s) {
		t.Fatalf("ECDSA verify failed")
	}
}

func TestECDSA_UnsupportedCurve(t *testing.T) {
	_, err := GenerateECDSAKey("P-999")
	if err == nil {
		t.Fatalf("Expected error for unsupported curve")
	}
}
