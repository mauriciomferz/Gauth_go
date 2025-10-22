package web

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rfc0111"
)

// TestSemanticEWMAStatePersistence ensures EWMA anomaly stats survive save/load cycles.
func TestSemanticEWMAStatePersistence(t *testing.T) {
	// Set up RFC0111 service and BetaServer wiring minimally.
	memAuthz := authz.NewMemoryAuthorizer()
	memAuthz.AddPolicy(authz.Policy{ID: "allow-all-alice", Subject: "alice", Resource: "*", Actions: []string{"*"}, Effect: authz.Allow})
	svc := rfc0111.NewService(audit.NewMemoryLogger(nil), memAuthz)
	bs := NewBetaServer("")
	bs.rfc0111Service = svc
	bs.initSemanticAnomaly()
	// Simulate semantic counter increments to produce rate samples.
	svc.SetSemanticSnapshot(map[string]uint64{"amount_limit_exceeded": 10})
	bs.semanticHistory = append(bs.semanticHistory, struct {
		At       time.Time
		Snapshot map[string]uint64
	}{At: time.Now().Add(-70 * time.Second), Snapshot: map[string]uint64{"amount_limit_exceeded": 2}})
	// Add an intermediate point within the 60s window to ensure non-zero elapsed duration for rate computation.
	bs.semanticHistory = append(bs.semanticHistory, struct {
		At       time.Time
		Snapshot map[string]uint64
	}{At: time.Now().Add(-10 * time.Second), Snapshot: map[string]uint64{"amount_limit_exceeded": 5}})
	bs.semanticHistory = append(bs.semanticHistory, struct {
		At       time.Time
		Snapshot map[string]uint64
	}{At: time.Now(), Snapshot: map[string]uint64{"amount_limit_exceeded": 10}})
	// Update anomalies from computed rates.
	r60, _ := bs.semanticRatesForWindows()
	bs.updateSemanticAnomalies(r60)
	beforeEWMA, beforeScores := bs.SemanticAnomalyStats()
	if beforeEWMA == 0 && beforeScores == 0 {
		t.Fatalf("expected non-zero EWMA entries before persistence")
	}
	// Persist to temp file.
	dir := t.TempDir()
	persistPath := filepath.Join(dir, "semantic.json")
	bs.semanticPersistPath = persistPath
	bs.saveSemanticPersistence()
	if _, err := os.Stat(persistPath); err != nil {
		t.Fatalf("expected persistence file: %v", err)
	}
	// Create new server instance and load file.
	bs2 := NewBetaServer("")
	bs2.rfc0111Service = svc
	bs2.semanticPersistPath = persistPath
	bs2.loadSemanticPersistence()
	afterEWMA, afterScores := bs2.SemanticAnomalyStats()
	if afterEWMA != beforeEWMA || afterScores != beforeScores {
		t.Fatalf("expected anomaly stats restored: before=(%d,%d) after=(%d,%d)", beforeEWMA, beforeScores, afterEWMA, afterScores)
	}
}
