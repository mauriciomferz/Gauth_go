package notary

// Weighted multi-signature rotation artifact (V2 placeholder implementation).
// This file provides data structures and deterministic canonical digest computation
// for the forthcoming weighted multi-sig rotation summary described in ADR_WEIGHTED_MULTISIG.
// Signing & verification are intentionally minimal (Ed25519 only) and will be expanded.

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"sort"
	"strings"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// WeightedRotationSigner models a single signer entry (weight + optional signature).
type WeightedRotationSigner struct {
	ID        string `json:"id"`
	Alg       string `json:"alg"`
	Weight    int    `json:"weight"`
	Signature string `json:"signature,omitempty"`
	Public    string `json:"public,omitempty"` // base64url encoded public key bytes (optional, excluded from canonical digest)
}

// WeightedRotationArtifact represents version 2 rotation artifact supporting weighted multi-sig.
// CanonicalDigest is computed over deterministic ordering of non-signature fields plus signer metadata
// (excluding signature bytes to allow independent signature attachment). See ADR for ordering rules.
type WeightedRotationArtifact struct {
	Version              int                      `json:"version"`
	ActiveKeySetID       string                   `json:"active_key_set_id"`
	PreviousArtifactHash string                   `json:"previous_artifact_hash,omitempty"`
	ThresholdWeight      int                      `json:"threshold_weight"`
	Signers              []WeightedRotationSigner `json:"signers"`
	AlgorithmSuite       []string                 `json:"algorithm_suite"`
	CanonicalDigest      string                   `json:"canonical_digest"`
	GeneratedAt          string                   `json:"generated_at"`
}

// PublicKeyRecord can carry multiple algorithm public key forms. Only one relevant per signer.
type PublicKeyRecord struct {
	Ed25519 ed25519.PublicKey
	ECDSA   *ecdsa.PublicKey
}

// --- Metrics (exported gauges) ---
var (
	rotationV2ThresholdWeight        = promauto.NewGauge(prom.GaugeOpts{Name: "gauth_rotation_v2_threshold_weight", Help: "Configured threshold weight for rotation V2 artifact."})
	rotationV2EffectiveWeight        = promauto.NewGauge(prom.GaugeOpts{Name: "gauth_rotation_v2_effective_weight", Help: "Sum of verified signer weights for rotation V2 artifact (placeholder: sum of configured weights)."})
	rotationV2SignerWeight           = promauto.NewGaugeVec(prom.GaugeOpts{Name: "gauth_rotation_v2_signer_weight", Help: "Configured weight per signer id for rotation V2."}, []string{"signer"})
	rotationV2VerifiedWeight         = promauto.NewGauge(prom.GaugeOpts{Name: "gauth_rotation_v2_verified_weight", Help: "Sum of weights from signatures that verified for latest served artifact."})
	rotationV2SignatureFailures      = promauto.NewCounterVec(prom.CounterOpts{Name: "gauth_rotation_v2_signature_failures_total", Help: "Total V2 signature verification failures by reason."}, []string{"reason"})
	rotationV2ThresholdViolations    = promauto.NewCounter(prom.CounterOpts{Name: "gauth_rotation_v2_threshold_violations_total", Help: "Total times V2 served artifact failed to meet threshold (pre-enforcement)."})
	rotationV2VerifiedWeightByAlg    = promauto.NewGaugeVec(prom.GaugeOpts{Name: "gauth_rotation_v2_verified_weight_alg", Help: "Verified weight contributed per algorithm."}, []string{"alg"})
	rotationV2SignatureFailuresByAlg = promauto.NewCounterVec(prom.CounterOpts{Name: "gauth_rotation_v2_signature_failures_by_alg_total", Help: "Signature verification failures partitioned by algorithm and reason."}, []string{"alg", "reason"})
	RotationV2ChainStarts            = promauto.NewCounter(prom.CounterOpts{Name: "gauth_rotation_v2_chain_starts_total", Help: "Number of times a rotation V2 chain started (empty previous hash)."})
	RotationV2ContinuityUpdates      = promauto.NewCounter(prom.CounterOpts{Name: "gauth_rotation_v2_continuity_updates_total", Help: "Number of continuity updates (previous hash advanced)."})
	rotationV2PublicKeysEmbedded     = promauto.NewCounter(prom.CounterOpts{Name: "gauth_rotation_v2_public_keys_embedded_total", Help: "Total artifacts where public key embedding flag enabled."})
	rotationV2EmbeddedKeyCount       = promauto.NewGauge(prom.GaugeOpts{Name: "gauth_rotation_v2_embedded_public_key_count", Help: "Count of embedded public keys in latest artifact."})
)

