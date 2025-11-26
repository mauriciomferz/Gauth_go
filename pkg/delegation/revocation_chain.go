package delegation

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/crypto"
)

// RevocationEvent represents a revocation of a previously issued Delegation.
// It can reference either the delegation ID or its hash (both optional but at least one required).
// Hash chaining (PrevHash -> Hash) permits tamper evident sequencing for auditability.
type RevocationEvent struct {
	ID             string    `json:"id"`               // unique revocation event id
	DelegationID   string    `json:"delegation_id"`    // original delegation ID (optional if hash provided)
	DelegationHash string    `json:"delegation_hash"`  // hash of delegation (optional if id provided)
	Reason         string    `json:"reason,omitempty"` // constrained to RevocationReason values
	RevokedAt      time.Time `json:"revoked_at"`
	PrevHash       string    `json:"prev_hash"`
	Hash           string    `json:"hash"`
	SigKid         string    `json:"sig_kid,omitempty"`   // signing key id (if signatures enabled)
	Signature      string    `json:"signature,omitempty"` // base64url(Ed25519 signature over canonical event fields)
}

// RevocationReason enumerates supported revocation reason codes.
// Keeping as string type ensures forward compatibility with external logs.
type RevocationReason string

const (
	RevocationReasonCompromise     RevocationReason = "compromise"         // key/material compromise suspected
	RevocationReasonUserRequest    RevocationReason = "user_request"       // end-user voluntary revocation
	RevocationReasonGrantorRevoked RevocationReason = "revoked_by_grantor" // grantor explicitly revoked
	RevocationReasonPolicyExpired  RevocationReason = "policy_expired"     // policy condition invalidated (e.g., org membership)
	RevocationReasonSuperseded     RevocationReason = "superseded"         // replaced by a new delegation / rotation
	RevocationReasonAbuse          RevocationReason = "abuse_detected"     // abuse / anomaly detection triggered
)

// validateReason enforces known reasons; empty reason is allowed (becomes grantor_revoked default).
func validateReason(r string) string {
	if r == "" {
		return string(RevocationReasonGrantorRevoked)
	}
	switch RevocationReason(r) {
	case RevocationReasonCompromise, RevocationReasonUserRequest, RevocationReasonGrantorRevoked,
		RevocationReasonPolicyExpired, RevocationReasonSuperseded, RevocationReasonAbuse:
		return r
	default:
		return string(RevocationReasonGrantorRevoked) // fallback to safe generic reason
	}
}

// RevocationChain maintains ordered revocation events.
// RevocationChain now maintains optional Signed Tree Head snapshots (Phase 4).
// treeHeads is append-only history of signed roots for audit / persistence.
type RevocationChain struct {
	events    []RevocationEvent
	merkle    *MerkleTree
	treeHeads []*SignedTreeHead
}

// OnRevocationAppended is an optional callback invoked after a revocation event is successfully appended.
// It is intended for audit logging / provenance emission (ID, hash, chain length, aggregate hash).
var OnRevocationAppended func(ev RevocationEvent, chainLen int, aggregateHash string)

// NewRevocationChain constructs an empty chain.
func NewRevocationChain() *RevocationChain {
	return &RevocationChain{events: make([]RevocationEvent, 0), merkle: NewMerkleTree(), treeHeads: make([]*SignedTreeHead, 0)}
}

// Append adds a new revocation event computing linkage and hash integrity.
func (c *RevocationChain) Append(e RevocationEvent) (RevocationEvent, error) {
	if e.ID == "" {
		return RevocationEvent{}, errors.New("revocation event id required")
	}
	if e.DelegationID == "" && e.DelegationHash == "" {
		return RevocationEvent{}, errors.New("delegation id or hash required")
	}
	e.Reason = validateReason(e.Reason)
	e.RevokedAt = time.Now().UTC()
	if len(c.events) > 0 {
		e.PrevHash = c.events[len(c.events)-1].Hash
	}
	h, err := hashRevocationEvent(e)
	if err != nil {
		return RevocationEvent{}, err
	}
	e.Hash = h
	// Optional signing (Phase 2): if EdDSA manager active, sign canonical bytes of event excluding Signature fields.
	// We perform signing after computing the hash to bind both raw fields and computed linkage hash.
	if km := crypto.GlobalEdDSARegistry; km != nil {
		if ak := km.Active(); ak != nil {
			payload, perr := signableBytes(e)
			if perr == nil {
				sig := ed25519.Sign(ak.Private, payload)
				e.SigKid = ak.ID
				e.Signature = base64.RawURLEncoding.EncodeToString(sig)
			}
		}
	}
	c.events = append(c.events, e)
	// Merkle append (Phase 3): incorporate event hash
	if c.merkle != nil {
		c.merkle.AppendLeaf(e.Hash)
	}
	if OnRevocationAppended != nil {
		OnRevocationAppended(e, len(c.events), c.AggregateHash())
	}
	return e, nil
}

// Events returns a copy of the underlying slice for external iteration.
func (c *RevocationChain) Events() []RevocationEvent {
	out := make([]RevocationEvent, len(c.events))
	copy(out, c.events)
	return out
}

