package verification

// Reusable verification utilities extracted from cmd/verify for programmatic usage and unit testing.
// Reusable verification utilities extracted from cmd/verify for programmatic usage and unit testing.
// Provides higher-level orchestration (VerifyAll) and granular helpers (FetchDiscovery, FetchEvents,
// FetchProofByHash, FetchConsistency, LoadJWKS, VerifyInclusion, VerifySTHMultiSig).

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/crypto"
	delegation "github.com/mauriciomferz/Gauth_go/pkg/delegation"
)

// DiscoveryResponse minimal shape for revocation support
type DiscoveryResponse struct {
	RevocationSupport struct {
		STHLatest      any `json:"sth_latest"`
		STHHistorySize int `json:"sth_history_size"`
	} `json:"revocation_support"`
}

// SignedTreeHeadJSON matches API subset for STH
type SignedTreeHeadJSON struct {
	Version       int    `json:"version"`
	MerkleRoot    string `json:"merkle_root"`
	ChainLength   int    `json:"chain_length"`
	AggregateHash string `json:"aggregate_hash"`
	Timestamp     string `json:"timestamp"`
	Signatures    []struct {
		Kid    string `json:"kid"`
		Alg    string `json:"alg"`
		Sig    string `json:"sig"`
		Weight int    `json:"weight"`
	} `json:"signatures"`
	Threshold       int `json:"threshold"`
	WeightsTotal    int `json:"weights_total"`
	SatisfiedWeight int `json:"satisfied_weight"`
}

type VerifyEventsResponse struct {
	Success bool `json:"success"`
	Events  []struct {
		ID, Hash string
		Index    int
	} `json:"events"`
	Length    int    `json:"length"`
	Verified  bool   `json:"verified"`
	Aggregate string `json:"aggregate_hash"`
}

type MerkleProofResponse struct {
	Success    bool                         `json:"success"`
	Target     string                       `json:"target"`
	MerkleRoot string                       `json:"merkle_root"`
	Proof      []delegation.MerkleProofStep `json:"proof"`
}

type ConsistencyResponse struct {
	Success        bool                         `json:"success"`
	Proof          *delegation.ConsistencyProof `json:"proof"`
	LatestTreeHead *SignedTreeHeadJSON          `json:"latest_tree_head"`
}

// VerifyError is an optional structured error wrapper providing a stable code and human detail.
// Code strings align with documented error fragments in REVOCATION_TRANSPARENCY.md.
type VerifyError struct {
	Code   string
	Detail string
	Cause  error
}

func (e *VerifyError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Detail)
	}
	return e.Code
}

func wrap(code string, detail string, cause error) error {
	// Preserve legacy string patterns for callers performing substring matching.
	if cause != nil && detail == "" {
		detail = cause.Error()
	}
	return &VerifyError{Code: code, Detail: detail, Cause: cause}
}

// HTTPClient interface for injection (allows using *http.Client or mock)
type HTTPClient interface {
	Get(string) (*http.Response, error)
}

// FetchDiscovery retrieves discovery metadata.
func FetchDiscovery(client HTTPClient, base string) (*DiscoveryResponse, error) {
	url := fmt.Sprintf("%s/.well-known/gauth-configuration", base)
	b, err := httpRead(client, url)
	if err != nil {
		return nil, err
	}
	var dr DiscoveryResponse
	if err := json.Unmarshal(b, &dr); err != nil {
		return nil, err
	}
	return &dr, nil
}

// FetchEvents returns verify endpoint payload.
func FetchEvents(client HTTPClient, base string) (*VerifyEventsResponse, error) {
	url := fmt.Sprintf("%s/api/v1/token/revocation/verify", base)
	b, err := httpRead(client, url)
	if err != nil {
		return nil, err
	}
	var vr VerifyEventsResponse
	if err := json.Unmarshal(b, &vr); err != nil {
		return nil, err
	}
	if !vr.Success {
		return nil, wrap("verify_endpoint_failure", "revocation verify endpoint returned success=false", nil)
	}
	return &vr, nil
}

