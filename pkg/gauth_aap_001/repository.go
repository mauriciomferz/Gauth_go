package gauth_aap_001

import "sync"

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
	m.store[p.ID] = p
	return nil
}

func (m *memoryRepository) Get(id string) (*PowerOfAttorney, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.store[id]
	return p, ok
}

func (m *memoryRepository) ListByPrincipal(principal string) []*PowerOfAttorney {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*PowerOfAttorney, 0)
	for _, p := range m.store {
		if p.Grantor == principal || p.Grantee == principal {
			out = append(out, p)
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
	m.store[p.ID] = p
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
				result = append(result, p)
				// Recursively find children of this descendant
				findDescendants(p.ID, currentDepth+1)
			}
		}
	}

	findDescendants(parentPoaID, 0)
	return result, nil
}