// Verify ensures hash integrity, linkage and chronological ordering (no future timestamps).
func (c *RevocationChain) Verify() error {
	for i, e := range c.events {
		h, err := hashRevocationEvent(e)
		if err != nil {
			return err
		}
		if h != e.Hash {
			return fmt.Errorf("revocation event hash mismatch at %d", i)
		}
		if i == 0 && e.PrevHash != "" {
			return errors.New("genesis revocation event has prev hash")
		}
		if i > 0 && e.PrevHash != c.events[i-1].Hash {
			return fmt.Errorf("broken revocation prev hash link at %d", i)
		}
		if e.RevokedAt.After(time.Now().Add(2 * time.Minute)) { // tolerate minor clock skew
			return fmt.Errorf("revocation event timestamp in future at %d", i)
		}
		// Signature verification (if present). We treat absence as acceptable (legacy mode), but if present must verify.
		if e.Signature != "" {
			km := crypto.GlobalEdDSARegistry
			if km == nil {
				return fmt.Errorf("signature present but no key manager available at %d", i)
			}
			payload, perr := signableBytes(e)
			if perr != nil {
				return fmt.Errorf("signable bytes error at %d: %v", i, perr)
			}
			sigBytes, derr := base64.RawURLEncoding.DecodeString(e.Signature)
			if derr != nil {
				return fmt.Errorf("invalid base64 signature at %d", i)
			}
			if err := km.ValidateSignature(e.SigKid, payload, sigBytes); err != nil {
				return fmt.Errorf("signature verification failed at %d: %v", i, err)
			}
		}
	}
	return nil
}

// IsDelegationRevoked checks whether a given delegation ID or hash appears in the chain.
func (c *RevocationChain) IsDelegationRevoked(delegationID, delegationHash string) bool {
	if c == nil {
		return false
	}
	for _, e := range c.events {
		if delegationID != "" && e.DelegationID == delegationID {
			return true
		}
		if delegationHash != "" && e.DelegationHash == delegationHash {
			return true
		}
	}
	return false
}

// BuildRevocationIndex converts the chain into a RevocationIndex (by DelegationID only) for compatibility.
func (c *RevocationChain) BuildRevocationIndex() *RevocationIndex {
	if c == nil {
		return NewRevocationIndex(nil)
	}
	list := make([]DelegationRevocation, 0, len(c.events))
	for _, e := range c.events {
		if e.DelegationID == "" {
			continue
		}
		list = append(list, DelegationRevocation{DelegationID: e.DelegationID, Reason: e.Reason})
	}
	return NewRevocationIndex(list)
}

// ValidateDelegationChainWithRevocations verifies the delegation chain integrity, then the revocation chain integrity,
// then ensures no active (non-expired) delegation has been revoked. Returns an error if integrity fails or revoked found.
func ValidateDelegationChainWithRevocations(chain *Chain, revChain *RevocationChain) error {
	if chain == nil {
		return errors.New("delegation chain required")
	}
	if err := chain.VerifyChain(); err != nil {
		return fmt.Errorf("delegation chain integrity: %w", err)
	}
	if revChain != nil {
		if err := revChain.Verify(); err != nil {
			return fmt.Errorf("revocation chain integrity: %w", err)
		}
		// check each delegation active state then revocation presence
		now := time.Now().UTC()
		for _, d := range chain.items { // same package access
			if now.After(d.ExpiresAt) {
				continue
			} // expired delegations ignored
			if revChain.IsDelegationRevoked(d.ID, d.Hash) {
				return fmt.Errorf("delegation %s revoked", d.ID)
			}
		}
	}
	return nil
}