// cryptoGlobalEdDSAResolve removed in favor of injected resolver.

// EncodeECDSAP256Uncompressed encodes an ECDSA P-256 public key into base64url of the uncompressed
// point (0x04 || X || Y) with X and Y left-padded to 32 bytes. Returns empty string on invalid input.
func EncodeECDSAP256Uncompressed(pk *ecdsa.PublicKey) string {
	if pk == nil || pk.Curve != elliptic.P256() || pk.X == nil || pk.Y == nil {
		return ""
	}
	xb := pk.X.Bytes()
	yb := pk.Y.Bytes()
	if len(xb) > 32 || len(yb) > 32 {
		return ""
	}
	xpad := make([]byte, 32)
	copy(xpad[32-len(xb):], xb)
	ypad := make([]byte, 32)
	copy(ypad[32-len(yb):], yb)
	uncompressed := append([]byte{0x04}, append(xpad, ypad...)...)
	return base64.RawURLEncoding.EncodeToString(uncompressed)
}

// BuildWeightedRotationArtifact constructs an artifact without signatures and computes canonical digest.
// Signers slice may contain entries with Signature empty; weights must be non-negative.
func BuildWeightedRotationArtifact(activeSetID, prevHash string, threshold int, signers []WeightedRotationSigner, algSuite []string, now time.Time) (WeightedRotationArtifact, error) {
	for _, s := range signers {
		if s.Weight < 0 {
			return WeightedRotationArtifact{}, fmt.Errorf("negative weight for signer %s", s.ID)
		}
	}
	// Defensive copy & deterministic ordering of signers by ID ascending.
	ordered := make([]WeightedRotationSigner, len(signers))
	copy(ordered, signers)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	// Deterministic ordering of algorithm suite.
	algs := make([]string, len(algSuite))
	copy(algs, algSuite)
	sort.Strings(algs)
	art := WeightedRotationArtifact{
		Version:              2,
		ActiveKeySetID:       activeSetID,
		PreviousArtifactHash: prevHash,
		ThresholdWeight:      threshold,
		Signers:              ordered,
		AlgorithmSuite:       algs,
		GeneratedAt:          now.UTC().Format(time.RFC3339Nano),
	}
	art.CanonicalDigest = computeWeightedRotationDigest(art)
	return art, nil
}

// computeWeightedRotationDigest returns sha256 hex digest over canonical preimage ordering:
// version|active_key_set_id|previous_artifact_hash|threshold_weight|alg_suite(csv)|signer_lines...\n
// signer_line = id|alg|weight (signatures excluded to allow attachment without changing digest)
func computeWeightedRotationDigest(art WeightedRotationArtifact) string {
	h := sha256.New()
	fmt.Fprintf(h, "%d|%s|%s|%d|", art.Version, art.ActiveKeySetID, art.PreviousArtifactHash, art.ThresholdWeight)
	// Algorithm suite joined by comma (already sorted)
	for i, alg := range art.AlgorithmSuite {
		if i > 0 {
			h.Write([]byte(","))
		}
		h.Write([]byte(alg))
	}
	h.Write([]byte("|"))
	for _, s := range art.Signers {
		fmt.Fprintf(h, "%s|%s|%d\n", s.ID, s.Alg, s.Weight)
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}

// ComputeDebugDigestForTests is exported ONLY for tests (not part of public API) to allow
// recomputation of canonical digest after mutating non-digest fields (e.g., embedding public keys).
// It mirrors computeWeightedRotationDigest logic. Use cautiously outside tests.
func ComputeDebugDigestForTests(art *WeightedRotationArtifact) string {
	if art == nil {
		return ""
	}
	return computeWeightedRotationDigest(*art)
}

// AttachEd25519Signature signs the artifact canonical digest preimage (domain separated) and appends signature.
// This is a placeholder; future versions may sign full JSON canonical bytes excluding signature fields.
func AttachEd25519Signature(art *WeightedRotationArtifact, priv ed25519.PrivateKey, signerID string, alg string, weight int) error {
	if art == nil {
		return fmt.Errorf("artifact nil")
	}
	if len(priv) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid ed25519 private key size")
	}
	// Ensure signer entry presence (weight checked earlier); add if missing.
	found := false
	for i := range art.Signers {
		if art.Signers[i].ID == signerID {
			found = true
			if art.Signers[i].Weight != weight {
				return fmt.Errorf("weight mismatch for existing signer")
			}
			if art.Signers[i].Alg != alg {
				return fmt.Errorf("alg mismatch for existing signer")
			}
			// proceed to signature assignment
			break
		}
	}
	if !found {
		art.Signers = append(art.Signers, WeightedRotationSigner{ID: signerID, Alg: alg, Weight: weight})
		// Re-sort signers after append and recompute digest;
		sort.Slice(art.Signers, func(i, j int) bool { return art.Signers[i].ID < art.Signers[j].ID })
		art.CanonicalDigest = computeWeightedRotationDigest(*art)
	}
	// Domain-separated signing over canonical digest string bytes.
	preimage := []byte("GAUTH_ROTATION_V2:" + art.CanonicalDigest)
	sig := ed25519.Sign(priv, preimage)
	// Assign (post sort find index)
	for i := range art.Signers {
		if art.Signers[i].ID == signerID {
			art.Signers[i].Signature = base64.RawURLEncoding.EncodeToString(sig)
			break
		}
	}
	return nil
}