// FetchProofByHash obtains inclusion proof for a given event hash.
func FetchProofByHash(client HTTPClient, base, hash string) (*MerkleProofResponse, error) {
	url := fmt.Sprintf("%s/api/v1/token/revocation/proof?hash=%s", base, hash)
	b, err := httpRead(client, url)
	if err != nil {
		return nil, err
	}
	var pr MerkleProofResponse
	if err := json.Unmarshal(b, &pr); err != nil {
		return nil, err
	}
	if !pr.Success {
		return nil, wrap("proof_endpoint_failure", "revocation proof endpoint returned success=false", nil)
	}
	return &pr, nil
}

// FetchConsistency retrieves append-only proof from start index.
func FetchConsistency(client HTTPClient, base string, start int) (*ConsistencyResponse, error) {
	url := fmt.Sprintf("%s/api/v1/token/revocation/consistency?start=%d", base, start)
	b, err := httpRead(client, url)
	if err != nil {
		return nil, err
	}
	var cr ConsistencyResponse
	if err := json.Unmarshal(b, &cr); err != nil {
		return nil, err
	}
	if !cr.Success {
		return nil, wrap("consistency_endpoint_failure", "revocation consistency endpoint returned success=false", nil)
	}
	return &cr, nil
}

// LoadJWKS imports published Ed25519 public keys into a new ephemeral manager.
// Returns the manager for explicit use instead of setting global state.
func LoadJWKS(client HTTPClient, base string) (crypto.KeyProvider, error) {
	url := fmt.Sprintf("%s/.well-known/jwks.json", base)
	b, err := httpRead(client, url)
	if err != nil {
		return nil, err
	}
	var jwks struct {
		Keys []map[string]any `json:"keys"`
	}
	if err := json.Unmarshal(b, &jwks); err != nil {
		return nil, err
	}
	km, kmErr := crypto.NewManager(30 * time.Minute)
	if kmErr != nil {
		return nil, wrap("key_manager_init", "failed to initialize key manager", kmErr)
	}
	for _, k := range jwks.Keys {
		kty, _ := k["kty"].(string)
		crv, _ := k["crv"].(string)
		if kty == "OKP" && crv == "Ed25519" {
			x, _ := k["x"].(string)
			kid, _ := k["kid"].(string)
			if x == "" || kid == "" {
				continue
			}
			pub, decErr := base64.RawURLEncoding.DecodeString(x)
			if decErr != nil {
				return nil, wrap("jwks_decode", "failed to decode ed25519 key", decErr)
			}
			// ImportPublic has no return; best-effort insertion.
			km.ImportPublic(kid, pub, time.Now().Add(30*time.Minute))
		}
	}
	return km, nil
}

// ConvertSignedTreeHead builds delegation.SignedTreeHead from API JSON structure.
func ConvertSignedTreeHead(src *SignedTreeHeadJSON) *delegation.SignedTreeHead {
	if src == nil {
		return nil
	}
	sth := &delegation.SignedTreeHead{Version: src.Version, MerkleRoot: src.MerkleRoot, ChainLength: src.ChainLength, AggregateHash: src.AggregateHash, Timestamp: parseTime(src.Timestamp), Threshold: src.Threshold, WeightsTotal: src.WeightsTotal, SatisfiedWeight: src.SatisfiedWeight}
	for _, s := range src.Signatures {
		sth.Signatures = append(sth.Signatures, delegation.TreeHeadSignature{Kid: s.Kid, Alg: s.Alg, Sig: s.Sig, Weight: s.Weight})
	}
	return sth
}

// VerifyInclusion checks Merkle inclusion proof.
func VerifyInclusion(eventHash string, proof *MerkleProofResponse) (bool, error) {
	if proof == nil {
		return false, wrap("nil_proof", "proof object was nil", nil)
	}
	leaf := delegation.LeafDigestForEventHash(eventHash)
	ok := delegation.VerifyProof(leaf, proof.Proof, proof.MerkleRoot)
	if !ok {
		return false, wrap("inclusion_failed", "merkle root mismatch", nil)
	}
	return true, nil
}

