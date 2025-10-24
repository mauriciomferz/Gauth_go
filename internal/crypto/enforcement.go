package crypto

// Detached signature enforcement utilities.
// If COMPLIANCE_REQUIRE_SIGNATURE=1 is set in environment, critical operations
// must supply a non-empty signature. This is a placeholder hook; higher-level
// call sites integrate it when performing authenticity-sensitive actions.

import (
    "errors"
    "os"
)

var ErrSignatureRequired = errors.New("detached_signature_required")

func RequireDetachedSignature() bool {
    return os.Getenv("COMPLIANCE_REQUIRE_SIGNATURE") == "1"
}

// EnforceDetachedSignature returns error if signature required and missing.
func EnforceDetachedSignature(payload []byte, sig []byte) error {
    if !RequireDetachedSignature() { return nil }
    if len(sig) == 0 { return ErrSignatureRequired }
    return nil
}