func hashRevocationEvent(e RevocationEvent) (string, error) {
	tmp := struct {
		ID             string    `json:"id"`
		DelegationID   string    `json:"delegation_id"`
		DelegationHash string    `json:"delegation_hash"`
		Reason         string    `json:"reason,omitempty"`
		RevokedAt      time.Time `json:"revoked_at"`
		PrevHash       string    `json:"prev_hash"`
	}{e.ID, e.DelegationID, e.DelegationHash, e.Reason, e.RevokedAt, e.PrevHash}
	data, err := json.Marshal(tmp)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

// signableBytes constructs canonical bytes over the event fields we commit to (excluding signature fields to avoid recursion).
func signableBytes(e RevocationEvent) ([]byte, error) {
	tmp := struct {
		ID             string    `json:"id"`
		DelegationID   string    `json:"delegation_id"`
		DelegationHash string    `json:"delegation_hash"`
		Reason         string    `json:"reason,omitempty"`
		RevokedAt      time.Time `json:"revoked_at"`
		PrevHash       string    `json:"prev_hash"`
		Hash           string    `json:"hash"`
	}{e.ID, e.DelegationID, e.DelegationHash, e.Reason, e.RevokedAt, e.PrevHash, e.Hash}
	return json.Marshal(tmp)
}

// NOTE: The revocation chain now directly depends on `internal/crypto.GlobalEdDSARegistry`.
// This simplifies signature logic and removes reflection-like indirection used earlier to avoid cycles.
// If the global registry is nil or token sig mode isn't eddsa, events remain unsigned (backward compatible).

// AggregateHash computes a stable hash over the entire revocation chain contents (order-dependent).
// Domain separated with prefix to avoid collision with individual event hashes.
func (c *RevocationChain) AggregateHash() string {
	if c == nil || len(c.events) == 0 {
		return ""
	}
	// Copy shallow slice to avoid mutation while sorting (though order must be preserved). We rely on original order.
	// Build JSON array deterministically of events' hash values and IDs.
	type mini struct{ ID, Hash string }
	arr := make([]mini, len(c.events))
	for i, e := range c.events {
		arr[i] = mini{ID: e.ID, Hash: e.Hash}
	}
	// We intentionally DO NOT sort to preserve sequence semantics; however we also compute a secondary digest that is
	// order-insensitive for optional comparison by sorting by ID then hashing again and concatenating, giving two dimensions.
	seqBytes, _ := json.Marshal(arr)
	hSeq := sha256.Sum256(append([]byte("GAuthRevocationChain_v1_seq:"), seqBytes...))
	// Order-insensitive component
	sorted := make([]mini, len(arr))
	copy(sorted, arr)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	setBytes, _ := json.Marshal(sorted)
	hSet := sha256.Sum256(append([]byte("GAuthRevocationChain_v1_set:"), setBytes...))
	combo := sha256.Sum256(append(hSeq[:], hSet[:]...))
	return hex.EncodeToString(combo[:])
}

// MerkleRoot returns current Merkle root (empty if no events or merkle disabled)
func (c *RevocationChain) MerkleRoot() string {
	if c == nil || c.merkle == nil {
		return ""
	}
	return c.merkle.Root()
}

// SignedTreeHead represents a signed commitment to the current revocation chain state.
// Multiple signatures can be attached (multi-sig scenario in later phase). For now, single EdDSA signature.
type SignedTreeHead struct {
	Version         int                 `json:"version"`
	MerkleRoot      string              `json:"merkle_root"`
	ChainLength     int                 `json:"chain_length"`
	AggregateHash   string              `json:"aggregate_hash"`
	Timestamp       time.Time           `json:"timestamp"`
	Signatures      []TreeHeadSignature `json:"signatures"`
	Threshold       int                 `json:"threshold,omitempty"`        // required cumulative weight (or count) for validity when multi-sig
	WeightsTotal    int                 `json:"weights_total,omitempty"`    // total available weight among signers
	SatisfiedWeight int                 `json:"satisfied_weight,omitempty"` // cumulative weight of attached signatures
}

// TreeHeadSignature holds an individual signature over the tree head canonical payload.
type TreeHeadSignature struct {
	Kid    string `json:"kid"`
	Alg    string `json:"alg"`
	Sig    string `json:"sig"`              // base64url(Ed25519 signature)
	Weight int    `json:"weight,omitempty"` // reserved for multi-sig phase (not populated yet)
}

// signableTreeHeadBytes produces canonical JSON bytes excluding signatures.
func signableTreeHeadBytes(sth *SignedTreeHead) ([]byte, error) {
	if sth == nil {
		return nil, errors.New("nil_sth")
	}
	ts := sth.Timestamp.UTC().Format(time.RFC3339) // canonical string (no fractional seconds)
	if sth.Version >= 2 {
		payload := struct {
			Version       int    `json:"version"`
			MerkleRoot    string `json:"merkle_root"`
			ChainLength   int    `json:"chain_length"`
			AggregateHash string `json:"aggregate_hash"`
			Timestamp     string `json:"timestamp"`
			Threshold     int    `json:"threshold"`
			WeightsTotal  int    `json:"weights_total"`
		}{sth.Version, sth.MerkleRoot, sth.ChainLength, sth.AggregateHash, ts, sth.Threshold, sth.WeightsTotal}
		return json.Marshal(payload)
	}
	tmp := struct {
		Version       int    `json:"version"`
		MerkleRoot    string `json:"merkle_root"`
		ChainLength   int    `json:"chain_length"`
		AggregateHash string `json:"aggregate_hash"`
		Timestamp     string `json:"timestamp"`
	}{sth.Version, sth.MerkleRoot, sth.ChainLength, sth.AggregateHash, ts}
	return json.Marshal(tmp)
}

// LatestTreeHead returns most recent signed tree head (nil if none signed yet).
func (c *RevocationChain) LatestTreeHead() *SignedTreeHead {
	if c == nil || len(c.treeHeads) == 0 {
		return nil
	}
	return c.treeHeads[len(c.treeHeads)-1]
}

// TreeHeads returns a copy of the signed tree head history.
func (c *RevocationChain) TreeHeads() []*SignedTreeHead {
	if c == nil {
		return nil
	}
	out := make([]*SignedTreeHead, len(c.treeHeads))
	copy(out, c.treeHeads)
	return out
}

// SignTreeHead creates and signs a new tree head snapshot using the active EdDSA key manager (if available).
// If key manager unavailable, returns unsigned tree head (Signatures slice empty) – still useful for anchoring.
func (c *RevocationChain) SignTreeHead() (*SignedTreeHead, error) {
	if c == nil {
		return nil, errors.New("nil_chain")
	}
	// If there are no revocation events yet, suppress creation of a SignedTreeHead.
	// Previously we would emit an empty tree head (chain_length=0, empty merkle root). This led to
	// confusing logs and, in some callback scenarios, index-out-of-range panics in higher level
	// logic that assumed at least one event existed when a rotation occurred. Returning (nil,nil)
	// makes callers explicit about handling the empty-chain case while remaining backward
	// compatible for code paths already checking for nil.
	if len(c.events) == 0 {
		return nil, nil
	}
	// Determine multi-sig threshold from environment
	threshold := 1
	if raw := os.Getenv("GAUTH_MULTI_SIG_THRESHOLD"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			threshold = v
		}
	}
	sth := &SignedTreeHead{Version: 1, MerkleRoot: c.MerkleRoot(), ChainLength: len(c.events), AggregateHash: c.AggregateHash(), Timestamp: time.Now().UTC()}
	weightsMap := map[string]int{}
	if raw := os.Getenv("GAUTH_MULTI_SIG_WEIGHTS"); raw != "" {
		parts := strings.Split(raw, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			kv := strings.SplitN(p, "=", 2)
			if len(kv) != 2 {
				continue
			}
			w, err := strconv.Atoi(kv[1])
			if err != nil || w <= 0 {
				continue
			}
			weightsMap[kv[0]] = w
		}
	}
	// Multi-sig engaged when threshold >1.
	if threshold > 1 {
		sth.Version = 2
		sth.Threshold = threshold
	}
	if km := crypto.GlobalEdDSARegistry; km != nil {
		keys := km.ListCurrent()
		// Compute total available weights (or count fallback)
		availableTotal := 0
		for _, k := range keys {
			if len(weightsMap) > 0 {
				availableTotal += weightsMap[k.ID]
			} else {
				availableTotal++
			}
		}
		sth.WeightsTotal = availableTotal
		// Sign sequentially until threshold satisfied (or include all if threshold unreachable)
		cumulative := 0
		for _, k := range keys {
			payload, err := signableTreeHeadBytes(sth)
			if err != nil {
				return nil, err
			}
			sig := ed25519.Sign(k.Private, payload)
			w := 1
			if len(weightsMap) > 0 {
				if weightsMap[k.ID] > 0 {
					w = weightsMap[k.ID]
				}
			}
			sth.Signatures = append(sth.Signatures, TreeHeadSignature{Kid: k.ID, Alg: "EdDSA", Sig: base64.RawURLEncoding.EncodeToString(sig), Weight: w})
			cumulative += w
			sth.SatisfiedWeight = cumulative
			if threshold > 1 && cumulative >= threshold {
				break
			}
		}
		// If threshold wasn't met (e.g., not enough keys), still return sth but caller verification will fail.
	}
	c.treeHeads = append(c.treeHeads, sth)
	// Optional persistence
	if p := os.Getenv("GAUTH_STH_PERSIST_PATH"); p != "" {
		_ = c.SaveSignedTreeHeads(p) // best-effort; ignore error (log could be added later)
	}
	return sth, nil
}

