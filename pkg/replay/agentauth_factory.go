package replay

import (
	"fmt"
)

// AgentAuthReplayStore is the minimal interface needed by pkg/agentauth for replay protection.
// This avoids importing the full agentauth package here (circular dependency prevention).
type AgentAuthReplayStore interface {
	CheckAndStore(jti string) error
}

// AgentAuthReplayStoreFactory is a function signature that creates a replay store from environment.
// This matches the signature expected by agentauth.RegisterDurableReplayStoreFactory.
type AgentAuthReplayStoreFactory func(metrics interface{}) (AgentAuthReplayStore, error)

// NewAgentAuthReplayStoreFromEnv creates a DurableReplayStore configured from environment variables
// and wraps it in an adapter that implements agentauth.ReplayStore interface.
//
// This function can be registered with agentauth.RegisterDurableReplayStoreFactory() to enable
// automatic configuration via WithDurableReplayFromEnvUsingFactory().
//
// Example usage:
//
//	import (
//	    "github.com/.../pkg/agentauth"
//	    "github.com/.../pkg/replay"
//	)
//
//	func init() {
//	    agentauth.RegisterDurableReplayStoreFactory(replay.NewAgentAuthReplayStoreFromEnv)
//	}
//
// Supported environment variables:
//   - AGENTAUTH_REPLAY_WAL_PATH (default: ./data/replay.wal)
//   - AGENTAUTH_REPLAY_TTL_SEC (default: 900 = 15 minutes)
//   - AGENTAUTH_REPLAY_SNAPSHOT_INTERVAL_SEC (default: 300 = 5 minutes)
//   - AGENTAUTH_REPLAY_EVICTION_POLICY (default: ttl, options: ttl|lru|size|ttl+size)
//   - AGENTAUTH_REPLAY_EVICTION_MAX_SIZE (default: 10000)
func NewAgentAuthReplayStoreFromEnv(metrics interface{}) (AgentAuthReplayStore, error) {
	// Convert metrics to DurableReplayMetrics if possible, otherwise use noop
	var durableMetrics DurableReplayMetrics
	if m, ok := metrics.(DurableReplayMetrics); ok {
		durableMetrics = m
	} else {
		durableMetrics = NoopReplayMetrics{}
	}

	// Create DurableReplayStore from environment
	store, err := NewDurableReplayStoreFromEnv(durableMetrics)
	if err != nil {
		return nil, fmt.Errorf("create durable replay store from env: %w", err)
	}

	// Wrap in adapter that implements agentauth.ReplayStore interface
	adapter := NewDurableReplayStoreAdapter(store)

	return adapter, nil
}