// AttachECDSASignature attaches an ECDSA P-256 signature encoded as base64url of ASN.1 DER (r,s) over the same domain separated preimage.
func AttachECDSASignature(art *WeightedRotationArtifact, priv *ecdsa.PrivateKey, signerID string, alg string, weight int) error {
	if art == nil {
		return fmt.Errorf("artifact nil")
	}
	if priv == nil || priv.Curve != elliptic.P256() {
		return fmt.Errorf("invalid ecdsa private key")
	}
	found := false
	for i := range art.Signers {
		if art.Signers[i].ID == signerID {
			found = true
			if art.Signers[i].Weight != weight {
				return fmt.Errorf("weight mismatch for existing signer")
			}
			if !strings.EqualFold(art.Signers[i].Alg, alg) {
				return fmt.Errorf("alg mismatch for existing signer")
			}
			break
		}
	}
	if !found {
		art.Signers = append(art.Signers, WeightedRotationSigner{ID: signerID, Alg: alg, Weight: weight})
		sort.Slice(art.Signers, func(i, j int) bool { return art.Signers[i].ID < art.Signers[j].ID })
		art.CanonicalDigest = computeWeightedRotationDigest(*art)
	}
	preimage := []byte("GAUTH_ROTATION_V2:" + art.CanonicalDigest)
	h := sha256.Sum256(preimage)
	r, s, err := ecdsa.Sign(rand.Reader, priv, h[:])
	if err != nil {
		return err
	}
	// DER encode
	der, err := asn1.Marshal(struct{ R, S *big.Int }{r, s})
	if err != nil {
		return err
	}
	for i := range art.Signers {
		if art.Signers[i].ID == signerID {
			art.Signers[i].Signature = base64.RawURLEncoding.EncodeToString(der)
			break
		}
	}
	return nil
}

// MarshalCanonicalJSON produces stable JSON bytes suitable for hashing if future design switches to JSON-based digests.
// Signers slice must already be deterministically ordered.
func MarshalCanonicalJSON(art *WeightedRotationArtifact) ([]byte, error) {
	if art == nil {
		return nil, fmt.Errorf("artifact nil")
	}
	// We rely on standard library deterministic map order for struct (Go 1.21+ maintains field order); signers already sorted.
	// Provide minimal wrapper omitting canonical_digest (recomputable) to avoid circular dependency.
	tmp := struct {
		Version              int                      `json:"version"`
		ActiveKeySetID       string                   `json:"active_key_set_id"`
		PreviousArtifactHash string                   `json:"previous_artifact_hash,omitempty"`
		ThresholdWeight      int                      `json:"threshold_weight"`
		Signers              []WeightedRotationSigner `json:"signers"`
		AlgorithmSuite       []string                 `json:"algorithm_suite"`
		GeneratedAt          string                   `json:"generated_at"`
	}{art.Version, art.ActiveKeySetID, art.PreviousArtifactHash, art.ThresholdWeight, art.Signers, art.AlgorithmSuite, art.GeneratedAt}
	return json.Marshal(tmp)
}

// ==== Config Loading ====

// WeightsConfig on-disk JSON structure.
type WeightsConfig struct {
	SchemaVersion   int    `json:"schema_version"`
	ActiveKeySetID  string `json:"active_key_set_id"`
	ThresholdWeight int    `json:"threshold_weight"`
	// Signers: alg is optional for backward compatibility; default assumed "ED25519" if empty.
	Signers []struct {
		ID     string `json:"id"`
		Alg    string `json:"alg,omitempty"`
		Weight int    `json:"weight"`
	} `json:"signers"`
	AlgorithmSuite []string `json:"algorithm_suite"`
}

