package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
)

// Entry represents a single ledger append.
type Entry struct {
	ID      int    `json:"id"`
	Type    string `json:"type"` // e.g. poa_issue, poa_revoke, decision
	Digest  string `json:"digest"` // hex hash of payload data (caller pre-hashes or raw string hashed internally)
	Payload string `json:"payload"` // original payload (for demo transparency)
}

// Proof contains Merkle inclusion path (list of sibling hashes) plus computed root.
type Proof struct {
	EntryID    int      `json:"entry_id"`
	Digest     string   `json:"digest"`
	Siblings   []string `json:"siblings"`    // sibling hashes in ascent order
	Orientations []string `json:"orientations"` // "L" if original node was left child, "R" if right child at that level
	Root       string   `json:"root"`
}

// Ledger implements a simple in-memory append-only log with Merkle root snapshots.
type Ledger struct {
	mu     sync.RWMutex
	entries []Entry
	roots   []string // historical roots; index correlates to emission sequence
}

// New creates an empty ledger.
func New() *Ledger { return &Ledger{entries: []Entry{}, roots: []string{}} }

// hash concatenates child hashes deterministically (left || right) and returns hex SHA-256.
func hash(left, right string) string {
	h := sha256.New()
	h.Write([]byte(left))
	h.Write([]byte(right))
	return hex.EncodeToString(h.Sum(nil))
}

// leafHash ensures consistent leaf hashing.
func leafHash(digest string) string {
	h := sha256.New(); h.Write([]byte("leaf:")); h.Write([]byte(digest)); return hex.EncodeToString(h.Sum(nil))
}

// Append adds a new entry (caller passes raw payload; digest auto-derived if empty). Returns assigned entry ID and new root if re-emitted.
// Root emission strategy: emit a new root after every append (simplified for demo). In real systems root cadence may be time / batch based.
func (l *Ledger) Append(entryType, payload, digest string) (int, string) {
	l.mu.Lock(); defer l.mu.Unlock()
	if digest == "" { // derive digest from payload
		ph := sha256.Sum256([]byte(payload))
		digest = hex.EncodeToString(ph[:])
	}
	id := len(l.entries)
	l.entries = append(l.entries, Entry{ID: id, Type: entryType, Digest: digest, Payload: payload})
	root := l.computeRoot()
	l.roots = append(l.roots, root)
	return id, root
}

// computeRoot builds a binary Merkle tree (pairwise hashing, last hash duplicated if odd). Returns hex root of current entries slice.
func (l *Ledger) computeRoot() string {
	if len(l.entries) == 0 { return "" }
	// Start with leaf hashes
	layer := make([]string, len(l.entries))
	for i, e := range l.entries { layer[i] = leafHash(e.Digest) }
	for len(layer) > 1 {
		var next []string
		for i := 0; i < len(layer); i += 2 {
			if i+1 == len(layer) { // duplicate last
				next = append(next, hash(layer[i], layer[i]))
			} else {
				next = append(next, hash(layer[i], layer[i+1]))
			}
		}
		layer = next
	}
	return layer[0]
}

// LatestRoot returns the most recently emitted root (empty if none).
func (l *Ledger) LatestRoot() string {
	l.mu.RLock(); defer l.mu.RUnlock()
	if len(l.roots) == 0 { return "" }
	return l.roots[len(l.roots)-1]
}

// Get returns entry by ID.
func (l *Ledger) Get(id int) (Entry, bool) {
	l.mu.RLock(); defer l.mu.RUnlock()
	if id < 0 || id >= len(l.entries) { return Entry{}, false }
	return l.entries[id], true
}

// Size returns number of entries.
func (l *Ledger) Size() int { l.mu.RLock(); defer l.mu.RUnlock(); return len(l.entries) }

// Proof constructs inclusion proof for given entry ID.
func (l *Ledger) Proof(id int) (Proof, error) {
	l.mu.RLock(); defer l.mu.RUnlock()
	if id < 0 || id >= len(l.entries) { return Proof{}, fmt.Errorf("entry_not_found") }
	// Build leaf layer
	layer := make([]string, len(l.entries))
	for i, e := range l.entries { layer[i] = leafHash(e.Digest) }
	path := []string{}
	orient := []string{}
	idx := id
	work := layer
	for len(work) > 1 {
		// Determine sibling index and capture sibling hash
		var sibIdx int
		if idx%2 == 0 { // even -> sibling right (or duplicate if none)
			sibIdx = idx + 1
			if sibIdx >= len(work) { sibIdx = idx } // duplicate last when odd
			orient = append(orient, "L")
		} else { // odd -> sibling left
			sibIdx = idx - 1
			orient = append(orient, "R")
		}
		path = append(path, work[sibIdx])
		// Move up one layer - rebuild parent layer
		var next []string
		for i := 0; i < len(work); i += 2 {
			if i+1 == len(work) {
				next = append(next, hash(work[i], work[i]))
			} else {
				next = append(next, hash(work[i], work[i+1]))
			}
		}
		// Parent index is idx/2
		idx = idx / 2
		work = next
	}
	return Proof{EntryID: id, Digest: l.entries[id].Digest, Siblings: path, Orientations: orient, Root: work[0]}, nil
}

// HistoricalRoots returns copy of all emitted roots.
func (l *Ledger) HistoricalRoots() []string { l.mu.RLock(); defer l.mu.RUnlock(); out := make([]string, len(l.roots)); copy(out, l.roots); return out }