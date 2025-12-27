package gauthplus

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCapabilityCache_Basic(t *testing.T) {
	cache := NewCapabilityCache(100 * time.Millisecond)

	// Test cache miss
	_, found := cache.Get("agent-1")
	if found {
		t.Error("expected cache miss")
	}

	// Test set and get
	assessment := &AICapabilityAssessment{
		AgentID:      "agent-1",
		OverallLevel: "L3",
	}
	cache.Set("agent-1", assessment)

	retrieved, found := cache.Get("agent-1")
	if !found {
		t.Fatal("expected cache hit")
	}
	if retrieved.AgentID != "agent-1" {
		t.Errorf("expected agent-1, got %s", retrieved.AgentID)
	}
}

func TestCapabilityCache_Expiration(t *testing.T) {
	cache := NewCapabilityCache(50 * time.Millisecond)

	assessment := &AICapabilityAssessment{
		AgentID:      "agent-2",
		OverallLevel: "L2",
	}

	cache.Set("agent-2", assessment)
	time.Sleep(100 * time.Millisecond)

	_, found := cache.Get("agent-2")
	if found {
		t.Error("expected cache entry to be expired")
	}
}

func TestCapabilityCache_Invalidate(t *testing.T) {
	cache := NewCapabilityCache(1 * time.Second)

	assessment := &AICapabilityAssessment{
		AgentID:      "agent-3",
		OverallLevel: "L4",
	}

	cache.Set("agent-3", assessment)
	cache.Invalidate("agent-3")

	_, found := cache.Get("agent-3")
	if found {
		t.Error("expected cache entry to be invalidated")
	}
}

