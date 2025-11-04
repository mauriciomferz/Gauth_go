package authorization

import (
	"errors"
	"sync"
	"time"
)

// Manager handles activation & retrieval of composite authorization artifacts.
type Manager struct {
	mu      sync.RWMutex
	current *CompositeAuthorizationState
}

var ErrConflict = errors.New("composite authorization conflict")
var ErrInvalid = errors.New("composite authorization invalid")

// Activate validates and sets a new composite authorization artifact.
// Returns state including continuity hash.
func (m *Manager) Activate(artifact *CompositeAuthorizationArtifact) (*CompositeAuthorizationState, error) {
	if artifact == nil {
		return nil, ErrInvalid
	}
	if artifact.AISystemID == "" || artifact.AuthorizationGrant == nil || artifact.PowersGranted == nil || artifact.DecisionAuthority == nil || artifact.TransactionRights == nil || artifact.ActionPermissions == nil || artifact.DualControlPrinciple == nil || artifact.AuthorizationCascade == nil {
		return nil, ErrInvalid
	}
	// Basic validity window sanity if both set
	if artifact.AuthorizationGrant != nil && artifact.AuthorizationGrant.ValidFrom != nil && artifact.AuthorizationGrant.ValidUntil != nil {
		if artifact.AuthorizationGrant.ValidUntil.Before(*artifact.AuthorizationGrant.ValidFrom) {
			return nil, ErrInvalid
		}
	}
	// Expiry must be after now
	if time.Now().After(artifact.ExpiresAt) {
		return nil, ErrInvalid
	}
	hash, err := CanonicalHash(artifact)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	prevHash := ""
	version := time.Now().UTC().Format("20060102T150405Z")
	if m.current != nil {
		// Conflict if overlapping validity windows (simple check: new valid_from < existing expires_at)
		if artifact.AuthorizationGrant != nil && artifact.AuthorizationGrant.ValidFrom != nil {
			if artifact.AuthorizationGrant.ValidFrom.Before(m.current.Artifact.ExpiresAt) {
				return nil, ErrConflict
			}
		}
		prevHash = m.current.CanonicalHash
	}
	state := &CompositeAuthorizationState{
		Artifact:             artifact,
		ActivatedAt:          time.Now().UTC(),
		Version:              version,
		PreviousArtifactHash: prevHash,
		CanonicalHash:        hash,
	}
	m.current = state
	return state, nil
}

// Current returns the active composite authorization state (nil if none).
func (m *Manager) Current() *CompositeAuthorizationState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}
