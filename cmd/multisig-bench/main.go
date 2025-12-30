package main

// RB14 Multi-Signature Benchmark Harness (initial skeleton)
// Measures signing + verification throughput across varying signer counts.
// Current implementation uses individual Ed25519 signatures (no true aggregate scheme yet).
// Aggregate signature size simulated as concatenation of individual signatures.
//
// Output: newline-delimited JSON records per signer group. Example:
// {"signers":8,"mode":"both","iterations":100,"avg_sign_ns":12345,"avg_verify_ns":23456,"bytes_per_signature":64,"aggregate_signature_bytes":512}
//
// Flags:
//   --signers        Comma-separated list of signer counts (default "1,2,4,8,16,32")
//   --iterations     Number of iterations per signer count (default 100)
//   --mode           Mode: sign|verify|both (default both)
//   --threshold      Threshold value (N-of-M) for future aggregated scheme (default 0 = ignore)
//   --summary-file   Optional path to write aggregate summary JSON (single object)
//   --metrics        Emit internal metrics (latency observations) to stdout at end
//   --seed           Deterministic seed integer for key generation (default 42)
//
// Roadmap:
//   - Integrate real multi-signature aggregate + threshold scheme
//   - Add p50/p95/p99 latency computation per signer count
//   - Export Prometheus metrics when run in sidecar mode
//   - SVG curve artifact (signers vs latency)
//   - Compare Ed25519 vs future BLS aggregated path
//

// Note: This harness avoids third-party benchmark frameworks to keep output machine-readable.

import (
	"crypto/ed25519"
	crand "crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	mrand "math/rand"
	"os"
	"strings"
	"time"

	imetrics "github.com/mauriciomferz/AgentAuth/internal/metrics"
)

const (
	modeSign   = "sign"
	modeVerify = "verify"
	modeBoth   = "both"
)

type record struct {
	Signers                 int    `json:"signers"`
	Mode                    string `json:"mode"`
	Iterations              int    `json:"iterations"`
	AvgSignNS               int64  `json:"avg_sign_ns,omitempty"`
	AvgVerifyNS             int64  `json:"avg_verify_ns,omitempty"`
	P50SignNS               int64  `json:"p50_sign_ns,omitempty"`
	P95SignNS               int64  `json:"p95_sign_ns,omitempty"`
	P99SignNS               int64  `json:"p99_sign_ns,omitempty"`
	P50VerifyNS             int64  `json:"p50_verify_ns,omitempty"`
	P95VerifyNS             int64  `json:"p95_verify_ns,omitempty"`
	P99VerifyNS             int64  `json:"p99_verify_ns,omitempty"`
	BytesPerSignature       int    `json:"bytes_per_signature"`
	AggregateSignatureBytes int    `json:"aggregate_signature_bytes"`
	Threshold               int    `json:"threshold,omitempty"`
}

type summary struct {
	SignerGroups int    `json:"signer_groups"`
	TotalRecords int    `json:"total_records"`
	TotalSigners int    `json:"total_signers_accumulated"`
	Mode         string `json:"mode"`
	Iterations   int    `json:"iterations"`
	Timestamp    string `json:"timestamp"`
}

