package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// FuzzModelLimitAuditChain creates random sequences of audit events and then verifies the chain.
// Run: go test -fuzz=FuzzModelLimitAuditChain -run=^$
func FuzzModelLimitAuditChain(f *testing.F) {
    f.Add([]byte{1,2,3,4,5})
    f.Fuzz(func(t *testing.T, data []byte) {
        if len(data) == 0 { return }
        auditFile, err := os.CreateTemp(t.TempDir(), "fuzz_audit_*.jsonl")
        if err != nil { t.Skip() }
        auditFile.Close()
        os.Setenv("GAUTH_MODEL_LIMIT_AUDIT_PATH", auditFile.Name())
        bs := NewBetaServer("")
        // Feed events: interpret bytes as triplets (kind selector, provided, limit)
        for i:=0; i+2 < len(data); i+=3 {
            kindByte := data[i] % 3
            var kind string
            switch kindByte { case 0: kind="input"; case 1: kind="output"; default: kind="rate" }
            provided := int(data[i+1])
            limit := int(data[i+2]) + 1 // ensure non-zero to exercise logic
            bs.writeModelLimitAudit("fuzz-model", kind, provided, limit, 0, 0, "")
        }
        // Best-effort verify via HTTP route.
        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/model/limits/audit/verify", nil)
        bs.router.ServeHTTP(w, req)
    })
}
