package gauthplus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCapabilityCache_CleanupLoop(t *testing.T) {
	// 1. Create cache with short TTL
	ttl := 100 * time.Millisecond
	cache := NewCapabilityCache(ttl)
	defer cache.Close()

	// 2. Add item
	agentID := "agent1"
	assessment := &AICapabilityAssessment{AgentID: agentID, OverallLevel: "L1"}
	cache.Set(agentID, assessment)

	// Verify exists
	got, found := cache.Get(agentID)
	assert.True(t, found)
	assert.Equal(t, assessment, got)

	// 3. Wait for cleanup
	// Loop runs at ttl/2 = 50ms.
	// Wait > ttl (100ms) + buffer
	time.Sleep(200 * time.Millisecond)

	// 4. Verify gone
	// Even if not expired/evicted instantly, Get() checks expiry.
	// But we want to check if the map was cleared strictly.
	// Since we don't expose internal map directly easily without racy checks,
	// we rely on Size() if available or Get() result.

	// Actually Size() is exposed!
	assert.Equal(t, 0, cache.Size(), "Cache should be empty after cleanup loop")
}

func TestDelegationChainCache_CleanupLoop(t *testing.T) {
	ttl := 100 * time.Millisecond
	cache := NewDelegationChainCache(ttl)
	defer cache.Close()

	agentID := "agent2"
	chain := []*AIDelegation{{ID: "del1"}}
	cache.Set(agentID, chain)

	assert.Equal(t, 1, cache.Size())

	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 0, cache.Size(), "Delegation cache should be empty after cleanup loop")
}

func TestAgentAuthPlusCaches_Lifecycle(t *testing.T) {
	// Verify no panics/leaks on rapid open/close
	c1 := NewCapabilityCache(time.Minute)
	c1.Close()

	c2 := NewDelegationChainCache(time.Minute)
	c2.Close()
}
