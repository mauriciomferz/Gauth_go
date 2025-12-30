package notary

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
)

// Rotation verification failure reasons.
const (
	reasonDescriptorNil      = "descriptor_nil"
	reasonKidMismatchOld     = "kid_mismatch_old"
	reasonKidMismatchNew     = "kid_mismatch_new"
	reasonMissingOldSig      = "missing_old_signature"
	reasonMissingNewSig      = "missing_new_signature"
	reasonSerializationError = "serialization_error"
	reasonOldSigInvalid      = "old_sig_invalid"
	reasonNewSigInvalid      = "new_sig_invalid"
)

// canonicalRotationDescriptor builds the canonical JSON (without signatures) used for signing.
// Domain separation prefix is applied before signing to prevent cross-protocol replay.
func canonicalRotationDescriptor(rd *KeyRotationDescriptor) ([]byte, error) {
	payload := struct {
		OldKeyID         string `json:"old_key_id"`
		NewKeyID         string `json:"new_key_id"`
		EffectiveTime    string `json:"effective_time"`
		Reason           string `json:"reason"`
		PrevRotationHash string `json:"prev_rotation_hash,omitempty"`
	}{OldKeyID: rd.OldKeyID, NewKeyID: rd.NewKeyID, EffectiveTime: rd.EffectiveTime, Reason: rd.Reason, PrevRotationHash: rd.PrevRotationHash}
	enc, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	// Domain separation string
	prefixed := append([]byte("AGENTAUTH_ROTATION_DESCRIPTOR:"), enc...)
	return prefixed, nil
}

// computeKeyID derives a short key id from an Ed25519 public key (first 8 hex chars).
func computeKeyID(pub ed25519.PublicKey) string {
	h := hex.EncodeToString(pub[:8])
	return "ed25519:" + h
}

// SignRotationDescriptor attaches dual signatures using old and new private keys.
// It will set OldKeyID/NewKeyID based on provided public keys if they are empty.
func SignRotationDescriptor(oldPriv, newPriv ed25519.PrivateKey, rd *KeyRotationDescriptor) error {
	if rd == nil {
		return nil
	}
	// Safe type assertions: ed25519.PrivateKey.Public() always returns ed25519.PublicKey.
	oldPub := oldPriv.Public().(ed25519.PublicKey) //nolint:errcheck // guaranteed by ed25519.PrivateKey implementation
	newPub := newPriv.Public().(ed25519.PublicKey) //nolint:errcheck // guaranteed by ed25519.PrivateKey implementation
	if rd.OldKeyID == "" {
		rd.OldKeyID = computeKeyID(oldPub)
	}
	if rd.NewKeyID == "" {
		rd.NewKeyID = computeKeyID(newPub)
	}
	// Build canonical payload
	msg, err := canonicalRotationDescriptor(rd)
	if err != nil {
		return err
	}
	oldSig := ed25519.Sign(oldPriv, msg)
	newSig := ed25519.Sign(newPriv, msg)
	rd.OldKeySignature = base64.RawURLEncoding.EncodeToString(oldSig)
	rd.NewKeySignature = base64.RawURLEncoding.EncodeToString(newSig)
	return nil
}

// VerifyRotationDescriptor verifies both signatures; returns (valid, reason).
// Reasons: kid_mismatch_old, kid_mismatch_new, missing_old_signature, missing_new_signature,
// old_sig_invalid, new_sig_invalid.
func VerifyRotationDescriptor(rd *KeyRotationDescriptor, oldPub, newPub ed25519.PublicKey) (bool, string) {
	if rd == nil {
		return false, reasonDescriptorNil
	}
	if rd.OldKeyID != computeKeyID(oldPub) {
		return false, reasonKidMismatchOld
	}
	if rd.NewKeyID != computeKeyID(newPub) {
		return false, reasonKidMismatchNew
	}
	if rd.OldKeySignature == "" {
		return false, reasonMissingOldSig
	}
	if rd.NewKeySignature == "" {
		return false, reasonMissingNewSig
	}
	msg, err := canonicalRotationDescriptor(rd)
	if err != nil {
		return false, reasonSerializationError
	}
	oldSigBytes, err := base64.RawURLEncoding.DecodeString(rd.OldKeySignature)
	if err != nil {
		return false, reasonOldSigInvalid
	}
	newSigBytes, err := base64.RawURLEncoding.DecodeString(rd.NewKeySignature)
	if err != nil {
		return false, reasonNewSigInvalid
	}
	if !ed25519.Verify(oldPub, msg, oldSigBytes) {
		return false, reasonOldSigInvalid
	}
	if !ed25519.Verify(newPub, msg, newSigBytes) {
		return false, reasonNewSigInvalid
	}
	return true, ""
}
