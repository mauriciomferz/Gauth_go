package delegation

// DelegationRevocation represents a revocation of a previously issued Delegation link.
// Minimal implementation for initial enforcement: we key revocations by delegation ID
// rather than hash to simplify external reference prior to hash computation in client code.
// Future iterations should move to hash-based (or signature-based) authenticity.
type DelegationRevocation struct {
	DelegationID string `json:"delegation_id"`
	Reason       string `json:"reason,omitempty"`
}

// RevocationIndex provides quick lookups for revoked delegation IDs.
type RevocationIndex struct {
	m map[string]DelegationRevocation
}

// NewRevocationIndex builds a new index from a slice of revocations.
func NewRevocationIndex(list []DelegationRevocation) *RevocationIndex {
	idx := &RevocationIndex{m: make(map[string]DelegationRevocation, len(list))}
	for _, r := range list {
		if r.DelegationID == "" {
			continue // ignore empty entries
		}
		idx.m[r.DelegationID] = r
	}
	return idx
}

// IsRevoked returns true if the provided delegation ID is revoked.
func (ri *RevocationIndex) IsRevoked(id string) bool {
	if ri == nil || ri.m == nil || id == "" {
		return false
	}
	_, ok := ri.m[id]
	return ok
}

// CheckRevocations scans the chain for any revoked delegation IDs and returns
// the first revoked ID encountered along with a boolean indicating if a
// revocation was found.
func CheckRevocations(chain *Chain, idx *RevocationIndex) (revokedID string, found bool) {
	if chain == nil || idx == nil {
		return "", false
	}
	for _, d := range chain.items { // same package: can access unexported slice
		if idx.IsRevoked(d.ID) {
			return d.ID, true
		}
	}
	return "", false
}
