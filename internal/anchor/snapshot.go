package anchor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// Snapshot represents a signed view of the external anchor receipt chain.
// Prototype: Merkle root computed as SHA256 concat of all receipt hashes in order.
// Future: switch to tree construction for O(log n) inclusion proofs.
type Snapshot struct {
	RootHash    string   `json:"root_hash"`
	Count       int      `json:"count"`
	GeneratedAt string   `json:"generated_at"`
	Hashes      []string `json:"hashes"`
	Signature   string   `json:"signature,omitempty"` // placeholder (Ed25519 hex or base64 in future)
	Version     int      `json:"version"`
}

// BuildSnapshot computes a simple root hash (SHA256 over concatenated hashes) and returns snapshot.
func BuildSnapshot(receipts []Receipt) Snapshot {
	var hashes []string
	for _, r := range receipts {
		if r.Hash != "" {
			hashes = append(hashes, r.Hash)
		}
	}
	h := sha256.New()
	for _, hx := range hashes { _, _ = h.Write([]byte(hx)) }
	root := hex.EncodeToString(h.Sum(nil))
	return Snapshot{RootHash: root, Count: len(hashes), GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), Hashes: hashes, Version: 1}
}

// CanonicalBytes returns canonical JSON serialization used for signing.
func (s Snapshot) CanonicalBytes() []byte {
	b, _ := json.Marshal(struct {
		RootHash    string `json:"root_hash"`
		Count       int    `json:"count"`
		GeneratedAt string `json:"generated_at"`
		Version     int    `json:"version"`
	}{RootHash: s.RootHash, Count: s.Count, GeneratedAt: s.GeneratedAt, Version: s.Version})
	return b
}

// SignSnapshot is a placeholder that returns the snapshot unchanged.
// Future: accept signer interface (Ed25519) and embed signature.
func SignSnapshot(s Snapshot) Snapshot { return s }
