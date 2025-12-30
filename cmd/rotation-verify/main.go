package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	notary "github.com/mauriciomferz/AgentAuth/internal/notary"
)

// artifactFile structure expecting top-level object containing an "artifact" field mirroring RotationSummaryV2.
type artifactEnvelope struct {
	Artifact *notary.WeightedRotationArtifact `json:"artifact"`
	// Accept alternate shapes by directly treating the file as artifact if top-level fields match.
	Version              int                             `json:"version"`
	ActiveKeySetID       string                          `json:"active_key_set_id"`
	PreviousArtifactHash string                          `json:"previous_artifact_hash"`
	ThresholdWeight      int                             `json:"threshold_weight"`
	Signers              []notary.WeightedRotationSigner `json:"signers"`
	AlgorithmSuite       []string                        `json:"algorithm_suite"`
	CanonicalDigest      string                          `json:"canonical_digest"`
	GeneratedAt          string                          `json:"generated_at"`
}

// simple in-memory resolver built from --pub or embedded signer.Public
type resolver struct {
	m map[string]notary.PublicKeyRecord
}

func (r *resolver) FindByID(id string) *notary.PublicKeyRecord {
	rec, ok := r.m[id]
	if !ok {
		return nil
	}
	return &rec
}

func parsePubSpec(spec string) (id string, rec notary.PublicKeyRecord, err error) {
	// format: id:ALG:base64urlpub  (ALG currently only ED25519)
	parts := strings.Split(spec, ":")
	if len(parts) < 3 {
		return "", rec, errors.New("invalid pub spec; want id:ALG:base64url")
	}
	id = parts[0]
	alg := strings.ToUpper(parts[1])
	rawB64 := parts[2]
	b, err := base64.RawURLEncoding.DecodeString(rawB64)
	if err != nil {
		return "", rec, fmt.Errorf("decode: %w", err)
	}
	switch alg {
	case "ED25519":
		if len(b) != ed25519.PublicKeySize {
			return "", rec, fmt.Errorf("ed25519 pub key wrong size %d", len(b))
		}
		rec.Ed25519 = ed25519.PublicKey(b)
	default:
		return "", rec, fmt.Errorf("unsupported alg %s", alg)
	}
	return id, rec, nil
}

func buildResolver(art *notary.WeightedRotationArtifact, userPubs []string) *resolver {
	r := &resolver{m: map[string]notary.PublicKeyRecord{}}
	// embed from artifact if present
	if art != nil {
		for _, s := range art.Signers {
			if s.Public != "" && strings.EqualFold(s.Alg, "ED25519") {
				if b, err := base64.RawURLEncoding.DecodeString(s.Public); err == nil && len(b) == ed25519.PublicKeySize {
					r.m[s.ID] = notary.PublicKeyRecord{Ed25519: ed25519.PublicKey(b)}
				}
			}
		}
	}
	for _, spec := range userPubs {
		if spec == "" {
			continue
		}
		id, rec, err := parsePubSpec(spec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: %v\n", err)
			continue
		}
		r.m[id] = rec // override/insert
	}
	return r
}

func loadArtifact(path string) (*notary.WeightedRotationArtifact, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var env artifactEnvelope
	if err := json.Unmarshal(b, &env); err != nil {
		return nil, err
	}
	if env.Artifact != nil {
		return env.Artifact, nil
	}
	// attempt direct artifact interpretation
	if env.Version == 2 && env.ActiveKeySetID != "" && env.CanonicalDigest != "" {
		art := notary.WeightedRotationArtifact{
			Version:              env.Version,
			ActiveKeySetID:       env.ActiveKeySetID,
			PreviousArtifactHash: env.PreviousArtifactHash,
			ThresholdWeight:      env.ThresholdWeight,
			Signers:              env.Signers,
			AlgorithmSuite:       env.AlgorithmSuite,
			CanonicalDigest:      env.CanonicalDigest,
			GeneratedAt:          env.GeneratedAt,
		}
		return &art, nil
	}
	return nil, errors.New("file did not contain recognizable artifact")
}

func main() {
	var file string
	var pubList multiFlag
	var expectDigest string
	var jsonOut bool
	flag.StringVar(&file, "file", "", "Path to Rotation V2 artifact JSON (or response)")
	flag.Var(&pubList, "pub", "Additional public key spec id:ALG:base64url (repeatable)")
	flag.StringVar(&expectDigest, "expect-digest", "", "If set, fail if artifact canonical_digest differs")
	flag.BoolVar(&jsonOut, "json", false, "Emit JSON result instead of human text")
	flag.Parse()
	if file == "" {
		fmt.Fprintln(os.Stderr, "--file required")
		os.Exit(2)
	}
	art, err := loadArtifact(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if expectDigest != "" && art.CanonicalDigest != expectDigest {
		fmt.Fprintf(os.Stderr, "error: digest mismatch (have %s)\n", art.CanonicalDigest)
		os.Exit(1)
	}
	// Build resolver (artifact embedded pubs + user pubs)
	r := buildResolver(art, pubList)
	vw, byAlg, failures := notary.VerifyArtifactSignatures(art, r)
	thresholdMet := vw >= art.ThresholdWeight
	if jsonOut {
		out := map[string]any{
			"verified_weight":  vw,
			"threshold_met":    thresholdMet,
			"failures":         failures,
			"by_alg":           byAlg,
			"canonical_digest": art.CanonicalDigest,
			"timestamp":        time.Now().UTC().Format(time.RFC3339Nano),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
		if !thresholdMet {
			os.Exit(3)
		}
		return
	}
	// human output
	fmt.Printf("Rotation V2 Artifact Digest: %s\n", art.CanonicalDigest)
	fmt.Printf("Threshold: %d  Verified: %d  Status: %s\n", art.ThresholdWeight, vw, map[bool]string{true: "OK", false: "NOT_MET"}[thresholdMet])
	// signer summary sorted by id
	sort.Slice(art.Signers, func(i, j int) bool { return art.Signers[i].ID < art.Signers[j].ID })
	for _, s := range art.Signers {
		emb := ""
		if rec := r.FindByID(s.ID); rec != nil && rec.Ed25519 != nil {
			emb = "pub"
		}
		fmt.Printf("  - %s (%s) weight=%d sig=%v %s\n", s.ID, s.Alg, s.Weight, s.Signature != "", emb)
	}
	if len(failures) > 0 {
		fmt.Printf("Failures: %v\n", failures)
	}
	for alg, w := range byAlg {
		fmt.Printf("Alg %s verified weight: %d\n", alg, w)
	}
	if !thresholdMet {
		os.Exit(3)
	}
}

// multiFlag collects repeated flags.
type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }
