package pdp

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNewDistributedPDP(t *testing.T) {
	config := &PDPConfig{
		NodeID:              "node1",
		Address:             "localhost:8080",
		CacheTTL:            5 * time.Minute,
		CacheMaxSize:        1000,
		HealthCheckInterval: 10 * time.Second,
		ClusterSyncInterval: 30 * time.Second,
		MaxRetries:          3,
		RequestTimeout:      5 * time.Second,
	}

	pdp, err := NewDistributedPDP(config)
	if err != nil {
		t.Fatalf("Failed to create PDP: %v", err)
	}

	if pdp.nodeID != "node1" {
		t.Errorf("Expected node ID node1, got %s", pdp.nodeID)
	}

	if !pdp.isHealthy {
		t.Error("PDP should be healthy initially")
	}
}

func TestNewDistributedPDP_ValidationErrors(t *testing.T) {
	tests := []struct {
		name   string
		config *PDPConfig
	}{
		{"nil config", nil},
		{"empty node ID", &PDPConfig{Address: "localhost:8080"}},
		{"empty address", &PDPConfig{NodeID: "node1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDistributedPDP(tt.config)
			if err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

func TestMakeDecision_PolicyEvaluation(t *testing.T) {
	config := &PDPConfig{
		NodeID:       "node1",
		Address:      "localhost:8080",
		CacheTTL:     5 * time.Minute,
		CacheMaxSize: 100,
	}

	pdp, _ := NewDistributedPDP(config)
	ctx := context.Background()

	req := &DecisionRequest{
		RequestID: "req-001",
		Subject:   "user:alice",
		Action:    "read",
		Resource:  "document:123",
		Context:   map[string]interface{}{"department": "engineering"},
		Timestamp: time.Now(),
		CacheTTL:  30 * time.Second,
	}

	resp, err := pdp.MakeDecision(ctx, req)
	if err != nil {
		t.Fatalf("Failed to make decision: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	if resp.RequestID != "req-001" {
		t.Errorf("Expected request ID req-001, got %s", resp.RequestID)
	}

	if resp.Decision != DecisionPermit {
		t.Errorf("Expected permit decision, got %s", resp.Decision)
	}
}

func TestMakeDecision_CriticalResourceDenial(t *testing.T) {
	config := &PDPConfig{
		NodeID:       "node1",
		Address:      "localhost:8080",
		CacheTTL:     5 * time.Minute,
		CacheMaxSize: 100,
	}

	pdp, _ := NewDistributedPDP(config)
	ctx := context.Background()

	req := &DecisionRequest{
		RequestID: "req-002",
		Subject:   "user:bob",
		Action:    "delete",
		Resource:  "critical:database",
		Timestamp: time.Now(),
		CacheTTL:  30 * time.Second,
	}

	resp, err := pdp.MakeDecision(ctx, req)
	if err != nil {
		t.Fatalf("Failed to make decision: %v", err)
	}

	if resp.Decision != DecisionDeny {
		t.Errorf("Expected deny decision for critical resource deletion, got %s", resp.Decision)
	}
}

func TestMakeDecision_CacheHit(t *testing.T) {
	config := &PDPConfig{
		NodeID:       "node1",
		Address:      "localhost:8080",
		CacheTTL:     5 * time.Minute,
		CacheMaxSize: 100,
	}

	pdp, _ := NewDistributedPDP(config)
	ctx := context.Background()

	req := &DecisionRequest{
		RequestID: "req-003",
		Subject:   "user:charlie",
		Action:    "read",
		Resource:  "document:456",
		Timestamp: time.Now(),
		CacheTTL:  10 * time.Second,
	}

	// First request - cache miss
	resp1, _ := pdp.MakeDecision(ctx, req)

	// Second request - should be cache hit
	req.RequestID = "req-004" // Different request ID but same decision parameters
	resp2, _ := pdp.MakeDecision(ctx, req)

	// Verify both responses are for the same policy decision
	if resp1.Decision != resp2.Decision {
		t.Errorf("Cache hit should return same decision: %s vs %s", resp1.Decision, resp2.Decision)
	}

	// Cache stats should show one entry
	stats := pdp.GetCacheStats()
	size := stats["size"].(int)
	if size != 1 {
		t.Errorf("Expected cache size 1, got %d", size)
	}
}

func TestInvalidateCache(t *testing.T) {
	config := &PDPConfig{
		NodeID:       "node1",
		Address:      "localhost:8080",
		CacheTTL:     5 * time.Minute,
		CacheMaxSize: 100,
	}

	pdp, _ := NewDistributedPDP(config)
	ctx := context.Background()

	// Start background workers
	_ = pdp.Start(ctx)                //nolint:errcheck
	defer func() { _ = pdp.Stop() }() //nolint:errcheck

	// Create a cached decision
	req := &DecisionRequest{
		RequestID: "req-005",
		Subject:   "user:david",
		Action:    "write",
		Resource:  "file:test.txt",
		Timestamp: time.Now(),
		CacheTTL:  60 * time.Second,
	}

	_, _ = pdp.MakeDecision(ctx, req) //nolint:errcheck

	// Verify cache has entry
	statsBefore := pdp.GetCacheStats()
	sizeBefore := statsBefore["size"].(int)
	if sizeBefore != 1 {
		t.Fatalf("Expected cache size 1, got %d", sizeBefore)
	}

	// Invalidate cache with wildcard pattern
	err := pdp.InvalidateCache("*", "test invalidation")
	if err != nil {
		t.Fatalf("Failed to invalidate cache: %v", err)
	}

	// Give invalidation worker time to process
	time.Sleep(100 * time.Millisecond)

	// Verify cache is empty
	statsAfter := pdp.GetCacheStats()
	sizeAfter := statsAfter["size"].(int)
	if sizeAfter != 0 {
		t.Errorf("Expected cache size 0 after invalidation, got %d", sizeAfter)
	}
}

func TestAddRemoveNode(t *testing.T) {
	config := &PDPConfig{
		NodeID:       "node1",
		Address:      "localhost:8080",
		CacheTTL:     5 * time.Minute,
		CacheMaxSize: 100,
	}

	pdp, _ := NewDistributedPDP(config)

	// Add a new node
	node2 := &ClusterNode{
		ID:       "node2",
		Address:  "localhost:8081",
		Status:   NodeStatusHealthy,
		LastSeen: time.Now(),
		Load:     0.3,
	}

	err := pdp.AddNode(node2)
	if err != nil {
		t.Fatalf("Failed to add node: %v", err)
	}

	// Verify cluster has 2 nodes (self + node2)
	status := pdp.GetClusterStatus()
	if len(status) != 2 {
		t.Errorf("Expected 2 nodes in cluster, got %d", len(status))
	}

	// Remove node2
	err = pdp.RemoveNode("node2")
	if err != nil {
		t.Fatalf("Failed to remove node: %v", err)
	}

	// Verify cluster has 1 node
	status = pdp.GetClusterStatus()
	if len(status) != 1 {
		t.Errorf("Expected 1 node in cluster after removal, got %d", len(status))
	}

	// Try to remove self (should fail)
	err = pdp.RemoveNode("node1")
	if err == nil {
		t.Error("Should not be able to remove self from cluster")
	}
}

func TestCacheEviction(t *testing.T) {
	config := &PDPConfig{
		NodeID:       "node1",
		Address:      "localhost:8080",
		CacheTTL:     5 * time.Minute,
		CacheMaxSize: 3, // Small cache for testing eviction
	}

	pdp, _ := NewDistributedPDP(config)
	ctx := context.Background()

	// Add 4 decisions (exceeds cache size of 3)
	for i := 1; i <= 4; i++ {
		req := &DecisionRequest{
			RequestID: fmt.Sprintf("req-%03d", i),
			Subject:   fmt.Sprintf("user:user%d", i),
			Action:    "read",
			Resource:  fmt.Sprintf("doc:%d", i),
			Timestamp: time.Now(),
			CacheTTL:  30 * time.Second,
		}
		_, _ = pdp.MakeDecision(ctx, req) //nolint:errcheck
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Cache should be at max size (3), oldest evicted
	stats := pdp.GetCacheStats()
	size := stats["size"].(int)
	if size != 3 {
		t.Errorf("Expected cache size 3 after eviction, got %d", size)
	}
}

func TestCacheCleanup(t *testing.T) {
	config := &PDPConfig{
		NodeID:       "node1",
		Address:      "localhost:8080",
		CacheTTL:     5 * time.Minute,
		CacheMaxSize: 100,
	}

	pdp, _ := NewDistributedPDP(config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start with cleanup worker
	_ = pdp.Start(ctx)                //nolint:errcheck
	defer func() { _ = pdp.Stop() }() //nolint:errcheck

	// Add decision with short TTL
	req := &DecisionRequest{
		RequestID: "req-short-ttl",
		Subject:   "user:expire",
		Action:    "read",
		Resource:  "doc:temp",
		Timestamp: time.Now(),
		CacheTTL:  100 * time.Millisecond, // Very short TTL
	}

	_, _ = pdp.MakeDecision(ctx, req) //nolint:errcheck

	// Verify cache has entry
	statsBefore := pdp.GetCacheStats()
	sizeBefore := statsBefore["size"].(int)
	if sizeBefore != 1 {
		t.Fatalf("Expected cache size 1, got %d", sizeBefore)
	}

	// Wait for entry to expire
	time.Sleep(200 * time.Millisecond)

	// Manually trigger cleanup
	pdp.cleanupExpiredCache()

	// Verify cache is empty
	statsAfter := pdp.GetCacheStats()
	sizeAfter := statsAfter["size"].(int)
	if sizeAfter != 0 {
		t.Errorf("Expected cache size 0 after cleanup, got %d", sizeAfter)
	}
}

func TestGetClusterStatus(t *testing.T) {
	config := &PDPConfig{
		NodeID:              "node1",
		Address:             "localhost:8080",
		CacheTTL:            5 * time.Minute,
		CacheMaxSize:        100,
		HealthCheckInterval: 1 * time.Second,
	}

	pdp, _ := NewDistributedPDP(config)

	// Add multiple nodes
	for i := 2; i <= 4; i++ {
		node := &ClusterNode{
			ID:       fmt.Sprintf("node%d", i),
			Address:  fmt.Sprintf("localhost:808%d", i),
			Status:   NodeStatusHealthy,
			LastSeen: time.Now(),
			Load:     float64(i) * 0.1,
		}
		_ = pdp.AddNode(node) //nolint:errcheck
	}

	status := pdp.GetClusterStatus()
	if len(status) != 4 {
		t.Errorf("Expected 4 nodes in cluster, got %d", len(status))
	}

	// Verify each node has expected status
	for _, node := range status {
		if node.Status != NodeStatusHealthy {
			t.Errorf("Node %s should be healthy, got %s", node.ID, node.Status)
		}
	}
}

func BenchmarkMakeDecision(b *testing.B) {
	config := &PDPConfig{
		NodeID:       "node1",
		Address:      "localhost:8080",
		CacheTTL:     5 * time.Minute,
		CacheMaxSize: 10000,
	}

	pdp, _ := NewDistributedPDP(config)
	ctx := context.Background()

	req := &DecisionRequest{
		RequestID: "bench-req",
		Subject:   "user:benchuser",
		Action:    "read",
		Resource:  "doc:bench",
		Timestamp: time.Now(),
		CacheTTL:  60 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pdp.MakeDecision(ctx, req) //nolint:errcheck
	}
}

func BenchmarkMakeDecision_NoCache(b *testing.B) {
	config := &PDPConfig{
		NodeID:       "node1",
		Address:      "localhost:8080",
		CacheTTL:     5 * time.Minute,
		CacheMaxSize: 10000,
	}

	pdp, _ := NewDistributedPDP(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &DecisionRequest{
			RequestID: fmt.Sprintf("bench-req-%d", i),
			Subject:   fmt.Sprintf("user:user%d", i),
			Action:    "read",
			Resource:  fmt.Sprintf("doc:%d", i),
			Timestamp: time.Now(),
			CacheTTL:  0, // No caching
		}
		_, _ = pdp.MakeDecision(ctx, req) //nolint:errcheck
	}
}
