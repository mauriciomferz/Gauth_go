package main

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	cryptoReg "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/crypto"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/notary"
	auditor "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/auditor"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/delegation"
	poa "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/poa"
)

// AuditorResult is a generic result envelope printed as JSON.
type AuditorResult struct {
	Success   bool        `json:"success"`
	Mode      string      `json:"mode"`
	Detail    interface{} `json:"detail,omitempty"`
	Error     string      `json:"error,omitempty"`
	Reason    string      `json:"reason,omitempty"`
	LatencyMs int64       `json:"latency_ms"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func main() {
	baseURL := flag.String("base-url", "http://localhost:8080", "Base URL of running GAuth server")
	mode := flag.String("mode", "help", "Operation: rotation|rotation-v2-remote|rotation-v2-file|poa-verify|poa-file|attestation-file|attestation-remote|revocation|revocation-proof|revocation-consistency|help")
	rotV2Prev := flag.String("rotation-v2-prev", "", "Expected previous artifact digest for continuity check (rotation-v2 modes)")
	rotV2File := flag.String("rotation-v2-file", "", "Path to rotation V2 artifact JSON (rotation-v2-file mode)")
	poaID := flag.String("poa-id", "", "PoA identifier (for remote fetch modes)")
	poaFile := flag.String("poa-file", "", "Path to PoA JSON file (for poa-file mode)")
	attFile := flag.String("attestation-file", "", "Path to attestation JSON (for attestation-file mode)")
	attID := flag.String("attestation-id", "", "Attestation identifier (for attestation-remote mode if endpoint supports)")
	revID := flag.String("revocation-id", "", "Revocation event ID for proof verification mode")
	revIndex := flag.Int("revocation-index", -1, "Revocation event index (0-based) for proof verification mode")
	revHash := flag.String("revocation-hash", "", "Revocation event hash for proof verification mode (hex)")
	olderSize := flag.Int("older-size", -1, "Older tree size for revocation consistency (sizes stub mode)")
	newerSize := flag.Int("newer-size", -1, "Newer tree size for revocation consistency (sizes stub mode)")
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()

	start := time.Now()
	res := AuditorResult{Mode: *mode}

	var err error
	switch *mode {
	case "help":
		printHelp()
		return
	case "rotation":
		res.Detail, err = auditRotation(*baseURL)
	case "rotation-v2-remote":
		res.Detail, err = auditRotationV2Remote(*baseURL, *rotV2Prev)
	case "rotation-v2-file":
		if *rotV2File == "" {
			err = errors.New("rotation-v2-file required")
			break
		}
		res.Detail, err = auditRotationV2File(*rotV2File, *rotV2Prev)
	case "poa-verify":
		if *poaID == "" {
			err = errors.New("poa-id required")
			break
		}
		res.Detail, err = auditRemotePOA(*baseURL, *poaID)
	case "poa-file":
		if *poaFile == "" {
			err = errors.New("poa-file required")
			break
		}
		res.Detail, err = auditLocalPOA(*poaFile)
	case "attestation-file":
		if *attFile == "" {
			err = errors.New("attestation-file required")
			break
		}
		res.Detail, err = auditLocalAttestation(*attFile)
	case "attestation-remote":
		// Attempt remote retrieval (endpoint may vary; using placeholder path). If attestation-id empty try default path.
		res.Detail, err = auditRemoteAttestation(*baseURL, *attID)
	case "revocation":
		res.Detail, err = auditRevocation(*baseURL)
	case "revocation-proof":
		res.Detail, err = auditRevocationProof(*baseURL, *revID, *revIndex, *revHash)
	case "revocation-consistency":
		res.Detail, err = auditRevocationConsistency(*baseURL, *olderSize, *newerSize)
	default:
		err = fmt.Errorf("unknown mode: %s", *mode)
	}
	res.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		res.Success = false
		res.Error = "audit_failed"
		res.Reason = err.Error()
	} else {
		res.Success = true
	}
	enc, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(enc))
	if !res.Success {
		os.Exit(1)
	}
	if *verbose {
		fmt.Fprintf(os.Stderr, "completed in %dms\n", res.LatencyMs)
	}
}

func printHelp() {
	fmt.Print(`GAuth Auditor CLI

Usage:
	auditor --mode rotation --base-url http://localhost:8080
	auditor --mode poa-verify --base-url http://localhost:8080 --poa-id <id>
	auditor --mode poa-file --poa-file ./poa.json
	auditor --mode attestation-file --attestation-file ./att.json
	auditor --mode attestation-remote --base-url http://localhost:8080 [--attestation-id <id>]
	auditor --mode revocation --base-url http://localhost:8080
	auditor --mode revocation-proof --base-url http://localhost:8080 --revocation-index 0
	auditor --mode revocation-proof --base-url http://localhost:8080 --revocation-id <event_id>
	auditor --mode revocation-proof --base-url http://localhost:8080 --revocation-hash <event_hash>
	auditor --mode rotation-v2-remote --base-url http://localhost:8080 [--rotation-v2-prev <prev_digest>]
	auditor --mode rotation-v2-file --rotation-v2-file ./artifact.json [--rotation-v2-prev <prev_digest>]

Modes:
	rotation             Fetch rotation summary and verify signatures
	rotation-v2-remote   Fetch weighted rotation V2 artifact and verify digest + continuity + signatures (if embedded public keys)
	rotation-v2-file     Verify local rotation V2 artifact JSON (digest + continuity + signatures if embedded)
	poa-verify           Fetch PoA by id (placeholder endpoint assumption) and verify digest + signatures
	poa-file             Verify local PoA JSON file (digest + signatures)
	attestation-file     Verify local attestation JSON (signature + combined hash)
	attestation-remote   Fetch remote attestation and verify signature (replay checked server-side)
	revocation           Verify revocation chain head + consistency proof (basic external verification)
	revocation-proof     Fetch Merkle proof for a revocation event (by id|index|hash) and verify inclusion

Exit codes: 0 success, 1 failure.
`)
}

// auditRotation fetches rotation summary endpoint and verifies legacy + multi signatures.
func auditRotation(base string) (interface{}, error) {
	url := strings.TrimSuffix(base, "/") + "/api/v1/rotation/summary"
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	var sum notary.RotationSummary
	if err := json.Unmarshal(body, &sum); err != nil {
		return nil, fmt.Errorf("decode_rotation_summary: %w", err)
	}
	reg := cryptoReg.GlobalEdDSARegistry
	if reg == nil {
		return nil, errors.New("eddsa_registry_unavailable")
	}
	// Verify all signatures collected.
	validCount := 0
	reasons := []string{}
	payload, err := canonicalRotationSummaryPayload(&sum)
	if err != nil {
		return nil, fmt.Errorf("canonical_payload_error: %w", err)
	}
	msg := append([]byte("GAUTH_ROTATION_SUMMARY:"), payload...)
	for _, sig := range sum.Signatures {
		k := reg.FindByID(sig.Kid)
		if k == nil {
			reasons = append(reasons, "kid_not_found:"+sig.Kid)
			continue
		}
		bsig, err := base64.RawURLEncoding.DecodeString(sig.Signature)
		if err != nil {
			reasons = append(reasons, "signature_decode:"+sig.Kid)
			continue
		}
		if ed25519.Verify(k.Public, msg, bsig) {
			validCount++
		} else {
			reasons = append(reasons, "signature_invalid:"+sig.Kid)
		}
	}
	return map[string]interface{}{
		"chain_length":     sum.ChainLength,
		"head_hash":        sum.HeadHash,
		"aggregate_hash":   sum.AggregateHash,
		"generated_at":     sum.GeneratedAt,
		"signatures_total": len(sum.Signatures),
		"signatures_valid": validCount,
		"reasons":          reasons,
	}, nil
}

// Attestation structure expected (matches server verify endpoint schema).
type Attestation struct {
	Success    bool   `json:"success"`
	Configured bool   `json:"configured"`
	Reason     string `json:"reason,omitempty"`
	Nonce      string `json:"nonce,omitempty"`
	Snapshot   struct {
		Hash        string `json:"hash"`
		GeneratedAt string `json:"generated_at"`
	} `json:"snapshot"`
	Audit *struct {
		HeadHash string `json:"head_hash"`
		Entries  int    `json:"entries"`
	} `json:"audit,omitempty"`
	Anchor *struct {
		LatestHash string `json:"latest_hash"`
		Entries    int    `json:"entries"`
		Interval   int    `json:"interval"`
	} `json:"anchor,omitempty"`
	StrictUnknown bool `json:"strict_unknown"`
	Surge         *struct {
		ModelID   string  `json:"model_id"`
		Last10Sec int     `json:"last_10s_exceed_events"`
		AvgActive float64 `json:"avg_active_seconds"`
		Factor    float64 `json:"factor"`
		MinEvents int     `json:"min_events"`
		Triggered bool    `json:"triggered"`
		At        string  `json:"triggered_at,omitempty"`
	} `json:"surge,omitempty"`
	Notarization *struct {
		Provider       string  `json:"provider"`
		Timestamp      string  `json:"timestamp"`
		LatencySeconds float64 `json:"latency_seconds"`
		Success        bool    `json:"success"`
	} `json:"notarization,omitempty"`
	Signature string `json:"signature"`
	SigKid    string `json:"sig_kid"`
	SigMode   string `json:"sig_mode"`
}

var attestationSeen sync.Map // local replay detection across CLI session

// auditLocalAttestation reads attestation from file and verifies signature & combined hash.
func auditLocalAttestation(path string) (interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var att Attestation
	if err := json.Unmarshal(data, &att); err != nil {
		return nil, fmt.Errorf("decode_attestation: %w", err)
	}
	return verifyAttestation(&att), nil
}

// auditRemoteAttestation fetches attestation; endpoint placeholder (/api/v1/attestation or /api/v1/attestation/<id>).
func auditRemoteAttestation(base, id string) (interface{}, error) {
	path := "/api/v1/attestation"
	if id != "" {
		path = path + "/" + id
	}
	url := strings.TrimSuffix(base, "/") + path
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	var att Attestation
	if err := json.Unmarshal(body, &att); err != nil {
		return nil, fmt.Errorf("decode_attestation: %w", err)
	}
	return verifyAttestation(&att), nil
}

// verifyAttestation performs signature verification and computes combined hash.
func verifyAttestation(att *Attestation) map[string]interface{} {
	if att == nil {
		return map[string]interface{}{"valid": false, "reason": "nil_attestation"}
	}
	result := map[string]interface{}{"provided_sig_kid": att.SigKid}
	if att.Signature == "" || att.SigKid == "" || att.SigMode != "eddsa" {
		result["valid"] = false
		result["reason"] = "fields_missing"
		return result
	}
	reg := cryptoReg.GlobalEdDSARegistry
	if reg == nil {
		result["valid"] = false
		result["reason"] = "registry_unavailable"
		return result
	}
	k := reg.FindByID(att.SigKid)
	if k == nil {
		result["valid"] = false
		result["reason"] = "kid_not_found"
		return result
	}
	// unsigned view
	type unsignedStruct struct {
		Success    bool   `json:"success"`
		Configured bool   `json:"configured"`
		Reason     string `json:"reason,omitempty"`
		Nonce      string `json:"nonce,omitempty"`
		Snapshot   struct {
			Hash        string `json:"hash"`
			GeneratedAt string `json:"generated_at"`
		} `json:"snapshot"`
		Audit *struct {
			HeadHash string `json:"head_hash"`
			Entries  int    `json:"entries"`
		} `json:"audit,omitempty"`
		Anchor *struct {
			LatestHash string `json:"latest_hash"`
			Entries    int    `json:"entries"`
			Interval   int    `json:"interval"`
		} `json:"anchor,omitempty"`
		StrictUnknown bool `json:"strict_unknown"`
		Surge         *struct {
			ModelID   string  `json:"model_id"`
			Last10Sec int     `json:"last_10s_exceed_events"`
			AvgActive float64 `json:"avg_active_seconds"`
			Factor    float64 `json:"factor"`
			MinEvents int     `json:"min_events"`
			Triggered bool    `json:"triggered"`
			At        string  `json:"triggered_at,omitempty"`
		} `json:"surge,omitempty"`
		Notarization *struct {
			Provider       string  `json:"provider"`
			Timestamp      string  `json:"timestamp"`
			LatencySeconds float64 `json:"latency_seconds"`
			Success        bool    `json:"success"`
		} `json:"notarization,omitempty"`
	}
	u := unsignedStruct{Success: att.Success, Configured: att.Configured, Reason: att.Reason, Nonce: att.Nonce, Snapshot: att.Snapshot, Audit: att.Audit, Anchor: att.Anchor, StrictUnknown: att.StrictUnknown, Surge: att.Surge, Notarization: att.Notarization}
	raw, _ := json.Marshal(u)
	msg := append([]byte("GAUTH_MODEL_LIMIT_ATTEST:"), raw...)
	sigBytes, err := base64.RawStdEncoding.DecodeString(att.Signature)
	if err != nil {
		result["valid"] = false
		result["reason"] = "signature_decode"
		return result
	}
	valid := ed25519.Verify(k.Public, msg, sigBytes)
	result["signature_valid"] = valid
	// Local replay detection
	if att.Nonce != "" {
		if _, seen := attestationSeen.Load(att.Nonce); seen {
			result["replay"] = true
		} else {
			attestationSeen.Store(att.Nonce, struct{}{})
		}
	}
	// Combined hash triple
	auditHead := ""
	if att.Audit != nil {
		auditHead = att.Audit.HeadHash
	}
	anchorHead := ""
	if att.Anchor != nil {
		anchorHead = att.Anchor.LatestHash
	}
	seed := fmt.Sprintf("attest|%s|%s|%s", att.Snapshot.Hash, auditHead, anchorHead)
	ch := sha256.Sum256([]byte(seed))
	result["combined_hash"] = fmt.Sprintf("sha256:%x", ch[:])
	result["valid"] = valid
	return result
}

// auditRevocation fetches revocation head and a consistency proof from start=0 to head length-1 (if length>1).
// Performs minimal verification: ensures length is non-negative, head hash non-empty, and consistency proof returns success HTTP.
func auditRevocation(base string) (interface{}, error) {
	headURL := strings.TrimSuffix(base, "/") + "/api/v1/token/revocation/head"
	headBody, err := httpGet(headURL)
	if err != nil {
		return nil, fmt.Errorf("revocation_head_fetch: %w", err)
	}
	var headResp struct {
		Head      string `json:"head"`
		Aggregate string `json:"aggregate"`
		Length    int    `json:"length"`
		Verified  bool   `json:"verified"`
	}
	if err := json.Unmarshal(headBody, &headResp); err != nil {
		return nil, fmt.Errorf("revocation_head_decode: %w", err)
	}
	result := map[string]interface{}{
		"head":          headResp.Head,
		"aggregate":     headResp.Aggregate,
		"length":        headResp.Length,
		"verified_flag": headResp.Verified,
	}
	if headResp.Head == "" {
		result["error"] = "empty_head"
	}
	// Fetch consistency proof only if length > 1
	if headResp.Length > 1 {
		consURL := strings.TrimSuffix(base, "/") + "/api/v1/token/revocation/consistency?start=0"
		consBody, err := httpGet(consURL)
		if err != nil {
			result["consistency_error"] = err.Error()
		} else {
			// Store raw for now; deeper verification (Merkle path reconstruction) can be added later.
			result["consistency_raw"] = json.RawMessage(consBody)
		}
	} else {
		result["consistency_skipped"] = true
	}
	return result, nil
}

// auditRevocationProof fetches Merkle inclusion proof for a revocation event and verifies inclusion.
// Identifier precedence: id > index > hash.
func auditRevocationProof(base, id string, index int, hash string) (interface{}, error) {
	if id == "" && index < 0 && hash == "" {
		return nil, errors.New("revocation identifier required (id|index|hash)")
	}
	trim := strings.TrimSuffix(base, "/")
	var q string
	switch {
	case id != "":
		q = "?id=" + id
	case index >= 0:
		q = fmt.Sprintf("?index=%d", index)
	case hash != "":
		q = "?hash=" + hash
	}
	proofURL := trim + "/api/v1/token/revocation/proof" + q
	body, err := httpGet(proofURL)
	if err != nil {
		return nil, fmt.Errorf("revocation_proof_fetch: %w", err)
	}
	var pr struct {
		Success    bool                         `json:"success"`
		Target     string                       `json:"target"`
		MerkleRoot string                       `json:"merkle_root"`
		Proof      []delegation.MerkleProofStep `json:"proof"`
	}
	if err := json.Unmarshal(body, &pr); err != nil {
		return nil, fmt.Errorf("revocation_proof_decode: %w", err)
	}
	if !pr.Success {
		return nil, fmt.Errorf("revocation_proof_unsuccessful")
	}
	eventHash := hash
	if eventHash == "" { // need to fetch events listing to resolve ID or index to hash
		listURL := trim + "/api/v1/token/revocation/verify"
		lbody, lerr := httpGet(listURL)
		if lerr != nil {
			return nil, fmt.Errorf("revocation_events_fetch: %w", lerr)
		}
		var evResp struct {
			Success bool `json:"success"`
			Events  []struct {
				ID    string `json:"id"`
				Hash  string `json:"hash"`
				Index int    `json:"index"`
			} `json:"events"`
		}
		if err := json.Unmarshal(lbody, &evResp); err != nil {
			return nil, fmt.Errorf("revocation_events_decode: %w", err)
		}
		found := false
		for _, e := range evResp.Events {
			if id != "" && e.ID == id {
				eventHash = e.Hash
				found = true
				break
			}
			if id == "" && index >= 0 && e.Index == index {
				eventHash = e.Hash
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("revocation_event_not_found")
		}
	}
	verRes := auditor.VerifyRevocationProof(eventHash, pr.Proof, pr.MerkleRoot)
	return map[string]interface{}{
		"target":      pr.Target,
		"merkle_root": pr.MerkleRoot,
		"event_hash":  eventHash,
		"included":    verRes.Included,
		"steps":       verRes.Steps,
		"reason":      verRes.Reason,
		"leaf_digest": verRes.LeafDigest,
	}, nil
}

// auditRevocationConsistency queries the sizes-based consistency stub endpoint.
// Treat 501 proof_unavailable as non-fatal (feature incubation) returning structured detail.
func auditRevocationConsistency(base string, older, newer int) (interface{}, error) {
	if older < 0 || newer < 0 || older > newer {
		return nil, errors.New("invalid sizes (older/newer)")
	}
	url := fmt.Sprintf("%s/api/v1/token/revocation/consistency_sizes?older=%d&newer=%d", strings.TrimSuffix(base, "/"), older, newer)
	body, err := httpGet(url)
	if err != nil {
		// If server returned 501 we still want to parse body for taxonomy; attempt extraction.
		// httpGet returns error with status; propagate.
		return nil, err
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode_consistency_sizes: %w", err)
	}
	// Success path includes proof.trivial when older==newer==current_length.
	return resp, nil
}

// auditRemotePOA fetches PoA by id and verifies digest + signatures. Endpoint assumed /api/v1/poa/<id> (placeholder).
func auditRemotePOA(base, id string) (interface{}, error) {
	url := strings.TrimSuffix(base, "/") + "/api/v1/poa/" + id
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	var doc poa.ProofOfAuthorization
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("decode_poa: %w", err)
	}
	return verifyPOA(&doc), nil
}

// auditLocalPOA reads PoA from file and verifies.
func auditLocalPOA(path string) (interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc poa.ProofOfAuthorization
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("decode_poa: %w", err)
	}
	return verifyPOA(&doc), nil
}

// verifyPOA performs digest and signature threshold verification.
func verifyPOA(doc *poa.ProofOfAuthorization) map[string]interface{} { return auditor.VerifyPOA(doc) }

// httpGet helper.
func httpGet(url string) ([]byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("http_%d: %s", resp.StatusCode, string(b))
	}
	return io.ReadAll(resp.Body)
}

// canonicalRotationSummaryPayload duplicates internal function for auditor without exporting.
func canonicalRotationSummaryPayload(sum *notary.RotationSummary) ([]byte, error) {
	if sum == nil {
		return nil, errors.New("nil_summary")
	}
	payload := struct {
		ChainLength   int    `json:"chain_length"`
		HeadHash      string `json:"head_hash"`
		AggregateHash string `json:"aggregate_hash"`
		GeneratedAt   string `json:"generated_at"`
	}{ChainLength: sum.ChainLength, HeadHash: sum.HeadHash, AggregateHash: sum.AggregateHash, GeneratedAt: sum.GeneratedAt}
	return json.Marshal(payload)
}

// auditRotationV2Remote fetches V2 endpoint and audits artifact.
func auditRotationV2Remote(base, expectedPrev string) (interface{}, error) {
	url := strings.TrimSuffix(base, "/") + "/api/v1/rotation/summary/v2"
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	return auditRotationV2ArtifactJSON(body, expectedPrev)
}

// auditRotationV2File reads artifact JSON from disk and audits.
func auditRotationV2File(path, expectedPrev string) (interface{}, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return auditRotationV2ArtifactJSON(b, expectedPrev)
}

// auditRotationV2ArtifactJSON performs digest recomputation and continuity check.
//
//nolint:gocyclo // Audit verification with multiple artifact types
func auditRotationV2ArtifactJSON(data []byte, expectedPrev string) (interface{}, error) {
	// Local signer type matching JSON schema (avoid anonymous struct duplication mismatch)
	type signerEntry struct {
		Signer    string `json:"signer"`
		Alg       string `json:"alg"`
		Weight    int    `json:"weight"`
		Signature string `json:"signature"`
		Public    string `json:"public,omitempty"`
	}
	var outer struct {
		Success  bool `json:"success"`
		Artifact struct {
			ThresholdWeight      int           `json:"threshold_weight"`
			Signers              []signerEntry `json:"signers"`
			ActiveKeySetID       string        `json:"active_key_set_id"`
			PreviousArtifactHash string        `json:"previous_artifact_hash"`
			AlgorithmSuite       []string      `json:"algorithm_suite"`
			CanonicalDigest      string        `json:"canonical_digest"`
			GeneratedAt          string        `json:"generated_at"`
		} `json:"artifact"`
	}
	if err := json.Unmarshal(data, &outer); err != nil {
		return nil, fmt.Errorf("decode_rotation_v2: %w", err)
	}
	art := outer.Artifact
	// Recompute canonical digest (excluding signatures & public keys) matching server logic
	h := sha256.New()
	fmt.Fprintf(h, "%d|%s|%s|%d|", 2, art.ActiveKeySetID, art.PreviousArtifactHash, art.ThresholdWeight)
	algCopy := append([]string(nil), art.AlgorithmSuite...)
	sort.Strings(algCopy)
	for i, a := range algCopy {
		if i > 0 {
			h.Write([]byte(","))
		}
		h.Write([]byte(a))
	}
	h.Write([]byte("|"))
	// Signers sorted by ID for digest recompute (copy into auxiliary slice for sorting)
	aux := make([]signerEntry, len(art.Signers))
	copy(aux, art.Signers)
	s := aux
	sort.Slice(s, func(i, j int) bool { return s[i].Signer < s[j].Signer })
	for _, si := range s {
		fmt.Fprintf(h, "%s|%s|%d\n", si.Signer, si.Alg, si.Weight)
	}
	computed := fmt.Sprintf("sha256:%x", h.Sum(nil))
	continuityOK := true
	if expectedPrev != "" && art.PreviousArtifactHash != expectedPrev {
		continuityOK = false
	}

	// Signature verification (only algorithms with embedded public keys). Preimage domain separation.
	preimage := []byte("GAUTH_ROTATION_V2:" + computed)
	verifiedWeight := 0
	failures := []string{}
	validSigners := []string{}
	for _, si := range s {
		if si.Signature == "" {
			continue
		}
		algUpper := strings.ToUpper(si.Alg)
		sigBytes, err := base64.RawURLEncoding.DecodeString(si.Signature)
		if err != nil {
			failures = append(failures, "signature_decode:"+si.Signer)
			continue
		}
		switch algUpper {
		case "ED25519":
			if si.Public == "" {
				failures = append(failures, "public_missing:"+si.Signer)
				continue
			}
			pubBytes, err := base64.RawURLEncoding.DecodeString(si.Public)
			if err != nil || len(pubBytes) != ed25519.PublicKeySize {
				failures = append(failures, "public_decode:"+si.Signer)
				continue
			}
			if len(sigBytes) != ed25519.SignatureSize || !ed25519.Verify(ed25519.PublicKey(pubBytes), preimage, sigBytes) {
				failures = append(failures, "signature_invalid:"+si.Signer)
				continue
			}
			verifiedWeight += si.Weight
			validSigners = append(validSigners, si.Signer)
		case "ECDSA-P256", "ECDSA_P256", "ECDSA-P256-SHA256":
			if si.Public == "" {
				failures = append(failures, "public_missing:"+si.Signer)
				continue
			}
			pubBytes, err := base64.RawURLEncoding.DecodeString(si.Public)
			if err != nil || len(pubBytes) != 65 || pubBytes[0] != 0x04 {
				failures = append(failures, "public_decode:"+si.Signer)
				continue
			}
			x := new(big.Int).SetBytes(pubBytes[1:33])
			y := new(big.Int).SetBytes(pubBytes[33:65])
			curve := elliptic.P256()
			if !curve.IsOnCurve(x, y) {
				failures = append(failures, "public_invalid_point:"+si.Signer)
				continue
			}
			var rs struct{ R, S *big.Int }
			if _, err := asn1.Unmarshal(sigBytes, &rs); err != nil || rs.R == nil || rs.S == nil {
				failures = append(failures, "signature_decode:"+si.Signer)
				continue
			}
			h := sha256.Sum256(preimage)
			pk := ecdsa.PublicKey{Curve: curve, X: x, Y: y}
			if !ecdsa.Verify(&pk, h[:], rs.R, rs.S) {
				failures = append(failures, "signature_invalid:"+si.Signer)
				continue
			}
			verifiedWeight += si.Weight
			validSigners = append(validSigners, si.Signer)
		default:
			if si.Public != "" {
				failures = append(failures, "alg_unsupported:"+algUpper+":"+si.Signer)
			}
		}
	}
	thresholdMet := verifiedWeight >= art.ThresholdWeight && art.ThresholdWeight > 0
	return map[string]interface{}{
		"digest_match":             computed == art.CanonicalDigest,
		"computed_digest":          computed,
		"artifact_digest":          art.CanonicalDigest,
		"threshold_weight":         art.ThresholdWeight,
		"signer_count":             len(art.Signers),
		"continuity_expected_prev": expectedPrev,
		"continuity_prev":          art.PreviousArtifactHash,
		"continuity_ok":            continuityOK,
		"verified_weight":          verifiedWeight,
		"threshold_met":            thresholdMet,
		"signature_failures":       failures,
		"signatures_valid":         len(validSigners),
		"valid_signers":            validSigners,
		"algorithms":               algCopy,
	}, nil
}