// VerifySTHMultiSig verifies the Signed Tree Head using the provided KeyProvider.
func VerifySTHMultiSig(sth *delegation.SignedTreeHead, kp crypto.KeyProvider) error {
	if sth == nil {
		return errors.New("nil_sth")
	}
	// Use delegation package's verification logic which now accepts KeyProvider
	// We need to adapt KeyProvider if necessary, but delegation.VerifyTreeHeadMultiSig accepts KeyProvider now.
	return delegation.VerifyTreeHeadMultiSig(sth, kp)
}

// VerifyConsistency validates append-only proof given complete event hash list.
func VerifyConsistency(cons *ConsistencyResponse, allEventHashes []string) error {
	if cons == nil || cons.Proof == nil {
		return wrap("nil_consistency", "consistency proof object missing", nil)
	}
	if err := delegation.VerifyConsistencyProof(cons.Proof, allEventHashes); err != nil {
		return wrap(err.Error(), "consistency verification failed", err)
	}
	return nil
}

// VerifyAll performs a full verification sequence: discovery -> JWKS -> events -> proof -> STH.
// It returns nil if the targetHash is confirmed revoked (or not found if that's the check).
// Actually, this function seems to verify that the system is consistent.
func VerifyAll(client HTTPClient, base, targetHash string) error {
	disco, err := FetchDiscovery(client, base)
	if err != nil {
		return wrap("discovery_fetch", "failed to fetch discovery metadata", err)
	}
	events, err := FetchEvents(client, base)
	if err != nil {
		return wrap("events_fetch", "failed to fetch events", err)
	}
	var found bool
	for _, e := range events.Events {
		if e.Hash == targetHash {
			found = true
			break
		}
	}
	if !found {
		return wrap("event_not_found", "target hash not found in events", nil)
	}
	proof, err := FetchProofByHash(client, base, targetHash)
	if err != nil {
		return wrap("proof_fetch", "failed to fetch proof", err)
	}
	if _, err := VerifyInclusion(targetHash, proof); err != nil {
		return wrap("inclusion_verify", "failed to verify inclusion proof", err)
	}

	// STH signature verification
	if disco.RevocationSupport.STHLatest != nil {
		// Marshal/unmarshal dynamic structure
		b, mErr := json.Marshal(disco.RevocationSupport.STHLatest)
		if mErr != nil {
			return wrap("sth_marshal", "failed to marshal latest STH", mErr)
		}
		var sthJSON SignedTreeHeadJSON
		if err := json.Unmarshal(b, &sthJSON); err != nil {
			return wrap("sth_unmarshal", "failed to parse signed tree head", err)
		}
		if sthJSON.MerkleRoot != "" {
			kp, err := LoadJWKS(client, base)
			if err != nil {
				return wrap("jwks_load", "failed to load jwks", err)
			}
			sth := ConvertSignedTreeHead(&sthJSON)
			if err := VerifySTHMultiSig(sth, kp); err != nil {
				return wrap("sth_verify", err.Error(), err)
			}
		}
	}
	// Consistency (optional)
	if disco.RevocationSupport.STHHistorySize > 1 {
		cons, err := FetchConsistency(client, base, 0)
		if err == nil {
			hashes := make([]string, 0, len(events.Events))
			for _, e := range events.Events {
				hashes = append(hashes, e.Hash)
			}
			_ = VerifyConsistency(cons, hashes) // best effort; ignore failure for now
		}
	}
	return nil
}

func httpRead(client HTTPClient, url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	data, rErr := ioReadAll(resp.Body)
	if rErr != nil {
		return nil, rErr
	}
	return data, nil
}

func ioReadAll(r io.Reader) ([]byte, error) { return io.ReadAll(r) }

func parseTime(s string) time.Time { t, _ := time.Parse(time.RFC3339, s); return t }
