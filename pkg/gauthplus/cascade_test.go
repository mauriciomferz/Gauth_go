package gauthplus

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mauriciomferz/AgentAuth/pkg/database"
)

func TestCascadeRevocationBenchmark(t *testing.T) {
	// This test requires a running database.
	// If not available, skip it.
	db, err := database.NewDB(&database.Config{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "password",
		Database: "gauth",
		SSLMode:  "disable",
	})
	if err != nil {
		t.Skip("Database not available for benchmark")
	}
	defer db.Close()

	ctx := context.Background()
	poaStore := NewPostgreSQLPoAStore(db)
	delSvc := NewPostgreSQLDelegationService(db)

	// cleanup before test
	_, _ = db.Pool.Exec(ctx, "DELETE FROM ai_delegations")
	_, _ = db.Pool.Exec(ctx, "DELETE FROM poa_records")

	// Create a root PoA
	poaID := uuid.New().String()
	agent1 := "agent-root"
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO poa_records (id, grantor_id, representative_id, status, actions, valid_from, valid_until)
		VALUES ($1, $2, $3, 'active', $4, $5, $6)
	`, poaID, "human-ceo", agent1, []string{"all"}, time.Now(), time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Failed to create root PoA: %v", err)
	}

	// Create a deep tree
	// Depth 5, Fan-out 3
	// Total nodes: 1 (root) + 3 + 9 + 27 + 81 + 243 = 364 delegations
	t.Log("Building deep delegation tree...")
	start := time.Now()
	createTree(ctx, delSvc, agent1, poaID, 1, 5, 3)
	t.Logf("Tree created in %v", time.Since(start))

	// Verify count
	var count int
	_ = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM ai_delegations").Scan(&count)
	t.Logf("Total delegations created: %d", count)

	// Measure revocation speed
	t.Log("Revoking root PoA...")
	start = time.Now()
	err = poaStore.RevokePoA(ctx, poaID, "human-ceo", "cleanup")
	if err != nil {
		t.Fatalf("Revocation failed: %v", err)
	}

	// Since it's async, wait a bit or poll
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		var revokedCount int
		_ = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM ai_delegations WHERE status = 'revoked'").Scan(&revokedCount)
		t.Logf("Internal status: %d / %d revoked", revokedCount, count)
		if revokedCount == count {
			t.Logf("Cascade revocation complete in %v", time.Since(start))
			return
		}
	}

	t.Errorf("Cascade revocation timed out")
}

func createTree(ctx context.Context, svc *PostgreSQLDelegationService, parentAgentID string, sourcePoAID string, depth int, maxDepth int, fanOut int) {
	if depth > maxDepth {
		return
	}

	for i := 0; i < fanOut; i++ {
		agentID := fmt.Sprintf("agent-d%d-f%d-%s", depth, i, uuid.New().String()[:8])
		del := &AIDelegation{
			ID:              uuid.New().String(),
			SourcePOAID:     sourcePoAID,
			SourceAgentID:   parentAgentID,
			TargetAgentID:   agentID,
			DelegatedScope:  []string{"read"},
			DelegationDepth: depth,
			MaxAllowedDepth: maxDepth,
			ValidFrom:       time.Now(),
			ValidUntil:      time.Now().Add(1 * time.Hour),
			Status:          "active",
		}
		_ = svc.CreateDelegation(ctx, del)
		createTree(ctx, svc, agentID, sourcePoAID, depth+1, maxDepth, fanOut)
	}
}
