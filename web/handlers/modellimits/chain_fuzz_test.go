package modellimits

import (
	"context"
	"os"
	"testing"
)

// FuzzModelLimitAuditChain creates random sequences of audit events and then verifies the chain.
func FuzzModelLimitAuditChain(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}
		auditFile, err := os.CreateTemp(t.TempDir(), "fuzz_audit_*.jsonl")
		if err != nil {
			t.Skip()
		}
		auditFile.Close()

		// Limits file not strictly needed if we are just testing audit writing directly via internal methods
		// or if we use CheckLimit with intent to write.
		// writeAudit is private.
		// But in unit test (package modellimits) `writeAudit` IS accessible if in `_test.go` same package.
		// So we can call `writeAudit` directly.

		h := NewHandler("", auditFile.Name(), "")
		if err := h.Init(context.Background()); err != nil {
			t.Skip()
		}

		for i := 0; i+2 < len(data); i += 3 {
			kindByte := data[i] % 3
			var kind string
			switch kindByte {
			case 0:
				kind = "input"
			case 1:
				kind = "output"
			default:
				kind = "rate"
			}
			provided := int(data[i+1])
			limit := int(data[i+2]) + 1
			// writeAudit(modelID, kind, provided, limit, windowStart, windowSeconds, userID)
			h.writeAudit("fuzz-model", kind, provided, limit, 0, 0, "")
		}

		// Verify
		entries, hash, valid, err := h.VerifyAudit()
		if err != nil {
			t.Errorf("verify error: %v", err)
		}
		if !valid {
			t.Errorf("chain invalid, entries=%d hash=%s", entries, hash)
		}
	})
}
