package semantic

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/audit"
	"github.com/mauriciomferz/Gauth_go/pkg/authz"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth_rfc_001"
)

// TestSemanticEWMAStatePersistence ensures EWMA anomaly stats survive save/load cycles.
func TestSemanticEWMAStatePersistence(t *testing.T) {
	// Persistence file
	dir := t.TempDir()
	persistPath := filepath.Join(dir, "semantic.json")

	// Set up Handler with real RFC0111 service to provide snapshots
	memAuthz := authz.NewMemoryAuthorizer()
	memAuthz.AddPolicy(authz.Policy{ID: "allow-all-alice", Subject: "alice", Resource: "*", Actions: []string{"*"}, Effect: authz.Allow})
	svc := gauth_rfc_001.NewService(audit.NewMemoryLogger(nil), memAuthz)

	h := NewHandler(svc, nil, persistPath)

	// Simulate semantic counter increments to produce rate samples.
	svc.SetSemanticSnapshot(map[string]uint64{"amount_limit_exceeded": 10})

	// Inject history to enable rate computation
	base := time.Now().Add(-70 * time.Second)
	h.AddSnapshot(base, map[string]uint64{"amount_limit_exceeded": 2})
	h.AddSnapshot(base.Add(30*time.Second), map[string]uint64{"amount_limit_exceeded": 5})
	h.AddSnapshot(base.Add(60*time.Second), map[string]uint64{"amount_limit_exceeded": 8})
	h.AddSnapshot(time.Now(), map[string]uint64{"amount_limit_exceeded": 10})

	// Update to compute rates and EWMA
	h.Update()

	ewmaCount, scoreCount := h.Stats()
	if ewmaCount == 0 && scoreCount == 0 {
		t.Fatalf("expected non-zero stats before persistence")
	}

	// Persist
	if err := h.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	if _, err := os.Stat(persistPath); err != nil {
		t.Fatalf("expected persistence file: %v", err)
	}

	// Load into new handler
	h2 := NewHandler(svc, nil, persistPath)
	if err := h2.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	// Check if state restored (EWMA stats primarily)
	afterEWMA, afterScores := h2.Stats()
	// Scores might not be persisted or might be recomputed on next update.
	// EWMA parameters MUST be persisted.
	if afterEWMA != ewmaCount {
		t.Fatalf("expected EWMA stats restored: before=%d after=%d", ewmaCount, afterEWMA)
	}
	_ = afterScores // avoid unused
}
