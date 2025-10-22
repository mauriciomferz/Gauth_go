// Package verification provides integration helpers for verifying revocation transparency
// artifacts (Merkle proofs, consistency proofs, discovery metadata). Test utilities live here.
package verification

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/crypto"
	delegation "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/delegation"
)

// buildTestServer constructs an HTTP test server exposing the subset of revocation endpoints
// needed by verification tests (discovery, verify, proof, jwks, optional consistency via enableConsistency flag).
func buildTestServer(rc *delegation.RevocationChain, enableConsistency bool) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/gauth-configuration", func(w http.ResponseWriter, r *http.Request) {
		sth := rc.LatestTreeHead()
		sigs := []map[string]any{}
		if sth != nil {
			for _, s := range sth.Signatures {
				sigs = append(sigs, map[string]any{"kid": s.Kid, "alg": s.Alg, "sig": s.Sig, "weight": s.Weight})
			}
		}
		sthJSON := map[string]any{}
		if sth != nil {
			sthJSON = map[string]any{"version": sth.Version, "merkle_root": sth.MerkleRoot, "chain_length": sth.ChainLength, "aggregate_hash": sth.AggregateHash, "timestamp": sth.Timestamp.Format(time.RFC3339), "signatures": sigs, "threshold": sth.Threshold, "weights_total": sth.WeightsTotal, "satisfied_weight": sth.SatisfiedWeight}
		}
		historySize := len(rc.TreeHeads())
		writeJSON(w, map[string]any{"revocation_support": map[string]any{"sth_latest": sthJSON, "sth_history_size": historySize}})
	})
	mux.HandleFunc("/api/v1/token/revocation/verify", func(w http.ResponseWriter, r *http.Request) {
		events := rc.Events()
		evJSON := make([]map[string]any, 0, len(events))
		for i, e := range events {
			evJSON = append(evJSON, map[string]any{"id": e.ID, "hash": e.Hash, "index": i})
		}
		writeJSON(w, map[string]any{"success": true, "events": evJSON, "length": len(events), "verified": true, "aggregate_hash": rc.AggregateHash()})
	})
	mux.HandleFunc("/api/v1/token/revocation/proof", func(w http.ResponseWriter, r *http.Request) {
		hash := r.URL.Query().Get("hash")
		idx := -1
		events := rc.Events()
		for i, e := range events {
			if e.Hash == hash {
				idx = i
				break
			}
		}
		if idx == -1 {
			writeJSON(w, map[string]any{"success": false, "target": hash})
			return
		}
		proof, root, _ := rc.GenerateMerkleProofByHash(hash)
		writeJSON(w, map[string]any{"success": true, "target": hash, "merkle_root": root, "proof": proof})
	})
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		km := crypto.GlobalEdDSARegistry
		keys := []map[string]any{}
		if km != nil && km.Active() != nil {
			k := km.Active()
			keys = append(keys, map[string]any{"kty": "OKP", "crv": "Ed25519", "kid": k.ID, "x": base64.RawURLEncoding.EncodeToString(k.Public)})
		}
		writeJSON(w, map[string]any{"keys": keys})
	})
	if enableConsistency {
		mux.HandleFunc("/api/v1/token/revocation/consistency", func(w http.ResponseWriter, r *http.Request) {
			proof, err := rc.GenerateConsistencyProof(0)
			if err != nil {
				writeJSON(w, map[string]any{"success": false})
				return
			}
			writeJSON(w, map[string]any{"success": true, "proof": proof, "latest_tree_head": map[string]any{"merkle_root": rc.LatestTreeHead().MerkleRoot}})
		})
	}
	return httptest.NewServer(mux)
}

// writeJSON is shared among tests.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	b, _ := json.Marshal(v)
	_, _ = w.Write(b)
}