func TestCapabilityCache_Clear(t *testing.T) {
	cache := NewCapabilityCache(1 * time.Second)

	cache.Set("agent-4", &AICapabilityAssessment{AgentID: "agent-4"})
	cache.Set("agent-5", &AICapabilityAssessment{AgentID: "agent-5"})

	if cache.Size() != 2 {
		t.Errorf("expected size 2, got %d", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("expected size 0, got %d", cache.Size())
	}
}

func TestCapabilityCache_CleanExpired(t *testing.T) {
	cache := NewCapabilityCache(50 * time.Millisecond)

	cache.Set("agent-6", &AICapabilityAssessment{AgentID: "agent-6"})
	cache.Set("agent-7", &AICapabilityAssessment{AgentID: "agent-7"})

	time.Sleep(100 * time.Millisecond)

	// Background cleanup loop may have already removed items
	cache.CleanExpired()
	// removed := cache.CleanExpired()
	// if removed != 2 {
	// 	t.Logf("background loop may have removed items already (removed manually: %d)", removed)
	// }
	if cache.Size() != 0 {
		t.Errorf("expected size 0 after cleanup, got %d", cache.Size())
	}
}

func TestDelegationChainCache_Basic(t *testing.T) {
	cache := NewDelegationChainCache(100 * time.Millisecond)

	// Test cache miss
	_, found := cache.Get("agent-1")
	if found {
		t.Error("expected cache miss")
	}

	// Test set and get
	chain := []*AIDelegation{
		{
			ID:            "del-1",
			SourceAgentID: "agent-1",
			TargetAgentID: "agent-2",
		},
	}

	cache.Set("agent-1", chain)

	retrieved, found := cache.Get("agent-1")
	if !found {
		t.Fatal("expected cache hit")
	}
	if len(retrieved) != 1 {
		t.Errorf("expected 1 delegation, got %d", len(retrieved))
	}
}

// Mock services for testing cached wrappers

type mockCapabilityService struct {
	getCallCount    int
	createCallCount int
}

func (m *mockCapabilityService) GetLatestAssessment(ctx context.Context, agentID string) (*AICapabilityAssessment, error) {
	m.getCallCount++
	if agentID == "error" {
		return nil, errors.New("mock error")
	}
	return &AICapabilityAssessment{
		AgentID:      agentID,
		OverallLevel: "L3",
	}, nil
}

func (m *mockCapabilityService) CreateAssessment(ctx context.Context, assessment *AICapabilityAssessment) error {
	m.createCallCount++
	return nil
}

func (m *mockCapabilityService) CheckCapabilityMatch(ctx context.Context, agentID string, requirements *CapabilityRequirements) (bool, []string, error) {
	return true, nil, nil
}

func (m *mockCapabilityService) GetExpiringAssessments(ctx context.Context, daysUntilExpiry int) ([]*AICapabilityAssessment, error) {
	return nil, nil
}

func TestCachedCapabilityService_CacheHit(t *testing.T) {
	ctx := context.Background()
	mock := &mockCapabilityService{}
	cached := NewCachedCapabilityService(mock, 1*time.Second)

	// First call - cache miss
	assessment1, err := cached.GetLatestAssessment(ctx, "agent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.getCallCount != 1 {
		t.Errorf("expected 1 database call, got %d", mock.getCallCount)
	}

	// Second call - cache hit
	assessment2, err := cached.GetLatestAssessment(ctx, "agent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.getCallCount != 1 {
		t.Errorf("expected 1 database call (cached), got %d", mock.getCallCount)
	}

	if assessment1.AgentID != assessment2.AgentID {
		t.Error("cached assessment differs from original")
	}
}

func TestCachedCapabilityService_CreateInvalidatesCache(t *testing.T) {
	ctx := context.Background()
	mock := &mockCapabilityService{}
	cached := NewCachedCapabilityService(mock, 1*time.Second)

	// Prime cache
	_, err := cached.GetLatestAssessment(ctx, "agent-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create new assessment (should invalidate)
	err = cached.CreateAssessment(ctx, &AICapabilityAssessment{
		AgentID:      "agent-2",
		OverallLevel: "L4",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reset counter after invalidation
	initialCount := mock.getCallCount

	// Next call should be cache miss
	_, err = cached.GetLatestAssessment(ctx, "agent-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.getCallCount != initialCount+1 {
		t.Errorf("expected database call after invalidation, got %d calls", mock.getCallCount-initialCount)
	}
}

type mockDelegationService struct {
	getCallCount    int
	createCallCount int
}

func (m *mockDelegationService) GetDelegationChain(ctx context.Context, agentID string) ([]*AIDelegation, error) {
	m.getCallCount++
	return []*AIDelegation{
		{ID: "del-1", SourceAgentID: agentID, TargetAgentID: "agent-target"},
	}, nil
}

func (m *mockDelegationService) CreateDelegation(ctx context.Context, delegation *AIDelegation) error {
	m.createCallCount++
	return nil
}

func (m *mockDelegationService) RevokeDelegation(ctx context.Context, delegationID, revokedBy, reason string) error {
	return nil
}

func (m *mockDelegationService) ValidateDelegation(ctx context.Context, sourceAgent, targetAgent string, scope []string, depth int) error {
	return nil
}

func (m *mockDelegationService) CheckMaxDepthExceeded(ctx context.Context, sourceAgentID string, currentDepth int) (bool, error) {
	return currentDepth > 5, nil
}

func TestCachedDelegationService_CacheHit(t *testing.T) {
	ctx := context.Background()
	mock := &mockDelegationService{}
	cached := NewCachedDelegationService(mock, 1*time.Second)

	// First call - cache miss
	chain1, err := cached.GetDelegationChain(ctx, "agent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.getCallCount != 1 {
		t.Errorf("expected 1 database call, got %d", mock.getCallCount)
	}

	// Second call - cache hit
	chain2, err := cached.GetDelegationChain(ctx, "agent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.getCallCount != 1 {
		t.Errorf("expected 1 database call (cached), got %d", mock.getCallCount)
	}

	if len(chain1) != len(chain2) {
		t.Error("cached chain differs from original")
	}
}

func TestCachedDelegationService_CreateInvalidatesAll(t *testing.T) {
	ctx := context.Background()
	mock := &mockDelegationService{}
	cached := NewCachedDelegationService(mock, 1*time.Second)

	// Prime cache with multiple chains
	_, _ = cached.GetDelegationChain(ctx, "agent-2")
	_, _ = cached.GetDelegationChain(ctx, "agent-3")

	if mock.getCallCount != 2 {
		t.Errorf("expected 2 database calls, got %d", mock.getCallCount)
	}

	// Create new delegation (should invalidate all)
	err := cached.CreateDelegation(ctx, &AIDelegation{
		SourceAgentID: "agent-2",
		TargetAgentID: "agent-4",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Next calls should be cache misses
	_, _ = cached.GetDelegationChain(ctx, "agent-2")
	_, _ = cached.GetDelegationChain(ctx, "agent-3")

	if mock.getCallCount != 4 {
		t.Errorf("expected 4 database calls (all caches invalidated), got %d", mock.getCallCount)
	}
}