// LoadWeightsConfig reads and validates multisig weight configuration.
func LoadWeightsConfig(path string) (*WeightsConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg WeightsConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	if cfg.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported schema_version %d", cfg.SchemaVersion)
	}
	if cfg.ThresholdWeight <= 0 {
		return nil, errors.New("threshold_weight must be > 0")
	}
	if len(cfg.Signers) == 0 {
		return nil, errors.New("signers empty")
	}
	// Validate uniqueness & weights
	seen := map[string]struct{}{}
	total := 0
	for i, s := range cfg.Signers {
		if s.ID == "" {
			return nil, errors.New("signer id empty")
		}
		if s.Weight <= 0 {
			return nil, fmt.Errorf("signer %s weight must be >0", s.ID)
		}
		if _, ok := seen[s.ID]; ok {
			return nil, fmt.Errorf("duplicate signer id %s", s.ID)
		}
		seen[s.ID] = struct{}{}
		if s.Alg == "" {
			cfg.Signers[i].Alg = "ED25519"
		}
		total += s.Weight
	}
	if cfg.ThresholdWeight > total {
		return nil, fmt.Errorf("threshold_weight %d exceeds total signer weight %d", cfg.ThresholdWeight, total)
	}
	return &cfg, nil
}

// BuildArtifactFromConfig produces an artifact (without signatures) and exports metrics.
func BuildArtifactFromConfig(cfg *WeightsConfig, prevHash string, now time.Time, resolver func(string) ed25519.PublicKey) (WeightedRotationArtifact, error) {
	if cfg == nil {
		return WeightedRotationArtifact{}, errors.New("config nil")
	}
	signers := make([]WeightedRotationSigner, 0, len(cfg.Signers))
	eff := 0
	embedFlag := os.Getenv("GAUTH_ROTATIONS_V2_EMBED_PUBS") == "1"
	embeddedCount := 0
	for _, s := range cfg.Signers {
		signer := WeightedRotationSigner{ID: s.ID, Alg: s.Alg, Weight: s.Weight}
		// Optional embedding: GAUTH_ROTATIONS_V2_EMBED_PUBS=1 attempts to resolve public key and embed.
		if embedFlag {
			algUpper := strings.ToUpper(s.Alg)
			switch algUpper {
			case "ED25519":
				if resolver != nil {
					if pk := resolver(s.ID); len(pk) == ed25519.PublicKeySize {
						signer.Public = base64.RawURLEncoding.EncodeToString(pk)
						embeddedCount++
					}
				}
			case "ECDSA-P256", "ECDSA_P256", "ECDSA-P256-SHA256":
				// Environment stub GAUTH_ROTATIONS_V2_ECDSA_KEYS="id:base64urlUncompressed,id2:base64urlUncompressed"
				rawMap := os.Getenv("GAUTH_ROTATIONS_V2_ECDSA_KEYS")
				if rawMap != "" {
					entries := strings.Split(rawMap, ",")
					for _, e := range entries {
						parts := strings.SplitN(e, ":", 2)
						if len(parts) != 2 || parts[0] != s.ID {
							continue
						}
						b, err := base64.RawURLEncoding.DecodeString(parts[1])
						if err != nil || len(b) != 65 || b[0] != 0x04 {
							continue
						}
						// Basic curve point validation
						x := new(big.Int).SetBytes(b[1:33])
						y := new(big.Int).SetBytes(b[33:65])
						if !elliptic.P256().IsOnCurve(x, y) {
							continue
						}
						signer.Public = parts[1]
						embeddedCount++
						break
					}
				}
			}
		}
		signers = append(signers, signer)
		eff += s.Weight
		rotationV2SignerWeight.WithLabelValues(s.ID).Set(float64(s.Weight))
	}
	rotationV2ThresholdWeight.Set(float64(cfg.ThresholdWeight))
	// Placeholder: effective weight = total configured (no verification yet)
	rotationV2EffectiveWeight.Set(float64(eff))
	if embedFlag {
		rotationV2PublicKeysEmbedded.Inc()
	}
	rotationV2EmbeddedKeyCount.Set(float64(embeddedCount))
	return BuildWeightedRotationArtifact(cfg.ActiveKeySetID, prevHash, cfg.ThresholdWeight, signers, cfg.AlgorithmSuite, now)
}

