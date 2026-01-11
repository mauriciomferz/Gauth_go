package agentauth

import (
	"fmt"
)

// DurableReplayStoreFactory is a function type that creates a durable replay store from environment.
// This allows pkg/replay to inject its implementation without creating circular dependencies.
type DurableReplayStoreFactory func(metrics interface{}) (ReplayStore, error)

var durableReplayFactory DurableReplayStoreFactory

// RegisterDurableReplayStoreFactory allows pkg/replay to register its factory function.
// This avoids circular dependencies while enabling auto-configuration.
//
// Example usage in main.go or init():
//
//	import "github.com/.../pkg/replay"
//	agentauth.RegisterDurableReplayStoreFactory(func(m interface{}) (agentauth.ReplayStore, error) {
//	    store, err := replay.NewDurableReplayStoreFromEnv(nil)
//	    if err != nil {
//	        return nil, err
//	    }
//	    return replay.NewDurableReplayStoreAdapter(store), nil
//	})
func RegisterDurableReplayStoreFactory(factory DurableReplayStoreFactory) {
	durableReplayFactory = factory
}

// WithDurableReplayFromEnvUsingFactory creates a WithReplayStore option using the registered factory.
// Returns an error if no factory has been registered (pkg/replay not imported).
func WithDurableReplayFromEnvUsingFactory() Option {
	return func(s *Service) error {
		if durableReplayFactory == nil {
			return fmt.Errorf("durable replay factory not registered - " +
				"import pkg/replay and call agentauth.RegisterDurableReplayStoreFactory first")
		}

		// Create replay store using factory
		replayStore, err := durableReplayFactory(s.metrics)
		if err != nil {
			return fmt.Errorf("create durable replay store: %w", err)
		}

		// Inject into service
		s.replay = replayStore
		return nil
	}
}
