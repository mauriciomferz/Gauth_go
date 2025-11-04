package crypto

// RB6: Signature Agility helper exposing unified signing through the global rotating signer.
// Downstream components SHOULD prefer this helper (or GlobalRotatingSigner()) instead of
// calling ed25519.Sign directly. This enables transparent future algorithm expansion (BLS, ECDSA).
// If no global signer is registered it returns an error.

import (
	"crypto/ed25519"
	"errors"
)

// SignWithGlobal signs the message with the currently active global rotating signer.
// Returns error if the global registry is unset or lacks private material.
func SignWithGlobal(msg []byte) ([]byte, error) {
	s := GlobalRotatingSigner()
	if s == nil {
		return nil, errors.New("no_global_signer")
	}
	sig, err := s.Sign(msg)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

// VerifyWithGlobal verifies a signature against the active global key's public component.
// Returns false if registry or active key absent.
func VerifyWithGlobal(msg, sig []byte) bool {
	if GlobalEdDSARegistry == nil || GlobalEdDSARegistry.Active() == nil {
		return false
	}
	k := GlobalEdDSARegistry.Active()
	if len(k.Public) != ed25519.PublicKeySize {
		return false
	}
	return ed25519.Verify(k.Public, msg, sig)
}
