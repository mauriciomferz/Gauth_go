package sync

import (
	"context"
	"fmt"
	"log"
	"time"
)

// PolicySyncer coordinates synchronization between a source and an authorizer.
type PolicySyncer struct {
	source      PolicySource
	authorizer  PolicyAuthorizer
	interval    time.Duration
	lastVersion string
}

// NewPolicySyncer creates a new syncer.
func NewPolicySyncer(source PolicySource, authorizer PolicyAuthorizer, interval time.Duration) *PolicySyncer {
	return &PolicySyncer{
		source:     source,
		authorizer: authorizer,
		interval:   interval,
	}
}

// Start begins the polling loop. Blocking.
func (s *PolicySyncer) Start(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Initial fetch
	if err := s.sync(ctx); err != nil {
		log.Printf("[PolicySyncer] Initial fetch failed: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.sync(ctx); err != nil {
				log.Printf("[PolicySyncer] Sync failed: %v", err)
			}
		}
	}
}

// sync performs a single synchronization step.
func (s *PolicySyncer) sync(ctx context.Context) error {
	policies, version, err := s.source.Fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}

	if version == s.lastVersion {
		return nil // No change
	}

	log.Printf("[PolicySyncer] Detected updates (version %s -> %s), replacing policies...", s.lastVersion, version)

	if err := s.authorizer.ReplacePolicies(ctx, policies); err != nil {
		return fmt.Errorf("replace policies failed: %w", err)
	}

	s.lastVersion = version
	log.Printf("[PolicySyncer] Policies updated successfully.")
	return nil
}
