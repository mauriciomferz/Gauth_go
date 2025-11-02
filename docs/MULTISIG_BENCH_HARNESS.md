# Multi-Signature Benchmark Harness (RB14)

Path: `cmd/multisig-bench`

## Purpose
Measure signing and verification latency across varying signer counts to establish baseline performance curves prior to introducing true aggregate multi-signature schemes.

## Current Status
Skeleton implementation using individual Ed25519 signatures; aggregate signature is simulated as concatenation of individual signatures. Provides per-signer-group average nanoseconds for sign and verify phases.

## Usage
```bash
go run ./cmd/multisig-bench --signers 1,2,4,8,16,32 --iterations 100 --mode both --seed 42 > bench.jsonl
```

Flags:
- `--signers` Comma separated list of signer counts (default `1,2,4,8,16,32`).
- `--iterations` Iterations per signer count (default `100`).
- `--mode` `sign|verify|both` (default `both`).
- `--threshold` Placeholder for future threshold aggregate scheme (ignored when 0).
- `--summary-file` Optional path to write summary JSON object.
- `--metrics` Emit internal latency metrics snapshot to stderr.
- `--seed` Deterministic seed for RNG (default `42`).

## Output
Newline-delimited JSON per signer count:
```json
{"signers":8,"mode":"both","iterations":100,"avg_sign_ns":12345,"avg_verify_ns":23456,"bytes_per_signature":64,"aggregate_signature_bytes":512}
```

Optional summary file:
```json
{
  "signer_groups": 6,
  "total_records": 6,
  "total_signers_accumulated": 63,
  "mode": "both",
  "iterations": 100,
  "timestamp": "2025-10-27T12:10:00Z"
}
```

## Metrics (when `--metrics` enabled)
Emits subset to stderr:
```json
{"multi_signature_verifications":0,"multi_signature_verification_failures":0,"multi_signature_aggregate_latency_count":6,"multi_signature_aggregate_latency_total_ns":987654,"multi_signature_aggregate_latency_max_ns":456789}
```

## Roadmap
1. Integrate real aggregate signature & batch verification primitives.
2. Add percentile latency (p50, p95, p99) and standard deviation.
3. Prometheus exposition mode (push or sidecar scrape).
4. Weight-based threshold scenarios (variable weights vs count-only).
5. Curve artifact generation (SVG) for documentation.
6. Comparative runs (Ed25519 individual vs aggregate BLS) once implemented.

## Design Notes
- Deterministic RNG only influences message content; key generation uses crypto/rand to maintain cryptographic integrity even in benchmark context.
- Harness avoids Go testing benchmarking to keep output machine-readable and CI friendly.
- Average latency computed as total nanoseconds divided by iterations; each iteration signs/verifies all signatures for the group.

## Error Handling
- Invalid signer list or mode exits with non-zero code and message to stderr.
- Verification failure (should not happen) aborts immediately for safety.

## Future Extensions
- Add `--json-output` for structured multi-object output file.
- Integrate memory profiler hooks for analyzing allocations per signer count.
- Parallel execution option to measure scaling under concurrency.
