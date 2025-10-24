package compliance

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
	disputes map[string]DisputeMetadata
}

func NewDisputeRegistry() *DisputeRegistry {
	return &DisputeRegistry{
		disputes: make(map[string]DisputeMetadata),
	}
}

// EscalateDispute creates or updates a dispute record for a flow/action.
func (r *DisputeRegistry) EscalateDispute(meta DisputeMetadata) {
	meta.Status = "escalated"
	meta.EscalationAt = nowUnix()
	r.disputes[meta.ID] = meta
}

// ResolveDispute marks a dispute as resolved/arbitrated.
func (r *DisputeRegistry) ResolveDispute(id string, arbitrator string, notes string) bool {
	d, ok := r.disputes[id]
	if !ok {
		return false
	}
	d.Status = "resolved"
	d.ResolutionAt = nowUnix()
	d.Arbitrator = arbitrator
	d.Notes = notes
	r.disputes[id] = d
	return true
}

// GetDispute returns a dispute record by ID.
func (r *DisputeRegistry) GetDispute(id string) (DisputeMetadata, bool) {
	d, ok := r.disputes[id]
	return d, ok
}

// ListDisputes returns all dispute records.
func (r *DisputeRegistry) ListDisputes() []DisputeMetadata {
	out := make([]DisputeMetadata, 0, len(r.disputes))
	for _, d := range r.disputes {
		out = append(out, d)
	}
	return out
}

// nowUnix returns the current unix timestamp.
func nowUnix() int64 {
	return int64(1690000000) // stub for demo, replace with time.Now().Unix() in production
}