func parseSignerList(raw string) ([]int, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("empty signers list")
	}
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	seen := make(map[int]struct{})
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var v int
		for _, ch := range p {
			if ch < '0' || ch > '9' {
				return nil, fmt.Errorf("invalid signer count: %s", p)
			}
		}
		_, err := fmt.Sscanf(p, "%d", &v)
		if err != nil || v <= 0 {
			return nil, fmt.Errorf("invalid signer count: %s", p)
		}
		if _, exists := seen[v]; exists {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	// deterministic ordering
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

//nolint:gocyclo // Main function with CLI arg parsing, benchmark orchestration - acceptable complexity for entry point
func main() {
	var signerStr string
	var iterations int
	var mode string
	var threshold int
	var summaryPath string
	var emitMetrics bool
	var seed int64

	flag.StringVar(&signerStr, "signers", "1,2,4,8,16,32", "Comma-separated signer counts")
	flag.IntVar(&iterations, "iterations", 100, "Iterations per signer count")
	flag.StringVar(&mode, "mode", "both", "Mode: sign|verify|both")
	flag.IntVar(&threshold, "threshold", 0, "Threshold for future aggregate scheme (N-of-M); 0 = ignore")
	flag.StringVar(&summaryPath, "summary-file", "", "Optional summary JSON output file path")
	flag.BoolVar(&emitMetrics, "metrics", false, "Emit internal latency metrics snapshot at end")
	flag.Int64Var(&seed, "seed", 42, "Deterministic seed for key generation")
	flag.Parse()

	signerCounts, err := parseSignerList(signerStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse signers error: %v\n", err)
		os.Exit(1)
	}
	mode = strings.ToLower(mode)
	if mode != modeSign && mode != modeVerify && mode != modeBoth {
		fmt.Fprintf(os.Stderr, "invalid mode: %s\n", mode)
		os.Exit(1)
	}
	if threshold < 0 {
		fmt.Fprintf(os.Stderr, "threshold must be >=0\n")
		os.Exit(1)
	}

	// Deterministic RNG for message content generation (not cryptographic; signing uses ed25519 library RNG implicitly for key generation if needed).
	// #nosec G404
	rng := mrand.New(mrand.NewSource(seed))
	msg := make([]byte, 256)
	if _, err := rng.Read(msg); err != nil {
		fmt.Fprintf(os.Stderr, "rng read error: %v\n", err)
		os.Exit(1)
	}

	// Metrics collector (optional)
	var m = imetrics.Noop
	mem, useMem := (&imetrics.Memory{}), false
	if emitMetrics {
		mem = imetrics.NewMemory()
		m = mem
		useMem = true
	}

	records := make([]record, 0, len(signerCounts))

	for _, signers := range signerCounts {
		// Generate signer key pairs deterministically: we derive a seed per signer from base seed.
		keys := make([]ed25519.PrivateKey, signers)
		pubs := make([]ed25519.PublicKey, signers)
		for i := 0; i < signers; i++ {
			// Use crypto/rand for key generation to avoid weak keys even if deterministic harness; message randomness handles determinism.
			pub, priv, kErr := ed25519.GenerateKey(crand.Reader)
			if kErr != nil {
				fmt.Fprintf(os.Stderr, "key gen error: %v\n", kErr)
				os.Exit(1)
			}
			keys[i] = priv
			pubs[i] = pub
		}
		var totalSignNS int64
		var totalVerifyNS int64
		signSamples := make([]int64, 0, iterations)
		verifySamples := make([]int64, 0, iterations)
		signatures := make([][]byte, signers)

		for iter := 0; iter < iterations; iter++ {
			// Optionally regenerate message slight variation per iteration to simulate different workloads.
			if _, err := rng.Read(msg[:32]); err == nil {
				_ = err // ignore error for determinism
			}
			if mode == modeSign || mode == modeBoth {
				t0 := time.Now()
				for i := 0; i < signers; i++ {
					signatures[i] = ed25519.Sign(keys[i], msg)
				}
				dur := time.Since(t0).Nanoseconds()
				totalSignNS += dur
				signSamples = append(signSamples, dur)
				if useMem {
					m.ObserveMultiSignatureAggregateLatency(time.Duration(dur) * time.Nanosecond)
				}
			}
			if mode == modeVerify || mode == modeBoth {
				// Ensure signatures exist if verify-only mode (create once).
				if mode == modeVerify {
					// Re-sign each iteration to bind signatures to current message mutation.
					for i := 0; i < signers; i++ {
						signatures[i] = ed25519.Sign(keys[i], msg)
					}
				}
				t1 := time.Now()
				for i := 0; i < signers; i++ {
					if !ed25519.Verify(pubs[i], msg, signatures[i]) {
						fmt.Fprintf(os.Stderr, "verification failed signer=%d group=%d\n", i, signers)
						os.Exit(2)
					}
				}
				durV := time.Since(t1).Nanoseconds()
				totalVerifyNS += durV
				verifySamples = append(verifySamples, durV)
				if useMem {
					m.ObserveMultiSignatureVerificationLatency(time.Duration(durV) * time.Nanosecond)
				}
			}
		}

		avgSign := int64(0)
		avgVerify := int64(0)
		if mode == modeSign || mode == modeBoth {
			avgSign = totalSignNS / int64(iterations)
		}
		if mode == modeVerify || mode == modeBoth {
			avgVerify = totalVerifyNS / int64(iterations)
		}
		p50s, p95s, p99s := computePercentiles(signSamples)
		p50v, p95v, p99v := computePercentiles(verifySamples)
		rec := record{
			Signers:                 signers,
			Mode:                    mode,
			Iterations:              iterations,
			AvgSignNS:               avgSign,
			AvgVerifyNS:             avgVerify,
			P50SignNS:               p50s,
			P95SignNS:               p95s,
			P99SignNS:               p99s,
			P50VerifyNS:             p50v,
			P95VerifyNS:             p95v,
			P99VerifyNS:             p99v,
			BytesPerSignature:       ed25519.SignatureSize,
			AggregateSignatureBytes: ed25519.SignatureSize * signers,
		}
		if threshold > 0 {
			rec.Threshold = threshold
		}
		// Emit newline JSON record
		b, _ := json.Marshal(rec)
		fmt.Println(string(b))
		records = append(records, rec)
	}

	if summaryPath != "" {
		s := summary{SignerGroups: len(signerCounts), TotalRecords: len(records), TotalSigners: func() int {
			tot := 0
			for _, v := range signerCounts {
				tot += v
			}
			return tot
		}(), Mode: mode, Iterations: iterations, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)}
		sb, _ := json.MarshalIndent(&s, "", "  ")
		if err := os.WriteFile(summaryPath, sb, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "write summary error: %v\n", err)
		}
	}

	if emitMetrics && useMem {
		snap := mem.SnapshotEx()
		// Only output multi-signature related subset for brevity
		out := map[string]any{
			"multi_signature_verifications":              snap.MultiSignatureVerifications,
			"multi_signature_verification_failures":      snap.MultiSignatureVerificationFailures,
			"multi_signature_aggregate_latency_count":    snap.MultiSignatureAggregateLatencyCount,
			"multi_signature_aggregate_latency_total_ns": snap.MultiSignatureAggregateLatencyTotalNS,
			"multi_signature_aggregate_latency_max_ns":   snap.MultiSignatureAggregateLatencyMaxNS,
		}
		jb, _ := json.Marshal(out)
		fmt.Fprintf(os.Stderr, "METRICS %s\n", string(jb))
	}
}

// computePercentiles returns p50, p95, p99 (nanoseconds) for given samples.
// For n=0 all returned values are 0. For small n it uses index rounding with
// clamp to last element to avoid out-of-range.
func computePercentiles(samples []int64) (p50, p95, p99 int64) {
	n := len(samples)
	if n == 0 {
		return 0, 0, 0
	}
	// Copy then sort ascending
	cp := make([]int64, n)
	copy(cp, samples)
	// Simple insertion sort (n expected small <10k)
	for i := 1; i < n; i++ {
		j := i
		for j > 0 && cp[j] < cp[j-1] {
			cp[j], cp[j-1] = cp[j-1], cp[j]
			j--
		}
	}
	// Helper to compute index
	idx := func(q float64) int {
		if q < 0 {
			q = 0
		} else if q > 1 {
			q = 1
		}
		// nearest rank method
		pos := int(q*float64(n-1) + 0.5)
		if pos >= n {
			pos = n - 1
		}
		if pos < 0 {
			pos = 0
		}
		return pos
	}
	p50 = cp[idx(0.50)]
	p95 = cp[idx(0.95)]
	p99 = cp[idx(0.99)]
	return p50, p95, p99
}
