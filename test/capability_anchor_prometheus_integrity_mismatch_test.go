package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/notary"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web"
	"github.com/gin-gonic/gin"
)

// TestCapabilityAnchorPrometheusIntegrityMismatch simulates a corruption in the receipt chain file
// and verifies that a subsequent verify=1 scrape reports mismatch (gauge value 0) and that the
// verification endpoint also reports mismatch.
func TestCapabilityAnchorPrometheusIntegrityMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Create temp directory for isolated receipt store file
	dir := t.TempDir()
	storePath := filepath.Join(dir, "receipts.json")

	// Build server with receipt store injected (simulate environment wiring)
	srv := web.NewBetaServer(":0")

	// For test we create a fresh receipt store manually and assign via reflection-like minimal interface use.
	rs := notary.NewReceiptStore(storePath)
	if err := rs.Load(); err != nil {
		t.Fatalf("load empty store: %v", err)
	}

	// Append two receipts to build a small chain
	for i := 0; i < 2; i++ {
		r := notary.Receipt{Hash: randomTestHash(i), Timestamp: time.Now().UTC().Format(time.RFC3339Nano), Provider: "memory", Version: 1, Success: true, LatencySeconds: 0.001}
		if _, err := rs.Append(r); err != nil {
			t.Fatalf("append receipt %d: %v", i, err)
		}
	}

	// Corrupt the second entry's chain_hash on disk to provoke mismatch.
	// Read file, modify JSON, rewrite.
	raw, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read store: %v", err)
	}
	var jf struct {
		Entries []struct {
			Hash           string  `json:"hash"`
			Timestamp      string  `json:"timestamp"`
			Provider       string  `json:"provider"`
			Version        int     `json:"version"`
			Success        bool    `json:"success"`
			LatencySeconds float64 `json:"latency_seconds"`
			PrevHash       string  `json:"prev_hash"`
			ChainHash      string  `json:"chain_hash"`
		} `json:"entries"`
		ChainHead string `json:"chain_head"`
	}
	if err2 := json.Unmarshal(raw, &jf); err2 != nil {
		t.Fatalf("unmarshal: %v", err2)
	}
	if len(jf.Entries) < 2 {
		t.Fatalf("expected at least 2 entries, got %d", len(jf.Entries))
	}
	jf.Entries[1].ChainHash = "deadbeefcorrupted" + jf.Entries[1].ChainHash
	mod, err := json.MarshalIndent(jf, "", "  ")
	if err != nil {
		t.Fatalf("remarshal: %v", err)
	}
	if err := os.WriteFile(storePath, mod, 0o600); err != nil {
		t.Fatalf("rewrite: %v", err)
	}

	// Reload into a new store instance so in-memory entries reflect corruption.
	rs2 := notary.NewReceiptStore(storePath)
	if err := rs2.Load(); err != nil {
		t.Fatalf("reload: %v", err)
	}

	// Inject rs2 into server (uses interface with methods we need). We rely on the server having an exported field setter.
	// Since receiptStore is unexported, we simulate by setting environment variables and reassigning via reflection if needed.
	// Simpler: leverage existing field through a helper if available; else use unsafe (avoid). For test, we expose minimal hook in BetaServer if missing.
	// Fallback: use reflection.
	setReceiptStoreViaReflection(t, srv, rs2)

	// First scrape with verify=1 to force computation.
	w := web.PerformRequest(srv, "GET", "/api/v1/beta/capabilities/anchor/metrics/prometheus?verify=1")
	if w.Code != 200 {
		t.Fatalf("scrape status: %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "gauth_capability_anchor_notarization_receipts_integrity 0") {
		t.Fatalf("expected mismatch gauge value 0, body=\n%s", body)
	}

	// Call verification endpoint and expect mismatch
	w2 := web.PerformRequest(srv, "GET", "/api/v1/beta/notarization/receipts/verify")
	if w2.Code != 200 {
		t.Fatalf("verify endpoint status: %d", w2.Code)
	}
	if !strings.Contains(w2.Body.String(), "\"integrity\":\"mismatch\"") {
		t.Fatalf("expected integrity=mismatch in verify endpoint response, got %s", w2.Body.String())
	}
}

// randomTestHash returns a deterministic pseudo hash for index i.
func randomTestHash(i int) string { return strings.Repeat("a", 62) + strconv.Itoa(i) }

// setReceiptStoreViaReflection assigns the receipt store to the unexported field on BetaServer.
func setReceiptStoreViaReflection(t *testing.T, srv *web.BetaServer, rs interface{}) {
	t.Helper()
	// Use reflection to set private field receiptStore.
	// This avoids changing production code just for test injection.
	importReflectSet(t, srv, rs)
}
