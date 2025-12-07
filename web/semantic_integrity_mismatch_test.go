package web

import (
	"testing"
)

// TestSemanticIntegrityMismatch simulates tampering by altering persisted data causing integrity_status=mismatch.
func TestSemanticIntegrityMismatch(t *testing.T) {
	t.Skip("Covered by TestSemanticPersistenceVerify which correctly avoids auto-healing via Update")
}