// PublicKeyResolver abstracts lookup of signer public key material.
type PublicKeyResolver interface {
	// FindByID returns a PublicKeyRecord for the signer id or nil if unknown.
	// Implementations may support multiple algorithms (Ed25519, ECDSA P-256).
	FindByID(id string) *PublicKeyRecord
}

// VerifyArtifactSignatures verifies each signature (Ed25519 only) and returns verifiedWeight, failures.
// It updates rotationV2VerifiedWeight metric. Failures recorded per reason.
func VerifyArtifactSignatures(art *WeightedRotationArtifact, resolver PublicKeyResolver) (int, map[string]int, []string) {
	if art == nil {
		return 0, map[string]int{}, []string{"artifact_nil"}
	}
	var failures []string
	verified := 0
	byAlgWeight := map[string]int{}
	preimage := []byte("GAUTH_ROTATION_V2:" + art.CanonicalDigest)
	for _, s := range art.Signers {
		if s.Signature == "" {
			continue
		}
		if resolver == nil {
			failures = append(failures, "resolver_nil")
			rotationV2SignatureFailures.WithLabelValues("resolver_nil").Inc()
			continue
		}
		rec := resolver.FindByID(s.ID)
		algUpper := strings.ToUpper(s.Alg)
		sigBytes, err := base64.RawURLEncoding.DecodeString(s.Signature)
		if err != nil {
			failures = append(failures, "signature_decode")
			rotationV2SignatureFailures.WithLabelValues("signature_decode").Inc()
			rotationV2SignatureFailuresByAlg.WithLabelValues(algUpper, "signature_decode").Inc()
			continue
		}
		switch algUpper {
		case "ED25519":
			if rec == nil || len(rec.Ed25519) != ed25519.PublicKeySize {
				failures = append(failures, "public_key_not_found")
				rotationV2SignatureFailures.WithLabelValues("public_key_not_found").Inc()
				rotationV2SignatureFailuresByAlg.WithLabelValues(algUpper, "public_key_not_found").Inc()
				continue
			}
			if len(sigBytes) != ed25519.SignatureSize || !ed25519.Verify(rec.Ed25519, preimage, sigBytes) {
				failures = append(failures, "signature_invalid")
				rotationV2SignatureFailures.WithLabelValues("signature_invalid").Inc()
				rotationV2SignatureFailuresByAlg.WithLabelValues(algUpper, "signature_invalid").Inc()
				continue
			}
			verified += s.Weight
			byAlgWeight[algUpper] += s.Weight
		case "ECDSA-P256", "ECDSA_P256", "ECDSA-P256-SHA256":
			if rec == nil || rec.ECDSA == nil || rec.ECDSA.Curve != elliptic.P256() {
				failures = append(failures, "public_key_not_found")
				rotationV2SignatureFailures.WithLabelValues("public_key_not_found").Inc()
				rotationV2SignatureFailuresByAlg.WithLabelValues(algUpper, "public_key_not_found").Inc()
				continue
			}
			h := sha256.Sum256(preimage)
			var rs struct{ R, S *big.Int }
			if _, err := asn1.Unmarshal(sigBytes, &rs); err != nil || rs.R == nil || rs.S == nil {
				failures = append(failures, "signature_decode")
				rotationV2SignatureFailures.WithLabelValues("signature_decode").Inc()
				rotationV2SignatureFailuresByAlg.WithLabelValues(algUpper, "signature_decode").Inc()
				continue
			}
			if !ecdsa.Verify(rec.ECDSA, h[:], rs.R, rs.S) {
				failures = append(failures, "signature_invalid")
				rotationV2SignatureFailures.WithLabelValues("signature_invalid").Inc()
				rotationV2SignatureFailuresByAlg.WithLabelValues(algUpper, "signature_invalid").Inc()
				continue
			}
			verified += s.Weight
			byAlgWeight[algUpper] += s.Weight
		default:
			failures = append(failures, "unknown_alg")
			rotationV2SignatureFailures.WithLabelValues("unknown_alg").Inc()
			rotationV2SignatureFailuresByAlg.WithLabelValues(algUpper, "unknown_alg").Inc()
		}
	}
	rotationV2VerifiedWeight.Set(float64(verified))
	for alg, w := range byAlgWeight {
		rotationV2VerifiedWeightByAlg.WithLabelValues(alg).Set(float64(w))
	}
	if art.ThresholdWeight > 0 && verified < art.ThresholdWeight {
		rotationV2ThresholdViolations.Inc()
	}
	return verified, byAlgWeight, failures
}
