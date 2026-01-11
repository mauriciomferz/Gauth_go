package compliance

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

// DisputeMetadata holds information for dispute escalation and arbitration tracking.
type DisputeMetadata struct {
	ID           string
	FlowID       string
	Jurisdiction Jurisdiction
	Entity       EntityType
	Reason       string
	Status       string // e.g. "pending", "escalated", "resolved", "arbitrated"
	EscalationAt int64  // timestamp
	ResolutionAt int64  // timestamp
	Arbitrator   string // arbitrator identity or reference
	Notes        string
}

// DisputeRegistry manages dispute records for flows/actions.
type DisputeRegistry struct {
	mu       sync.RWMutex
	disputes map[string]DisputeMetadata
	path     string
}

// NewDisputeRegistry creates a new registry backed by the given file path.
// If path is empty, it operates in-memory only.
func NewDisputeRegistry(path string) (*DisputeRegistry, error) {
	r := &DisputeRegistry{
		disputes: make(map[string]DisputeMetadata),
		path:     path,
	}
	if path != "" {
		if err := r.load(); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *DisputeRegistry) load() error {
	f, err := os.Open(r.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if err := json.NewDecoder(f).Decode(&r.disputes); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	return nil
}

func (r *DisputeRegistry) save() error {
	if r.path == "" {
		return nil
	}
	// #nosec G304 - Path is configured at startup
	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return json.NewEncoder(f).Encode(r.disputes)
}

// EscalateDispute creates or updates a dispute record for a flow/action.
func (r *DisputeRegistry) EscalateDispute(meta DisputeMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	meta.Status = "escalated"
	meta.EscalationAt = nowUnix()
	r.disputes[meta.ID] = meta
	return r.save()
}

// ResolveDispute marks a dispute as resolved/arbitrated.
func (r *DisputeRegistry) ResolveDispute(id string, arbitrator string, notes string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	d, ok := r.disputes[id]
	if !ok {
		// Return error or false? Original returned bool.
		// Let's return error for clearer signature with persistence
		return os.ErrNotExist // Or custom error
	}
	d.Status = "resolved"
	d.ResolutionAt = nowUnix()
	d.Arbitrator = arbitrator
	d.Notes = notes
	r.disputes[id] = d
	return r.save()
}

// GetDispute returns a dispute record by ID.
func (r *DisputeRegistry) GetDispute(id string) (DisputeMetadata, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.disputes[id]
	return d, ok
}

// ListDisputes returns all dispute records.
func (r *DisputeRegistry) ListDisputes() []DisputeMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]DisputeMetadata, 0, len(r.disputes))
	for _, d := range r.disputes {
		out = append(out, d)
	}
	return out
}

// nowUnix returns the current unix timestamp.
func nowUnix() int64 {
	return time.Now().Unix()
}
