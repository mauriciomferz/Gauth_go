package cascade

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/config"
	"github.com/mauriciomferz/AgentAuth/pkg/agentauth_aap_001"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
)

//nolint:gocyclo // Comprehensive cascade processor test
func TestCascadeProcessor(t *testing.T) {
	createTestPOA := func(id, parentID string, depth int) *agentauth_aap_001.PowerOfAttorney {
		now := time.Now().UTC()
		return &agentauth_aap_001.PowerOfAttorney{
			ID:          id,
			Version:     1,
			Grantor:     "alice",
			Grantee:     "bob",
			Scope:       []string{"read"},
			ValidFrom:   now,
			ValidUntil:  now.Add(time.Hour),
			Status:      agentauth_aap_001.POAStatusActive,
			CreatedAt:   now,
			UpdatedAt:   now,
			ParentPOAID: parentID,
			Depth:       depth,
		}
	}

	setupHierarchy := func() (*memoryRepo, []*agentauth_aap_001.PowerOfAttorney) {
		repo := &memoryRepo{store: make(map[string]*agentauth_aap_001.PowerOfAttorney)}

		// Create hierarchy: root -> child1 -> grandchild1
		//                        -> child2
		root := createTestPOA("root", "", 0)
		child1 := createTestPOA("child1", "root", 1)
		child2 := createTestPOA("child2", "root", 1)
		grandchild1 := createTestPOA("grandchild1", "child1", 2)

		poas := []*agentauth_aap_001.PowerOfAttorney{root, child1, child2, grandchild1}
		for _, poa := range poas {
			_ = repo.Create(poa)
		}

		return repo, poas
	}

	t.Run("revoke mode cascades correctly", func(t *testing.T) {
		repo, _ := setupHierarchy()
		auditor := audit.NewMemoryLogger(nil)

		cfg := config.CascadeConfig{
			Enabled:   true,
			Mode:      config.CascadeModeRevoke,
			MaxDepth:  10,
			BatchSize: 100,
		}

		processor := NewProcessor(repo, auditor, cfg, nil)
		result, err := processor.ProcessCascadeRevocation(context.Background(), "root", "admin")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ProcessedCount != 3 { // child1, child2, grandchild1
			t.Errorf("expected 3 processed, got %d", result.ProcessedCount)
		}

		if result.SuccessCount != 3 {
			t.Errorf("expected 3 successful, got %d", result.SuccessCount)
		}

		if result.FailureCount != 0 {
			t.Errorf("expected 0 failures, got %d", result.FailureCount)
		}

		if result.MaxDepthReached != 2 {
			t.Errorf("expected max depth 2, got %d", result.MaxDepthReached)
		}

		// Check that descendants are actually revoked
		child1, _ := repo.Get("child1")
		if child1.Status != agentauth_aap_001.POAStatusRevoked {
			t.Errorf("child1 should be revoked, got %s", child1.Status)
		}

		child2, _ := repo.Get("child2")
		if child2.Status != agentauth_aap_001.POAStatusRevoked {
			t.Errorf("child2 should be revoked, got %s", child2.Status)
		}

		grandchild1, _ := repo.Get("grandchild1")
		if grandchild1.Status != agentauth_aap_001.POAStatusRevoked {
			t.Errorf("grandchild1 should be revoked, got %s", grandchild1.Status)
		}

		// Root should remain unchanged
		root, _ := repo.Get("root")
		if root.Status != agentauth_aap_001.POAStatusActive {
			t.Errorf("root should remain active, got %s", root.Status)
		}
	})

	t.Run("suspend mode suspends descendants", func(t *testing.T) {
		repo, _ := setupHierarchy()
		auditor := audit.NewMemoryLogger(nil)

		cfg := config.CascadeConfig{
			Enabled:   true,
			Mode:      config.CascadeModeSuspend,
			MaxDepth:  10,
			BatchSize: 100,
		}

		processor := NewProcessor(repo, auditor, cfg, nil)
		result, err := processor.ProcessCascadeRevocation(context.Background(), "root", "admin")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.SuccessCount != 3 {
			t.Errorf("expected 3 successful, got %d", result.SuccessCount)
		}

		// Check that descendants are suspended
		child1, _ := repo.Get("child1")
		if child1.Status != agentauth_aap_001.POAStatusSuspended {
			t.Errorf("child1 should be suspended, got %s", child1.Status)
		}

		if child1.RevocationReason != "cascade_suspension:parent_revoked:depth_1" {
			t.Errorf("wrong revocation reason: %s", child1.RevocationReason)
		}
	})

	t.Run("notify mode does not change status", func(t *testing.T) {
		repo, originalPOAs := setupHierarchy()
		auditor := audit.NewMemoryLogger(nil)

		cfg := config.CascadeConfig{
			Enabled:   true,
			Mode:      config.CascadeModeNotify,
			MaxDepth:  10,
			BatchSize: 100,
		}

		processor := NewProcessor(repo, auditor, cfg, nil)
		result, err := processor.ProcessCascadeRevocation(context.Background(), "root", "admin")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.SuccessCount != 3 {
			t.Errorf("expected 3 successful, got %d", result.SuccessCount)
		}

		// Allow async audit processing to complete
		time.Sleep(50 * time.Millisecond)

		// Check that descendants are unchanged
		for _, originalPOA := range originalPOAs {
			if originalPOA.ID == "root" {
				continue // Skip root
			}
			currentPOA, _ := repo.Get(originalPOA.ID)
			if currentPOA.Status != originalPOA.Status {
				t.Errorf("POA %s status changed from %s to %s in notify mode",
					originalPOA.ID, originalPOA.Status, currentPOA.Status)
			}
		}

		// Check audit logs for notifications
		entries, err := auditor.Query(context.Background(), nil)
		if err != nil {
			t.Fatalf("failed to query audit entries: %v", err)
		}
		notificationCount := 0
		for _, entry := range entries {
			if entry.Action == "cascade_notification" {
				notificationCount++
			}
		}
		if notificationCount != 3 {
			t.Errorf("expected 3 notification entries, got %d", notificationCount)
		}
	})

	t.Run("depth limit works correctly", func(t *testing.T) {
		repo, _ := setupHierarchy()
		auditor := audit.NewMemoryLogger(nil)

		cfg := config.CascadeConfig{
			Enabled:   true,
			Mode:      config.CascadeModeRevoke,
			MaxDepth:  1, // Only process direct children
			BatchSize: 100,
		}

		processor := NewProcessor(repo, auditor, cfg, nil)
		result, err := processor.ProcessCascadeRevocation(context.Background(), "root", "admin")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ProcessedCount != 2 { // Only child1 and child2, not grandchild1
			t.Errorf("expected 2 processed with depth limit 1, got %d", result.ProcessedCount)
		}

		if result.MaxDepthReached != 1 {
			t.Errorf("expected max depth reached 1, got %d", result.MaxDepthReached)
		}

		// Check that grandchild is not revoked
		grandchild1, _ := repo.Get("grandchild1")
		if grandchild1.Status != agentauth_aap_001.POAStatusActive {
			t.Errorf("grandchild1 should remain active with depth limit, got %s", grandchild1.Status)
		}

		// But children should be revoked
		child1, _ := repo.Get("child1")
		if child1.Status != agentauth_aap_001.POAStatusRevoked {
			t.Errorf("child1 should be revoked, got %s", child1.Status)
		}
	})

	t.Run("batch processing works correctly", func(t *testing.T) {
		repo, _ := setupHierarchy()
		auditor := audit.NewMemoryLogger(nil)

		cfg := config.CascadeConfig{
			Enabled:   true,
			Mode:      config.CascadeModeRevoke,
			MaxDepth:  10,
			BatchSize: 1, // Process one at a time
		}

		processor := NewProcessor(repo, auditor, cfg, nil)
		result, err := processor.ProcessCascadeRevocation(context.Background(), "root", "admin")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have more batches due to small batch size
		if result.BatchCount < 2 {
			t.Errorf("expected multiple batches with batch size 1, got %d", result.BatchCount)
		}

		if result.SuccessCount != 3 {
			t.Errorf("expected 3 successful despite batching, got %d", result.SuccessCount)
		}
	})

	t.Run("disabled cascade returns error", func(t *testing.T) {
		repo, _ := setupHierarchy()
		auditor := audit.NewMemoryLogger(nil)

		cfg := config.CascadeConfig{
			Enabled:   false, // Disabled
			Mode:      config.CascadeModeRevoke,
			MaxDepth:  10,
			BatchSize: 100,
		}

		processor := NewProcessor(repo, auditor, cfg, nil)
		_, err := processor.ProcessCascadeRevocation(context.Background(), "root", "admin")

		if err == nil {
			t.Fatal("expected error when cascade processing disabled")
		}

		if err.Error() != "cascade processing disabled" {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("no descendants to process", func(t *testing.T) {
		repo := &memoryRepo{store: make(map[string]*agentauth_aap_001.PowerOfAttorney)}

		// Only create root with no children
		root := createTestPOA("root", "", 0)
		_ = repo.Create(root)

		auditor := audit.NewMemoryLogger(nil)
		cfg := config.CascadeConfig{
			Enabled:   true,
			Mode:      config.CascadeModeRevoke,
			MaxDepth:  10,
			BatchSize: 100,
		}

		processor := NewProcessor(repo, auditor, cfg, nil)
		result, err := processor.ProcessCascadeRevocation(context.Background(), "root", "admin")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ProcessedCount != 0 {
			t.Errorf("expected 0 processed with no descendants, got %d", result.ProcessedCount)
		}

		if result.SuccessCount != 0 {
			t.Errorf("expected 0 successful with no descendants, got %d", result.SuccessCount)
		}
	})

	t.Run("context cancellation stops processing", func(t *testing.T) {
		repo, _ := setupHierarchy()
		auditor := audit.NewMemoryLogger(nil)

		cfg := config.CascadeConfig{
			Enabled:   true,
			Mode:      config.CascadeModeRevoke,
			MaxDepth:  10,
			BatchSize: 1, // Small batches to increase chance of cancellation
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		processor := NewProcessor(repo, auditor, cfg, nil)
		result, err := processor.ProcessCascadeRevocation(ctx, "root", "admin")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have some errors due to context cancellation
		if len(result.Errors) == 0 {
			t.Error("expected errors due to context cancellation")
		}
	})
}

// memoryRepo is a test implementation of POARepository
type memoryRepo struct {
	store map[string]*agentauth_aap_001.PowerOfAttorney
}

func (m *memoryRepo) Create(p *agentauth_aap_001.PowerOfAttorney) error {
	m.store[p.ID] = p
	return nil
}

func (m *memoryRepo) Get(id string) (*agentauth_aap_001.PowerOfAttorney, bool) {
	p, ok := m.store[id]
	return p, ok
}

func (m *memoryRepo) ListByPrincipal(principal string) []*agentauth_aap_001.PowerOfAttorney {
	var result []*agentauth_aap_001.PowerOfAttorney
	for _, p := range m.store {
		if p.Grantor == principal || p.Grantee == principal {
			result = append(result, p)
		}
	}
	return result
}

func (m *memoryRepo) Update(p *agentauth_aap_001.PowerOfAttorney) error {
	if _, ok := m.store[p.ID]; !ok {
		return nil // Not found
	}
	m.store[p.ID] = p
	return nil
}

func (m *memoryRepo) ListDescendants(parentPoaID string, maxDepth int) ([]*agentauth_aap_001.PowerOfAttorney, error) {
	if parentPoaID == "" {
		return []*agentauth_aap_001.PowerOfAttorney{}, nil
	}

	var result []*agentauth_aap_001.PowerOfAttorney
	visited := make(map[string]bool)

	var findDescendants func(currentParentID string, currentDepth int)
	findDescendants = func(currentParentID string, currentDepth int) {
		if maxDepth > 0 && currentDepth >= maxDepth {
			return
		}
		if visited[currentParentID] {
			return
		}
		visited[currentParentID] = true

		for _, p := range m.store {
			if p != nil && p.ParentPOAID == currentParentID {
				if visited[p.ID] {
					continue
				}
				result = append(result, p)
				findDescendants(p.ID, currentDepth+1)
			}
		}
	}

	findDescendants(parentPoaID, 0)
	return result, nil
}
