// # Licensing
//
// This file is part of the GAuth project and is licensed under the Apache License 2.0.
// It incorporates code and concepts from:
//   - OAuth 2.0 and OpenID Connect (Apache 2.0 License)
//   - Model Context Protocol (MIT License)
// See the LICENSE file in the project root for details.

package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/common"
)

// DelegationChainValidator provides secure delegation chain validation
type DelegationChainValidator struct {
	mu              sync.RWMutex
	maxDepth        int
	maxChainLength  int
	cycleDetection  map[string]bool
	rateLimit       map[string]time.Time
	rateLimitWindow time.Duration
}

// NewDelegationChainValidator creates a secure delegation validator
func NewDelegationChainValidator() *DelegationChainValidator {
	return &DelegationChainValidator{
		maxDepth:        5,  // Maximum delegation depth
		maxChainLength:  10, // Maximum chain length
		cycleDetection:  make(map[string]bool),
		rateLimit:       make(map[string]time.Time),
		rateLimitWindow: time.Minute,
	}
}

// ValidateDelegationChain performs comprehensive delegation chain validation
func (v *DelegationChainValidator) ValidateDelegationChain(
	ctx context.Context, chain []common.DelegationLink,
) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// 1. Check chain length limits
	if len(chain) == 0 {
		return fmt.Errorf("empty delegation chain")
	}
	if len(chain) > v.maxChainLength {
		return fmt.Errorf("delegation chain too long: %d > %d", len(chain), v.maxChainLength)
	}

	// 2. Detect cycles using graph traversal
	visited := make(map[string]bool)
	inProgress := make(map[string]bool)

	for _, link := range chain {
		if err := v.detectCycle(link.FromID, chain, visited, inProgress); err != nil {
			return fmt.Errorf("delegation cycle detected: %w", err)
		}
	}

	// 3. Validate chain connectivity and depth
	for i, link := range chain {
		// Check depth limits
		if link.Level > v.maxDepth {
			return fmt.Errorf("delegation depth exceeded: %d > %d", link.Level, v.maxDepth)
		}

		// Validate chain connectivity
		if i > 0 {
			prevLink := chain[i-1]
			if link.FromID != prevLink.ToID {
				return fmt.Errorf("broken delegation chain at level %d", i)
			}
			if link.Level != prevLink.Level+1 {
				return fmt.Errorf("invalid delegation level progression at %d", i)
			}
		}

		// Validate delegation types
		if err := v.validateDelegationType(link.Type, link.Level); err != nil {
			return fmt.Errorf("invalid delegation type at level %d: %w", link.Level, err)
		}

		// Check time constraints
		if link.Time.After(time.Now()) {
			return fmt.Errorf("future delegation not allowed at level %d", link.Level)
		}
		if time.Since(link.Time) > 365*24*time.Hour {
			return fmt.Errorf("delegation too old at level %d", link.Level)
		}
	}

	// 4. Rate limiting check
	chainHash := v.computeChainHash(chain)
	if lastSeen, exists := v.rateLimit[chainHash]; exists {
		if time.Since(lastSeen) < v.rateLimitWindow {
			return fmt.Errorf("delegation validation rate limit exceeded")
		}
	}
	v.rateLimit[chainHash] = time.Now()

	// 5. Clean up old rate limit entries
	v.cleanupRateLimit()

	return nil
}

// detectCycle uses DFS to detect cycles in delegation chain
func (v *DelegationChainValidator) detectCycle(
	nodeID string, chain []common.DelegationLink, visited, inProgress map[string]bool,
) error {
	if inProgress[nodeID] {
		return fmt.Errorf("cycle detected involving node %s", nodeID)
	}
	if visited[nodeID] {
		return nil
	}

	visited[nodeID] = true
	inProgress[nodeID] = true

	// Find all outgoing edges from this node
	for _, link := range chain {
		if link.FromID == nodeID {
			if err := v.detectCycle(link.ToID, chain, visited, inProgress); err != nil {
				return err
			}
		}
	}

	inProgress[nodeID] = false
	return nil
}

// validateDelegationType ensures delegation type is appropriate for level
func (v *DelegationChainValidator) validateDelegationType(delegationType string, level int) error {
	validTypes := map[string][]int{
		"human-to-human": {1, 2, 3},
		"human-to-ai":    {1, 2, 3, 4, 5},
		"ai-to-ai":       {2, 3, 4, 5},
	}

	allowedLevels, exists := validTypes[delegationType]
	if !exists {
		return fmt.Errorf("unknown delegation type: %s", delegationType)
	}

	for _, allowedLevel := range allowedLevels {
		if level == allowedLevel {
			return nil
		}
	}

	return fmt.Errorf("delegation type %s not allowed at level %d", delegationType, level)
}

// computeChainHash creates a simple string representation for rate limiting
// TODO: Replace with proper cryptographic hash from ProperCrypto
func (v *DelegationChainValidator) computeChainHash(chain []common.DelegationLink) string {
	result := ""
	for _, link := range chain {
		result += fmt.Sprintf("%s-%s-%s-%d-",
			link.FromID, link.ToID, link.Type, link.Level)
	}
	return result
}

// cleanupRateLimit removes old rate limit entries
func (v *DelegationChainValidator) cleanupRateLimit() {
	cutoff := time.Now().Add(-v.rateLimitWindow * 2)
	for hash, timestamp := range v.rateLimit {
		if timestamp.Before(cutoff) {
			delete(v.rateLimit, hash)
		}
	}
}

// ScopeEscalationPrevention prevents scope escalation through composition
type ScopeEscalationPrevention struct {
	mu                    sync.RWMutex
	allowedCombinations   map[string][]string
	forbiddenCombinations map[string][]string
	scopeHierarchy        map[string]int
}

// NewScopeEscalationPrevention creates scope escalation prevention system
func NewScopeEscalationPrevention() *ScopeEscalationPrevention {
	return &ScopeEscalationPrevention{
		allowedCombinations: map[string][]string{
			"read":  {"read", "audit"},
			"write": {"read", "write"},
			"admin": {"read", "write", "admin"},
		},
		forbiddenCombinations: map[string][]string{
			"read":  {"admin", "delete", "system"},
			"write": {"admin", "system"},
		},
		scopeHierarchy: map[string]int{
			"read":   1,
			"write":  2,
			"delete": 3,
			"admin":  4,
			"system": 5,
		},
	}
}

// ValidateScopeComposition ensures scope combinations don't grant unintended permissions
func (s *ScopeEscalationPrevention) ValidateScopeComposition(
	requestedScopes []string, baseScopes []string,
) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check for forbidden combinations
	for _, requested := range requestedScopes {
		forbidden, exists := s.forbiddenCombinations[requested]
		if !exists {
			continue
		}

		for _, scope := range baseScopes {
			for _, forbiddenScope := range forbidden {
				if scope == forbiddenScope {
					return fmt.Errorf("forbidden scope combination: %s with %s",
						requested, forbiddenScope)
				}
			}
		}
	}

	// Check scope hierarchy escalation
	for _, requested := range requestedScopes {
		requestedLevel, exists := s.scopeHierarchy[requested]
		if !exists {
			return fmt.Errorf("unknown scope: %s", requested)
		}

		for _, base := range baseScopes {
			baseLevel, exists := s.scopeHierarchy[base]
			if !exists {
				continue
			}

			// Prevent escalation to higher privilege level
			if requestedLevel > baseLevel+1 {
				return fmt.Errorf("scope escalation prevented: %s (%d) > %s (%d)",
					requested, requestedLevel, base, baseLevel)
			}
		}
	}

	return nil
}
