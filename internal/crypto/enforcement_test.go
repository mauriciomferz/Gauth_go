package crypto

import "testing"

func TestDetachedSignatureEnforcement(t *testing.T) {
    t.Setenv("COMPLIANCE_REQUIRE_SIGNATURE", "1")
    if err := EnforceDetachedSignature([]byte("payload"), nil); err == nil {
        t.Fatalf("expected error when signature required and nil")
    }
    if err := EnforceDetachedSignature([]byte("payload"), []byte("sig")); err != nil {
        t.Fatalf("unexpected error with signature present: %v", err)
    }
}