// VerifyTreeHeadSignature verifies the first signature on a tree head (multi-sig expansion later).
func VerifyTreeHeadSignature(sth *SignedTreeHead) error {
	if sth == nil {
		return errors.New("nil_sth")
	}
	if len(sth.Signatures) == 0 {
		return errors.New("no_signatures")
	}
	sigEntry := sth.Signatures[0]
	km := crypto.GlobalEdDSARegistry
	if km == nil {
		return errors.New("no_key_manager")
	}
	payload, err := signableTreeHeadBytes(sth)
	if err != nil {
		return err
	}
	sigBytes, err := base64.RawURLEncoding.DecodeString(sigEntry.Sig)
	if err != nil {
		return fmt.Errorf("sig_decode: %w", err)
	}
	if err := km.ValidateSignature(sigEntry.Kid, payload, sigBytes); err != nil {
		return fmt.Errorf("signature_invalid: %w", err)
	}
	return nil
}

// VerifyTreeHeadMultiSig checks cumulative weights (or signature count fallback) against threshold.
func VerifyTreeHeadMultiSig(sth *SignedTreeHead) error {
	if sth == nil {
		return errors.New("nil_sth")
	}
	if sth.Threshold <= 1 { // fallback to single signature verification
		return VerifyTreeHeadSignature(sth)
	}
	if len(sth.Signatures) == 0 {
		return errors.New("no_signatures")
	}
	// First verify each signature cryptographically.
	km := crypto.GlobalEdDSARegistry
	if km == nil {
		return errors.New("no_key_manager")
	}
	payload, err := signableTreeHeadBytes(sth)
	if err != nil {
		return err
	}
	cumulative := 0
	for i, sigEntry := range sth.Signatures {
		sigBytes, derr := base64.RawURLEncoding.DecodeString(sigEntry.Sig)
		if derr != nil {
			return fmt.Errorf("sig_decode_%d: %v", i, derr)
		}
		if err := km.ValidateSignature(sigEntry.Kid, payload, sigBytes); err != nil {
			return fmt.Errorf("signature_invalid_%d: %v", i, err)
		}
		w := sigEntry.Weight
		if w <= 0 {
			w = 1
		}
		cumulative += w
	}
	if cumulative < sth.Threshold {
		return fmt.Errorf("threshold_not_met: have=%d need=%d", cumulative, sth.Threshold)
	}
	return nil
}

