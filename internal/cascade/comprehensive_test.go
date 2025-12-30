package cascade

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/config"
	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/gauth_aap_001"
)

// TestCascadeProcessorComprehensive adds comprehensive test scenarios for cascade revocation
//
//nolint:gocyclo // End-to-end cascade testing
func TestCascadeProcessorComprehensive(t *testing.T) {
	// Helper to create a test POA with realistic attributes
	createRealisticPOA := func(id, parentID, grantor, grantee string, scope []string, depth int) *gauth_aap_001.PowerOfAttorney {
		now := time.Now().UTC()
		return &gauth_aap_001.PowerOfAttorney{
			ID:          id,
			Version:     1,
			Grantor:     grantor,
			Grantee:     grantee,
			Scope:       scope,
			ValidFrom:   now,
			ValidUntil:  now.Add(24 * time.Hour),
			Status:      gauth_aap_001.POAStatusActive,
			CreatedAt:   now,
			UpdatedAt:   now,
			ParentPOAID: parentID,
			Depth:       depth,
		}
	}

	// Setup a complex delegation tree for realistic testing
	setupComplexHierarchy := func() (*memoryRepo, map[string]*gauth_aap_001.PowerOfAttorney) {
		repo := &memoryRepo{store: make(map[string]*gauth_aap_001.PowerOfAttorney)}
		poas := make(map[string]*gauth_aap_001.PowerOfAttorney)

		// Create a complex tree:
		// ceo -> [cto, cfo]
		// cto -> [eng-lead1, eng-lead2]
		// cfo -> [finance-lead]
		// eng-lead1 -> [dev1, dev2]
		// eng-lead2 -> [dev3]
		// finance-lead -> [analyst1, analyst2]

		ceo := createRealisticPOA("ceo", "", "board", "ceo", []string{"*"}, 0)
		cto := createRealisticPOA("cto", "ceo", "ceo", "cto", []string{"engineering.*"}, 1)
		cfo := createRealisticPOA("cfo", "ceo", "ceo", "cfo", []string{"finance.*"}, 1)

		engLead1 := createRealisticPOA("eng-lead1", "cto", "cto", "eng-lead1", []string{"engineering.backend.*"}, 2)
		engLead2 := createRealisticPOA("eng-lead2", "cto", "cto", "eng-lead2", []string{"engineering.frontend.*"}, 2)
		financeLead := createRealisticPOA("finance-lead", "cfo", "cfo", "finance-lead", []string{"finance.reporting.*"}, 2)

		dev1 := createRealisticPOA("dev1", "eng-lead1", "eng-lead1", "dev1", []string{"engineering.backend.api"}, 3)
		dev2 := createRealisticPOA("dev2", "eng-lead1", "eng-lead1", "dev2", []string{"engineering.backend.db"}, 3)
		dev3 := createRealisticPOA("dev3", "eng-lead2", "eng-lead2", "dev3", []string{"engineering.frontend.ui"}, 3)

		analyst1 := createRealisticPOA("analyst1", "finance-lead", "finance-lead", "analyst1", []string{"finance.reporting.monthly"}, 3)
		analyst2 := createRealisticPOA("analyst2", "finance-lead", "finance-lead", "analyst2", []string{"finance.reporting.quarterly"}, 3)

		allPOAs := []*gauth_aap_001.PowerOfAttorney{
			ceo, cto, cfo, engLead1, engLead2, financeLead,
			dev1, dev2, dev3, analyst1, analyst2,
		}

		for _, poa := range allPOAs {
			_ = repo.Create(poa)
			poas[poa.ID] = poa
		}

		return repo, poas
	}

	t.Run("complex delegation tree cascade", func(t *testing.T) {
		repo, _ := setupComplexHierarchy()
		auditor := audit.NewMemoryLogger(nil)
		cfg := config.CascadeConfig{
			Enabled:   true,
			Mode:      config.CascadeModeRevoke,
			BatchSize: 3,
			MaxDepth:  10,
		}
		processor := NewProcessor(repo, auditor, cfg, metrics.Noop)

		// Revoke the CTO - should cascade to all engineering POAs
		ctx := context.Background()
		result, err := processor.ProcessCascadeRevocation(ctx, "cto", "admin")

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should process: eng-lead1, eng-lead2, dev1, dev2, dev3 (5 descendants)
		if result.ProcessedCount != 5 {
			t.Errorf("Expected 5 processed POAs, got %d", result.ProcessedCount)
		}

		if result.SuccessCount != 5 {
			t.Errorf("Expected 5 successful revocations, got %d", result.SuccessCount)
		}

		// Verify all engineering POAs are revoked
		engineeringPOAs := []string{"eng-lead1", "eng-lead2", "dev1", "dev2", "dev3"}
		for _, id := range engineeringPOAs {
			poa, exists := repo.Get(id)
			if !exists {
				t.Errorf("POA %s should exist", id)
				continue
			}
			if poa.Status != gauth_aap_001.POAStatusRevoked {
				t.Errorf("POA %s should be revoked, got status: %s", id, poa.Status)
			}
		}

		// Verify finance POAs are NOT affected
		financePOAs := []string{"cfo", "finance-lead", "analyst1", "analyst2"}
		for _, id := range financePOAs {
			poa, exists := repo.Get(id)
			if !exists {
				t.Errorf("POA %s should exist", id)
				continue
			}
			if poa.Status != gauth_aap_001.POAStatusActive {
				t.Errorf("POA %s should remain active, got status: %s", id, poa.Status)
			}
		}

		// Verify batch processing occurred
		if result.BatchCount < 2 {
			t.Errorf("Expected at least 2 batches for 5 POAs with batch size 3, got %d", result.BatchCount)
		}

		// Verify max depth reached
		if result.MaxDepthReached != 3 {
			t.Errorf("Expected max depth 3, got %d", result.MaxDepthReached)
		}

		// Allow async audit processing to complete
		time.Sleep(50 * time.Millisecond)

		// Check audit logs have been created
		events, err := auditor.Query(context.Background(), nil)
		if err != nil {
			t.Errorf("Failed to query audit events: %v", err)
		}
		if len(events) < 7 { // initiation + 5 revocations + completion
			t.Errorf("Expected at least 7 audit events, got %d", len(events))
		}
	})

	t.Run("suspend mode preserves hierarchy structure", func(t *testing.T) {
		repo, _ := setupComplexHierarchy()
		auditor := audit.NewMemoryLogger(nil)
		cfg := config.CascadeConfig{
			Enabled:   true,
			Mode:      config.CascadeModeSuspend,
			BatchSize: 5,
			MaxDepth:  10,
		}
		processor := NewProcessor(repo, auditor, cfg, metrics.Noop)

		ctx := context.Background()
		result, err := processor.ProcessCascadeRevocation(ctx, "cfo", "admin")

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should process: finance-lead, analyst1, analyst2 (3 descendants)
		if result.ProcessedCount != 3 {
			t.Errorf("Expected 3 processed POAs, got %d", result.ProcessedCount)
		}

		// Verify all finance POAs are suspended (not revoked)
		financePOAs := []string{"finance-lead", "analyst1", "analyst2"}
		for _, id := range financePOAs {
			poa, exists := repo.Get(id)
			if !exists {
				t.Errorf("POA %s should exist", id)
				continue
			}
			if poa.Status != gauth_aap_001.POAStatusSuspended {
				t.Errorf("POA %s should be suspended, got status: %s", id, poa.Status)
			}
		}
	})

	t.Run("performance test - deep hierarchy", func(t *testing.T) {
		// Create a deep chain: root -> level1 -> level2 -> ... -> level20
		repo := &memoryRepo{store: make(map[string]*gauth_aap_001.PowerOfAttorney)}

		const maxLevels = 20
		var previousID = ""

		for i := 0; i < maxLevels; i++ {
			id := fmt.Sprintf("level-%d", i)
			poa := createRealisticPOA(
				id,
				previousID,
				fmt.Sprintf("user-%d", i),
				fmt.Sprintf("user-%d", i+1),
				[]string{fmt.Sprintf("scope.level.%d", i)},
				i,
			)
			_ = repo.Create(poa)
			previousID = id
		}

		auditor := audit.NewMemoryLogger(nil)
		cfg := config.CascadeConfig{
			Enabled:   true,
			Mode:      config.CascadeModeRevoke,
			BatchSize: 5,
			MaxDepth:  25, // Allow processing the full chain
		}
		processor := NewProcessor(repo, auditor, cfg, metrics.Noop)

		start := time.Now()
		ctx := context.Background()
		result, err := processor.ProcessCascadeRevocation(ctx, "level-0", "admin")
		processingTime := time.Since(start)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should process all 19 descendants (level-1 through level-19)
		if result.ProcessedCount != maxLevels-1 {
			t.Errorf("Expected %d processed POAs, got %d", maxLevels-1, result.ProcessedCount)
		}

		// Performance check - should complete within reasonable time
		if processingTime > 5*time.Second {
			t.Errorf("Processing took too long: %v", processingTime)
		}

		t.Logf("Processed %d POAs in %v", result.ProcessedCount, processingTime)

		// Verify max depth tracking
		if result.MaxDepthReached != maxLevels-1 {
			t.Errorf("Expected max depth %d, got %d", maxLevels-1, result.MaxDepthReached)
		}
	})

	t.Run("error injection - repository failures", func(t *testing.T) {
		// Create a failing repository that simulates partial failures
		failingRepo := &failingMemoryRepo{
			memoryRepo:  &memoryRepo{store: make(map[string]*gauth_aap_001.PowerOfAttorney)},
			failureRate: 0.3, // 30% failure rate
			failAfterID: "failure-point",
		}

		// Set up hierarchy
		root := createRealisticPOA("root", "", "alice", "bob", []string{"*"}, 0)
		child1 := createRealisticPOA("child1", "root", "bob", "charlie", []string{"read.*"}, 1)
		failurePoint := createRealisticPOA("failure-point", "root", "bob", "dave", []string{"write.*"}, 1)
		child3 := createRealisticPOA("child3", "root", "bob", "eve", []string{"admin.*"}, 1)

		for _, poa := range []*gauth_aap_001.PowerOfAttorney{root, child1, failurePoint, child3} {
			_ = failingRepo.Create(poa)
		}

		auditor := audit.NewMemoryLogger(nil)
		cfg := config.CascadeConfig{
			Enabled:   true,
			Mode:      config.CascadeModeRevoke,
			BatchSize: 2,
			MaxDepth:  10,
		}
		processor := NewProcessor(failingRepo, auditor, cfg, metrics.Noop)

		ctx := context.Background()
		result, err := processor.ProcessCascadeRevocation(ctx, "root", "admin")

		// Should still complete but with errors
		if err != nil {
			t.Fatalf("Expected no error from processor, got: %v", err)
		}

		// Should have some failures
		if result.FailureCount == 0 {
			t.Error("Expected some errors due to repository failures")
		}

		// Total processed + errors should equal total descendants
		totalAttempted := result.SuccessCount + result.FailureCount
		if totalAttempted != 3 { // child1, failure-point, child3
			t.Errorf("Expected 3 total attempts, got %d", totalAttempted)
		}

		t.Logf("Error injection test: %d successes, %d errors", result.SuccessCount, result.FailureCount)
	})

	t.Run("concurrent cascade operations", func(t *testing.T) {
		repo, _ := setupComplexHierarchy()
		auditor := audit.NewMemoryLogger(nil)
		cfg := config.CascadeConfig{
			Enabled:   true,
			Mode:      config.CascadeModeNotify,
			BatchSize: 2,
			MaxDepth:  10,
		}
		processor := NewProcessor(repo, auditor, cfg, metrics.Noop)

		// Start multiple cascade operations concurrently
		ctx := context.Background()
		results := make(chan *ProcessorResult, 2)
		errors := make(chan error, 2)

		// Concurrent operation 1: cascade from CTO
		go func() {
			result, err := processor.ProcessCascadeRevocation(ctx, "cto", "admin1")
			results <- result
			errors <- err
		}()

		// Concurrent operation 2: cascade from CFO
		go func() {
			result, err := processor.ProcessCascadeRevocation(ctx, "cfo", "admin2")
			results <- result
			errors <- err
		}()

		// Collect results
		for i := 0; i < 2; i++ {
			err := <-errors
			if err != nil {
				t.Errorf("Concurrent operation %d failed: %v", i+1, err)
			}
			result := <-results
			if result.ProcessedCount == 0 {
				t.Errorf("Concurrent operation %d processed no POAs", i+1)
			}
		}

		// Allow async audit processing to complete
		time.Sleep(50 * time.Millisecond)

		// Verify audit logs contain entries from both operations
		events, err := auditor.Query(context.Background(), nil)
		if err != nil {
			t.Errorf("Failed to query audit events: %v", err)
		}
		if len(events) < 4 { // At least 2 initiations + 2 completions
			t.Errorf("Expected at least 4 audit events from concurrent operations, got %d", len(events))
		}
	})

	t.Run("cascade with metrics integration", func(t *testing.T) {
		repo, _ := setupComplexHierarchy()
		auditor := audit.NewMemoryLogger(nil)
		memoryMetrics := metrics.NewMemory()
		cfg := config.CascadeConfig{
			Enabled:   true,
			Mode:      config.CascadeModeRevoke,
			BatchSize: 3,
			MaxDepth:  10,
		}
		processor := NewProcessor(repo, auditor, cfg, memoryMetrics)

		ctx := context.Background()
		result, err := processor.ProcessCascadeRevocation(ctx, "cto", "admin")

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Verify metrics were recorded
		snapshot := memoryMetrics.SnapshotEx()

		// Check cascade trigger metric
		if snapshot.CascadeRevocationTriggered != 1 {
			t.Errorf("Expected cascade trigger metric to be 1, got %d", snapshot.CascadeRevocationTriggered)
		}

		// Check descendants processed metric
		// #nosec G115: test code, ProcessedCount is small value
		if snapshot.CascadeDescendantsProcessed != uint64(result.ProcessedCount) {
			t.Errorf("Expected descendants processed metric to be %d, got %d", result.ProcessedCount, snapshot.CascadeDescendantsProcessed)
		} // Check batch processing metric
		// #nosec G115: test code, BatchCount is small value
		if snapshot.CascadeBatchProcessed != uint64(result.BatchCount) {
			t.Errorf("Expected batches processed metric to be %d, got %d", result.BatchCount, snapshot.CascadeBatchProcessed)
		}
	})
}

// failingMemoryRepo simulates repository failures for error injection testing
type failingMemoryRepo struct {
	*memoryRepo
	failureRate float64
	failAfterID string
	updateCalls int
}

func (f *failingMemoryRepo) Update(p *gauth_aap_001.PowerOfAttorney) error {
	f.updateCalls++

	// Fail updates after encountering the specific ID
	if p.ID == f.failAfterID {
		return fmt.Errorf("simulated repository failure for ID: %s", p.ID)
	}

	// Random failures based on failure rate
	if f.updateCalls > 2 && float64(f.updateCalls%10) < f.failureRate*10 {
		return fmt.Errorf("simulated random repository failure")
	}

	return f.memoryRepo.Update(p)
}
