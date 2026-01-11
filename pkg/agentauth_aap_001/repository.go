package agentauth_aap_001

import (
	"sync"
	"time"
)

func cloneStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneIntMap(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneTimePtr(in *time.Time) *time.Time {
	if in == nil {
		return nil
	}
	v := *in
	return &v
}

func cloneSignature(in *POASignature) *POASignature {
	if in == nil {
		return nil
	}
	out := *in
	if len(in.Canonical) > 0 {
		out.Canonical = make([]byte, len(in.Canonical))
		copy(out.Canonical, in.Canonical)
	}
	return &out
}

func cloneMultiSignatures(in map[string]*POASignature) map[string]*POASignature {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]*POASignature, len(in))
	for k, v := range in {
		out[k] = cloneSignature(v)
	}
	return out
}

func clonePendingRevocation(in *PendingRevocationState) *PendingRevocationState {
	if in == nil {
		return nil
	}
	out := *in
	out.EvidenceHashes = cloneStringSlice(in.EvidenceHashes)
	if len(in.Approvals) > 0 {
		out.Approvals = make(map[string]time.Time, len(in.Approvals))
		for k, v := range in.Approvals {
			out.Approvals[k] = v
		}
	}
	return &out
}

func clonePOA(in *PowerOfAttorney) *PowerOfAttorney {
	if in == nil {
		return nil
	}
	out := *in
	out.Scope = cloneStringSlice(in.Scope)
	out.Restrictions = cloneStringMap(in.Restrictions)
	out.Witnesses = cloneStringSlice(in.Witnesses)
	out.Attestations = cloneStringSlice(in.Attestations)
	out.Signers = cloneStringSlice(in.Signers)
	out.Controllers = cloneStringSlice(in.Controllers)
	out.EvidenceHashes = cloneStringSlice(in.EvidenceHashes)
	out.RevokedScopes = cloneStringSlice(in.RevokedScopes)
	out.Weights = cloneIntMap(in.Weights)
	out.Signature = cloneSignature(in.Signature)
	out.MultiSignatures = cloneMultiSignatures(in.MultiSignatures)
	out.PendingRevocation = clonePendingRevocation(in.PendingRevocation)
	out.RevokedAt = cloneTimePtr(in.RevokedAt)
	return &out
}

// Milestone 2A: Persistence abstraction scaffold.
// NOTE: Existing code still uses the in-struct map directly; the POARepository
// interface and memoryRepository implementation are introduced now so that
// subsequent changes can migrate logic incrementally without a large diff.

// POARepository defines minimal persistence operations for PowerOfAttorney records.
// A future persistent implementation (e.g. BoltDB / Postgres) will satisfy this.
type POARepository interface {
	Create(p *PowerOfAttorney) error
	Get(id string) (*PowerOfAttorney, bool)
	ListByPrincipal(principal string) []*PowerOfAttorney
	Update(p *PowerOfAttorney) error
	// ListDescendants finds all POAs that have the given POA ID as their parent
	// maxDepth limits how deep to search (0 = unlimited depth)
	// Returns descendants organized by depth level for batch processing
	ListDescendants(parentPoaID string, maxDepth int) ([]*PowerOfAttorney, error)
}

// memoryRepository is an in-memory implementation (prototype / tests).
type memoryRepository struct {
	mu    sync.RWMutex
	store map[string]*PowerOfAttorney
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{store: make(map[string]*PowerOfAttorney)}
}

func (m *memoryRepository) Create(p *PowerOfAttorney) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[p.ID] = clonePOA(p)
	return nil
}

func (m *memoryRepository) Get(id string) (*PowerOfAttorney, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.store[id]
	if !ok {
		return nil, false
	}
	return clonePOA(p), true
}

func (m *memoryRepository) ListByPrincipal(principal string) []*PowerOfAttorney {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*PowerOfAttorney, 0)
	for _, p := range m.store {
		if p.Grantor == principal || p.Grantee == principal {
			out = append(out, clonePOA(p))
		}
	}
	return out
}

func (m *memoryRepository) Update(p *PowerOfAttorney) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.store[p.ID]; !ok {
		return nil
	}
	m.store[p.ID] = clonePOA(p)
	return nil
}

func (m *memoryRepository) ListDescendants(parentPoaID string, maxDepth int) ([]*PowerOfAttorney, error) {
	if parentPoaID == "" {
		return []*PowerOfAttorney{}, nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*PowerOfAttorney
	visited := make(map[string]bool) // Prevent cycles - shared across all recursive calls

	// Recursive depth-first search for descendants
	var findDescendants func(currentParentID string, currentDepth int)
	findDescendants = func(currentParentID string, currentDepth int) {
		if maxDepth > 0 && currentDepth >= maxDepth {
			return // Hit depth limit
		}
		if visited[currentParentID] {
			return // Cycle prevention
		}
		visited[currentParentID] = true

		for _, p := range m.store {
			if p != nil && p.ParentPOAID == currentParentID {
				// Check if we've already processed this POA to prevent cycles
				if visited[p.ID] {
					continue // Skip already visited descendants
				}
				result = append(result, clonePOA(p))
				// Recursively find children of this descendant
				findDescendants(p.ID, currentDepth+1)
			}
		}
	}

	findDescendants(parentPoaID, 0)
	return result, nil
}
