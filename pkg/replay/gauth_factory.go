package replay

import (
	"fmt"
)

// GAuthReplayStore is the minimal interface needed by pkg/gauth for replay protection.
// This avoids importing the full gauth package here (circular dependency prevention).
type GAuthReplayStore interface {
	CheckAndStore(jti string) error
}

// GAuthReplayStoreFactory is a function signature that creates a replay store from environment.
// This matches the signature expected by gauth.RegisterDurableReplayStoreFactory.
type GAuthReplayStoreFactory func(metrics interface{}) (GAuthReplayStore, error)

// NewGAuthReplayStoreFromEnv creates a DurableReplayStore configured from environment variables
// and wraps it in an adapter that implements gauth.ReplayStore interface.
//
// This function can be registered with gauth.RegisterDurableReplayStoreFactory() to enable
// automatic configuration via WithDurableReplayFromEnvUsingFactory().
//
// Example usage:
//
//	import (
//	    "github.com/.../pkg/gauth"
//	    "github.com/.../pkg/replay"
//	)
//
//	func init() {
//	    gauth.RegisterDurableReplayStoreFactory(replay.NewGAuthReplayStoreFromEnv)
//	}
//
// Supported environment variables:
//   - GAUTH_REPLAY_WAL_PATH (default: ./data/replay.wal)
//   - GAUTH_REPLAY_TTL_SEC (default: 900 = 15 minutes)
//   - GAUTH_REPLAY_SNAPSHOT_INTERVAL_SEC (default: 300 = 5 minutes)
//   - GAUTH_REPLAY_EVICTION_POLICY (default: ttl, options: ttl|lru|size|ttl+size)
//   - GAUTH_REPLAY_EVICTION_MAX_SIZE (default: 10000)
func NewGAuthReplayStoreFromEnv(metrics interface{}) (GAuthReplayStore, error) {
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

	// Wrap in adapter that implements gauth.ReplayStore interface
	adapter := NewDurableReplayStoreAdapter(store)

	return adapter, nil
}