// SaveSignedTreeHeads writes the current signed tree head history to disk as JSON array.
func (c *RevocationChain) SaveSignedTreeHeads(path string) error {
	if c == nil {
		return errors.New("nil_chain")
	}
	data, err := json.MarshalIndent(c.treeHeads, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// LoadSignedTreeHeads loads signed tree heads from disk, verifies each signature set, and replaces history if valid.
// Invalid entries are skipped; if none valid, existing in-memory history remains unchanged.
func (c *RevocationChain) LoadSignedTreeHeads(path string) error {
	if c == nil {
		return errors.New("nil_chain")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var loaded []*SignedTreeHead
	if err := json.Unmarshal(b, &loaded); err != nil {
		return err
	}
	var valid []*SignedTreeHead
	for _, sth := range loaded {
		if sth == nil {
			continue
		}
		// Choose verification path
		var verr error
		if sth.Threshold > 1 {
			verr = VerifyTreeHeadMultiSig(sth)
		} else {
			verr = VerifyTreeHeadSignature(sth)
		}
		if verr == nil {
			valid = append(valid, sth)
		}
	}
	if len(valid) > 0 {
		c.treeHeads = valid
	}
	return nil
}

// ConsistencyProof demonstrates append-only growth between two tree heads (start -> end).
// For prototype we include the intermediate leaf hashes of the new range and re-compute root incrementally for verification.
// More efficient schemes (RFC6962 style) could be adopted later.
type ConsistencyProof struct {
	StartLength int      `json:"start_length"`
	EndLength   int      `json:"end_length"`
	StartRoot   string   `json:"start_root"`
	EndRoot     string   `json:"end_root"`
	NewLeaves   []string `json:"new_leaves"` // raw event hashes appended after start snapshot
}

// ConsistencyProofV2 follows an RFC6962-like logarithmic proof (audit path) between two tree sizes.
// It contains a list of intermediate node hashes sufficient to reconstruct both start and end roots proving append-only growth.
// Reference: Certificate Transparency RFC6962 Section 2.1.3 (adapted for our domain-separated hash scheme).
type ConsistencyProofV2 struct {
	StartLength int      `json:"start_length"`
	EndLength   int      `json:"end_length"`
	Path        []string `json:"path"`       // audit path node digests (siblings)
	Positions   []string `json:"positions"`  // sibling position relative to evolving hash: "L" or "R"
	StartRoot   string   `json:"start_root"` // optional for convenience
	EndRoot     string   `json:"end_root"`   // optional for convenience
	// PrefixRoots & PrefixSizes provide an alternate reconstruction path for StartRoot without full tree rebuild.
	// They represent maximal power-of-two aligned subtree roots whose concatenation (in order) covers the first StartLength leaves.
	// Example: StartLength=13 -> binary decomposition 8 + 4 + 1 yields PrefixSizes [8,4,1].
	// PrefixRoots hold the Merkle root digest of each block's subtree. Verification can rebuild StartRoot by treating
	// these roots as leaves of a meta-tree applying the same copy-up parent formation.
	PrefixRoots []string `json:"prefix_roots,omitempty"`
	PrefixSizes []int    `json:"prefix_sizes,omitempty"`
	// PrefixBridges optionally supplies the parent digests formed when consecutively merging prefix blocks.
	// For blocks B0,B1,B2,... we compute parent hash P01 = HASH_NODE(B0,B1), then P012 = HASH_NODE(P01,B2), etc.
	// This linear merge chain enables O(k) verification (k=number of blocks) without constructing the meta-tree levels.
	// If provided, fast reconstruction can validate each bridge digest while folding toward StartRoot.
	PrefixBridges []string `json:"prefix_bridges,omitempty"`
}

// GenerateConsistencyProof builds a consistency proof between a previous tree head index and current latest.
// startIndex refers to index within c.treeHeads slice. Returns error if indices invalid or no growth.
func (c *RevocationChain) GenerateConsistencyProof(startIndex int) (*ConsistencyProof, error) {
	if c == nil {
		return nil, errors.New("nil_chain")
	}
	if startIndex < 0 || startIndex >= len(c.treeHeads) {
		return nil, errors.New("invalid_start_index")
	}
	latest := c.LatestTreeHead()
	if latest == nil {
		return nil, errors.New("no_latest_tree_head")
	}
	start := c.treeHeads[startIndex]
	if start.ChainLength >= latest.ChainLength {
		return nil, errors.New("no_growth")
	}
	// Collect new leaves event hashes
	newHashes := []string{}
	for i := start.ChainLength; i < len(c.events); i++ {
		newHashes = append(newHashes, c.events[i].Hash)
	}
	proof := &ConsistencyProof{StartLength: start.ChainLength, EndLength: latest.ChainLength, StartRoot: start.MerkleRoot, EndRoot: latest.MerkleRoot, NewLeaves: newHashes}
	return proof, nil
}

// GenerateConsistencyProofV2 builds a logarithmic sized consistency proof between tree head at startIndex and latest.
// startIndex selects the historical SignedTreeHead in c.treeHeads slice. Implementation derived from RFC6962 algorithm.
//
//nolint:gocyclo // Merkle proof generation with path construction
func (c *RevocationChain) GenerateConsistencyProofV2(startIndex int) (*ConsistencyProofV2, error) {
	if c == nil {
		return nil, errors.New("nil_chain")
	}
	if startIndex < 0 || startIndex >= len(c.treeHeads) {
		return nil, errors.New("invalid_start_index")
	}
	latest := c.LatestTreeHead()
	if latest == nil {
		return nil, errors.New("no_latest_tree_head")
	}
	start := c.treeHeads[startIndex]
	if start.ChainLength == 0 {
		return nil, errors.New("start_chain_empty")
	}
	if start.ChainLength >= latest.ChainLength {
		return nil, errors.New("no_growth")
	}
	oldSize := start.ChainLength
	newSize := latest.ChainLength
	// Pre-hash leaves (leaf domain). This is O(n) but we avoid building all level arrays; future improvement may stream only needed ranges.
	leaves := make([]string, len(c.events))
	for i, ev := range c.events {
		leaves[i] = LeafDigestForEventHash(ev.Hash)
	}

	// --- Prefix block decomposition via streaming segment stack ---
	type segment struct {
		h  string
		sz int
	}
	stack := []segment{}
	// appendSegment integrates a new leaf hash forming canonical power-of-two segments.
	appendSegment := func(leafHash string) {
		stack = append(stack, segment{h: leafHash, sz: 1})
		// Merge while top two have equal size.
		for len(stack) >= 2 {
			a := stack[len(stack)-2]
			b := stack[len(stack)-1]
			if a.sz != b.sz {
				break
			}
			parent := sha256.Sum256(append(append([]byte("GAUTH_MERKLE_NODE:"), []byte(a.h)...), []byte(b.h)...))
			ph := hex.EncodeToString(parent[:])
			// replace a with merged, pop b
			stack[len(stack)-2] = segment{h: ph, sz: a.sz * 2}
			stack = stack[:len(stack)-1]
		}
	}
	for i := 0; i < oldSize; i++ {
		appendSegment(leaves[i])
	}
	// Snapshot prefix decomposition left-to-right: stack currently holds segments ordered left->right
	prefixRoots := make([]string, len(stack))
	prefixSizes := make([]int, len(stack))
	for i, s := range stack {
		prefixRoots[i] = s.h
		prefixSizes[i] = s.sz
	}
	// Build bridges with right-to-left reduction to mirror multi-block reconstruction algorithm.
	prefixBridges := []string{}
	if len(stack) > 1 {
		tmp := make([]segment, len(stack))
		copy(tmp, stack)
		for len(tmp) > 1 {
			last := tmp[len(tmp)-1]
			prev := tmp[len(tmp)-2]
			parent := sha256.Sum256(append(append([]byte("GAUTH_MERKLE_NODE:"), []byte(prev.h)...), []byte(last.h)...))
			ph := hex.EncodeToString(parent[:])
			prefixBridges = append(prefixBridges, ph)
			tmp[len(tmp)-2] = segment{h: ph, sz: prev.sz + last.sz}
			tmp = tmp[:len(tmp)-1]
			for len(tmp) >= 2 { // collapse equal sizes greedily
				a := tmp[len(tmp)-2]
				b := tmp[len(tmp)-1]
				if a.sz != b.sz {
					break
				}
				parent2 := sha256.Sum256(append(append([]byte("GAUTH_MERKLE_NODE:"), []byte(a.h)...), []byte(b.h)...))
				ph2 := hex.EncodeToString(parent2[:])
				prefixBridges = append(prefixBridges, ph2)
				tmp[len(tmp)-2] = segment{h: ph2, sz: a.sz * 2}
				tmp = tmp[:len(tmp)-1]
			}
		}
	}

	// --- Consistency path construction ---
	// Interval-based streaming alternative avoids full temp tree build when GAUTH_CONSISTENCY_V2_INTERVAL_PATH=1.
	useInterval := os.Getenv("GAUTH_CONSISTENCY_V2_INTERVAL_PATH") == "1"
	if useInterval {
		// We derive the audit path between oldSize and newSize using binary interval decomposition.
		// Algorithm sketch (RFC6962 inspired):
		// 1. Decompose oldSize into power-of-two blocks (already in prefixSizes/prefixRoots).
		// 2. Starting from virtual node representing the entire old tree, walk upward comparing with newSize to decide inclusion.
		// For simplicity in this prototype we fall back to a lightweight node cache built only for needed siblings.
		// We compute required sibling digests on-the-fly by hashing ranges of leaves using the same segment merge logic.
		path := []string{}
		positions := []string{}
		// Helper to compute root of range [l,r) quickly by streaming merge (copy-up) without global rebuild.
		computeRangeRoot := func(l, r int) string {
			if l >= r {
				return ""
			}
			segStack := []struct {
				h  string
				sz int
			}{}
			push := func(h string) {
				segStack = append(segStack, struct {
					h  string
					sz int
				}{h, 1})
				for len(segStack) >= 2 {
					A := segStack[len(segStack)-2]
					B := segStack[len(segStack)-1]
					if A.sz != B.sz {
						break
					}
					p := sha256.Sum256(append(append([]byte("GAUTH_MERKLE_NODE:"), []byte(A.h)...), []byte(B.h)...))
					segStack[len(segStack)-2] = struct {
						h  string
						sz int
					}{hex.EncodeToString(p[:]), A.sz * 2}
					segStack = segStack[:len(segStack)-1]
				}
			}
			for i := l; i < r; i++ {
				push(leaves[i])
			}
			// fold stack left-to-right
			for len(segStack) > 1 {
				A := segStack[0]
				B := segStack[1]
				p := sha256.Sum256(append(append([]byte("GAUTH_MERKLE_NODE:"), []byte(A.h)...), []byte(B.h)...))
				segStack[0] = struct {
					h  string
					sz int
				}{hex.EncodeToString(p[:]), A.sz + B.sz}
				segStack = append(segStack[:1], segStack[2:]...)
				// attempt merges with next if equal sizes repeat
				for len(segStack) >= 2 {
					A = segStack[0]
					B = segStack[1]
					if A.sz != B.sz {
						break
					}
					p2 := sha256.Sum256(append(append([]byte("GAUTH_MERKLE_NODE:"), []byte(A.h)...), []byte(B.h)...))
					segStack[0] = struct {
						h  string
						sz int
					}{hex.EncodeToString(p2[:]), A.sz * 2}
					segStack = append(segStack[:1], segStack[2:]...)
				}
			}
			return segStack[0].h
		}
		// We iteratively compute siblings needed from oldSize to reach newSize.
		// For each step, consider whether oldSize is odd or whether merging with next block crosses newSize.
		old := oldSize
		end := newSize
		for old < end {
			// Handle odd old (right child scenario): include left sibling covering [old-1, old)
			if old%2 == 1 {
				leftRoot := computeRangeRoot(old-1, old)
				path = append(path, leftRoot)
				positions = append(positions, "L")
				old-- // normalize to even boundary
				continue
			}
			// Determine largest power-of-two block ending at old.
			lsb := old & -old
			if lsb == 0 {
				break
			}
			blockStart := old - lsb
			_ = computeRangeRoot(blockStart, old)
			next := old + lsb
			if next <= end {
				sibRoot := computeRangeRoot(old, next)
				path = append(path, sibRoot)
				positions = append(positions, "R")
				old = next
				continue
			}
			break
		}
		if len(path) > 0 {
			proof := &ConsistencyProofV2{StartLength: oldSize, EndLength: newSize, Path: path, Positions: positions, StartRoot: start.MerkleRoot, EndRoot: latest.MerkleRoot, PrefixRoots: prefixRoots, PrefixSizes: prefixSizes, PrefixBridges: prefixBridges}
			return proof, nil
		}
		// else fall through to legacy path construction below
	}
	// Fallback: original temporary tree traversal for correctness.
	tempTree := NewMerkleTree()
	for _, ev := range c.events {
		tempTree.AppendLeaf(ev.Hash)
	}
	tempTree.rebuildIfNeeded()
	path := []string{}
	positions := []string{}
	oldIdx := oldSize - 1
	newIdx := newSize - 1
	level := 0
	maxLevel := len(tempTree.levels) - 1
	for level < maxLevel && oldIdx != newIdx {
		nodes := tempTree.levels[level]
		if oldIdx%2 == 1 { // right child -> include left sibling
			sibling := nodes[oldIdx-1].digest
			path = append(path, sibling)
			positions = append(positions, "L")
			oldIdx /= 2
			newIdx /= 2
			level++
			continue
		}
		if oldIdx != newIdx && oldIdx+1 < len(nodes) { // even oldIdx include right sibling present in new tree
			sibling := nodes[oldIdx+1].digest
			path = append(path, sibling)
			positions = append(positions, "R")
		}
		oldIdx /= 2
		newIdx /= 2
		level++
	}
	for level < maxLevel {
		nodes := tempTree.levels[level]
		if newIdx%2 == 0 && newIdx+1 < len(nodes) {
			sibling := nodes[newIdx+1].digest
			path = append(path, sibling)
			positions = append(positions, "R")
		}
		newIdx /= 2
		level++
	}
	proof := &ConsistencyProofV2{StartLength: oldSize, EndLength: newSize, Path: path, Positions: positions, StartRoot: start.MerkleRoot, EndRoot: latest.MerkleRoot, PrefixRoots: prefixRoots, PrefixSizes: prefixSizes, PrefixBridges: prefixBridges}
	return proof, nil
}

// VerifyConsistencyProofV2 verifies logarithmic consistency proof ensuring append-only growth.
// Recomputes old and new roots using provided path. For prototype we simply trust provided StartRoot/EndRoot fields
// after reconstructing from full leaf set for integrity, then ensure that path enables derivation following our recorded sequence.
//
//nolint:gocyclo // Merkle proof verification logic
//nolint:gocyclo // Merkle proof verification logic
func VerifyConsistencyProofV2(proof *ConsistencyProofV2, allEventHashes []string) error {
	if proof == nil {
		return errors.New("nil_proof")
	}
	if proof.StartLength <= 0 || proof.EndLength <= proof.StartLength {
		return errors.New("invalid_lengths")
	}
	if len(allEventHashes) < proof.EndLength {
		return errors.New("insufficient_event_hashes")
	}
	if len(proof.Path) == 0 || len(proof.Positions) != len(proof.Path) {
		return errors.New("invalid_path")
	}
	// Rebuild full start tree (retain O(n) for correctness) but validate prefix decomposition invariants when provided.
	startTree := NewMerkleTree()
	for i := 0; i < proof.StartLength; i++ {
		startTree.AppendLeaf(allEventHashes[i])
	}
	canonicalStartRoot := startTree.Root()
	if canonicalStartRoot != proof.StartRoot {
		return fmt.Errorf("start_root_mismatch: expected %s got %s", proof.StartRoot, canonicalStartRoot)
	}
	if len(proof.PrefixRoots) > 0 {
		if len(proof.PrefixRoots) != len(proof.PrefixSizes) {
			return errors.New("prefix_roots_sizes_length_mismatch")
		}
		sum := 0
		for _, s := range proof.PrefixSizes {
			if s <= 0 || (s&(s-1)) != 0 {
				return fmt.Errorf("prefix_size_not_power_of_two:%d", s)
			}
			sum += s
		}
		if sum != proof.StartLength {
			return fmt.Errorf("prefix_sizes_sum_mismatch: have=%d expected=%d", sum, proof.StartLength)
		}
		startTree.rebuildIfNeeded()
		offset := 0
		for i, blkSize := range proof.PrefixSizes {
			// level k for size 2^k
			k := 0
			for (1 << k) < blkSize {
				k++
			}
			if k >= len(startTree.levels) {
				return fmt.Errorf("prefix_level_oob:%d", k)
			}
			nodeIdx := offset >> k
			levelNodes := startTree.levels[k]
			if nodeIdx < 0 || nodeIdx >= len(levelNodes) {
				return fmt.Errorf("prefix_node_idx_oob: level=%d idx=%d", k, nodeIdx)
			}
			expectedDigest := levelNodes[nodeIdx].digest
			if expectedDigest != proof.PrefixRoots[i] {
				return fmt.Errorf("prefix_root_mismatch_at_%d", i)
			}
			offset += blkSize
		}
		// Attempt fast reconstruction when enabled; now supports multi-block using bridges.
		if fast := ReconstructStartRootFromPrefixBlocks(proof.PrefixRoots, proof.PrefixSizes, proof.StartLength, proof.PrefixBridges); fast != "" && fast != canonicalStartRoot {
			return fmt.Errorf("fast_reconstruction_mismatch: expected %s got %s", canonicalStartRoot, fast)
		}
	}
	endTree := NewMerkleTree()
	for i := 0; i < proof.EndLength; i++ {
		endTree.AppendLeaf(allEventHashes[i])
	}
	if endTree.Root() != proof.EndRoot {
		return fmt.Errorf("end_root_mismatch: expected %s got %s", proof.EndRoot, endTree.Root())
	}
	// Path integrity: ensure each path element corresponds to a node digest in end tree (excluding root) to detect tampering.
	endTree.rebuildIfNeeded()
	nodeSet := make(map[string]struct{})
	for levelIdx := 0; levelIdx < len(endTree.levels)-1; levelIdx++ { // exclude top root level
		for _, n := range endTree.levels[levelIdx] {
			nodeSet[n.digest] = struct{}{}
		}
	}
	for i, d := range proof.Path {
		if _, ok := nodeSet[d]; !ok {
			return fmt.Errorf("path_element_not_in_tree_at_%d", i)
		}
	}
	// Logarithmic bound check on path size.
	maxAllowed := 2
	for x := proof.EndLength; x > 1; x /= 2 {
		maxAllowed++
	}
	if len(proof.Path) > maxAllowed {
		return fmt.Errorf("path_too_long: have=%d max=%d", len(proof.Path), maxAllowed)
	}
	return nil
}

// VerifyConsistencyProof performs naive verification by reconstructing merkle tree from all events up to start length and end length
// and comparing supplied roots, then ensuring that appending NewLeaves transitions from start root to end root.
// This is O(n) for prototype; future optimization can adopt logarithmic RFC6962 proof format.
func VerifyConsistencyProof(proof *ConsistencyProof, allEventHashes []string) error {
	if proof == nil {
		return errors.New("nil_proof")
	}
	if proof.StartLength >= proof.EndLength {
		return errors.New("invalid_lengths")
	}
	if len(allEventHashes) < proof.EndLength {
		return errors.New("insufficient_event_hashes")
	}
	// Build start tree
	startTree := NewMerkleTree()
	for i := 0; i < proof.StartLength; i++ {
		startTree.AppendLeaf(allEventHashes[i])
	}
	if startTree.Root() != proof.StartRoot {
		return fmt.Errorf("start_root_mismatch: expected %s got %s", proof.StartRoot, startTree.Root())
	}
	// Build end tree
	endTree := NewMerkleTree()
	for i := 0; i < proof.EndLength; i++ {
		endTree.AppendLeaf(allEventHashes[i])
	}
	if endTree.Root() != proof.EndRoot {
		return fmt.Errorf("end_root_mismatch: expected %s got %s", proof.EndRoot, endTree.Root())
	}
	// Sanity: proof.NewLeaves should match slice allEventHashes[StartLength:EndLength]
	expectedNew := allEventHashes[proof.StartLength:proof.EndLength]
	if len(expectedNew) != len(proof.NewLeaves) {
		return errors.New("new_leaves_length_mismatch")
	}
	for i := range expectedNew {
		if expectedNew[i] != proof.NewLeaves[i] {
			return fmt.Errorf("new_leaves_mismatch_at_%d", i)
		}
	}
	return nil
}

// GenerateMerkleProof returns proof for event ID (lookup index). Error if not found.
func (c *RevocationChain) GenerateMerkleProof(eventID string) ([]MerkleProofStep, string, error) {
	if c == nil || c.merkle == nil {
		return nil, "", errors.New("merkle_not_enabled")
	}
	idx := -1
	for i, ev := range c.events {
		if ev.ID == eventID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil, "", errors.New("event_not_found")
	}
	return c.merkle.GenerateProof(idx)
}

// GenerateMerkleProofByIndex returns merkle proof for given 0-based index.
func (c *RevocationChain) GenerateMerkleProofByIndex(index int) ([]MerkleProofStep, string, error) {
	if c == nil || c.merkle == nil {
		return nil, "", errors.New("merkle_not_enabled")
	}
	if index < 0 || index >= len(c.events) {
		return nil, "", errors.New("index_out_of_range")
	}
	return c.merkle.GenerateProof(index)
}

// GenerateMerkleProofByHash returns proof for event hash (exact match) if present.
func (c *RevocationChain) GenerateMerkleProofByHash(hash string) ([]MerkleProofStep, string, error) {
	if c == nil || c.merkle == nil {
		return nil, "", errors.New("merkle_not_enabled")
	}
	idx := -1
	for i, ev := range c.events {
		if ev.Hash == hash {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil, "", errors.New("hash_not_found")
	}
	return c.merkle.GenerateProof(idx)
}
