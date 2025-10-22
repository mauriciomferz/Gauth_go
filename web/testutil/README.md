# testutil package

Helpers and JSON fixtures supporting web-layer capability & policy tests.

## Contents
- `fixtures.go`: Canonical JSON fixture constants (valid + negative cases).
- `parse.go`: Lightweight parsing, validation, hashing helpers.
- `fixtures_test.go`: Unit tests covering success, error, and validation paths.
- `registry_list.go`: Enumerations of valid fixtures, iteration, and canonicalization helper.
 - `validation_table_test.go`: Table-driven tests for all registry fixtures and canonical hash.
 - `policy_list.go`: Valid policy bundle fixtures & iteration helper.
 - `policy_test.go`: Tests for policy bundle canonicalization & hashing.
 - `benchmark_test.go`: Micro benchmarks for hashing & canonicalization routines.
 - `fuzz_test.go`: Fuzz targets for capability registry & policy bundle parsing (Go 1.18+).

## Fixture Catalog

| Constant | Category | Purpose / Notes |
|----------|----------|-----------------|
| `CapTransferV1` | Base | Single capability & mapping (execute). |
| `CapTransferIssueV1` | Composite | Adds issue capability + mapping. |
| `CapTransferIssueDelegationCreateV1` | Composite | Adds delegation create capability & mapping. |
| `CapTransferAuditV1` | Hash Change | Introduces audit capability for hash drift tests. |
| `CapAlphaV1` | Minimal | Simplest stable capability registry. |
| `CapAlphaBetaIssueV1` | Composite | Two capabilities, single action mapping referencing beta. |
| `CapAlphaBetaGammaDelegationIssueV1` | Composite / Unstable | Adds unstable gamma capability; delegation + issue mappings. |
| `CapABDelegationIssuePerm1V1` | Permutation | Ordering variant #1 (delegation/create list order differs). |
| `CapABDelegationIssuePerm2V1` | Permutation | Ordering variant #2 to test permutation hash stability/prev hash semantics. |
| `CapABCDelegationIssueV1` | Semantic Change | Adds new capability `cap.c` referenced by mapping (causes hash + prev hash update). |
| `CapAlphaUnknownMapping` | Negative | Action mapping references missing capability id. |
| `CapAlphaDuplicateIDs` | Negative | Duplicate capability id entries to trigger duplicate error. |
| `CapAlphaMissingSchemaVersion` | Negative | Omits `schema_version` field. |
| `PolicyBundleB1V1` | Policy Base | Single policy bundle for append tests. |
| `PolicyBundleB2V1` | Policy Append | Second bundle appended after B1 (persistence scenarios). |
| `PolicyBundleMultiPerm1V1` | Policy Permutation | Multi-policy bundle order variant #1 for canonicalization tests. |
| `PolicyBundleMultiPerm2V1` | Policy Permutation | Multi-policy bundle order variant #2 (should hash identically to variant #1). |
| `PolicyBundleMultiPlusP3V1` | Policy Semantic Change | Adds third policy `p3` (introduces new actions/effect) causing hash change. |

Golden canonical hashes (see `hash_golden_test.go`) guard against unintended canonicalization changes:

| Fixture | Canonical SHA256 |
|---------|-----------------|
| `CapAlphaV1` | `3f75890a75a5c856e3027876ae7e05dc5f47569c51c7f5612ef5f8b481a1fd98` |
| `CapAlphaBetaIssueV1` | `c9efab04a4b6e74be7f3c0668b4abe039a29dec7523f60076ce839d392422fc0` |
| `CapABDelegationIssuePerm1V1` | `bccd9d013c5ba68e840e9c26f68b4dac4f00c1783ac3ed684901a93e9a8910e0` |
| `CapABDelegationIssuePerm2V1` | `1c20e5f8a106e1649518be4afab4d636ce5d51d269eb63dccdc54dc1892bed5e` |
| `CapABCDelegationIssueV1` | `07781cd6b285b56465009ba1b79c17c4e43ec145d157d5b2cb64c583fbad6b78` |
| `PolicyBundleB1V1` | `0f4828f2d48daaca4a7cacb905d015fbedebc1a8e93a80ba0591a74323e20092` |
| `PolicyBundleMultiPerm1V1` | `415ef7e5328fead1469224134a0c880185a46360f5471da9a415301398c86931` |
| `PolicyBundleMultiPerm2V1` | `415ef7e5328fead1469224134a0c880185a46360f5471da9a415301398c86931` |
| `PolicyBundleMultiPlusP3V1` | `a9cd9b6a460a926a20fec3b3d79fe45f243c2f68ee86b2bc9a68a6a341dfcc79` |

## Usage
```go
import "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web/testutil"

func setup(t *testing.T) {
    reg := testutil.MustCapabilityRegistry(testutil.CapTransferIssueDelegationCreateV1)
    // use reg.Capabilities, reg.ActionMapping...
}

func hashFixture() string {
    return testutil.SHA256Hex(testutil.CapTransferV1)
}

// Iterate all valid registries
testutil.IterateValidRegistries(func(name, raw string) bool {
    reg := testutil.MustCapabilityRegistry(raw)
    _ = reg // use it
    return true // continue
})

// Canonical representation (deterministic ordering)
canon := testutil.CanonicalizeRegistry(testutil.CapTransferIssueDelegationCreateV1)

// Canonical hash (order-insensitive for capabilities & action keys)
hash := testutil.CanonicalRegistryHash(testutil.CapTransferIssueDelegationCreateV1)

// Policy bundle canonicalization & hash
pcbCanon := testutil.CanonicalizePolicyBundle(testutil.PolicyBundleB1V1)
pcbHash := testutil.CanonicalPolicyBundleHash(testutil.PolicyBundleB1V1)

// Typed error extraction
if _, err := testutil.ParseCapabilityRegistry(testutil.CapAlphaUnknownMapping); err != nil {
    if regErr, ok := testutil.AsCapabilityRegistryError(err); ok {
        // handle structured fields regErr.Kind, regErr.Action, regErr.CapabilityID
    }
}

// Fuzzing (runs only with -fuzz flag):
//   go test -fuzz=FuzzParseCapabilityRegistry -run=^$ ./web/testutil
//   go test -fuzz=FuzzParsePolicyBundle -run=^$ ./web/testutil
// Add new seed fixtures by appending constants in fixtures.go and updating slices in fuzz_test.go.
```

## Error Semantics
- `ErrMissingSchemaVersion`: `schema_version` absent or zero.
- `ErrDuplicateCapabilityID`: More than one capability entry shares the same `id`.
- Unknown capability referenced by an action mapping returns a formatted error from `ParseCapabilityRegistry`.

Use `Must*` variants in tests where a failure should abort immediately; use `Parse*` when asserting specific error values.

## Adding New Fixtures
1. Append new JSON constant to `fixtures.go` (preserve one-line formatting if byte-for-byte comparisons or hash stability matters).
2. Add corresponding parse test if it introduces new structural aspects.
3. Avoid reformatting existing JSON to keep historical hashes stable.

## Future Enhancements (Considerations)
- Normalization / canonicalization for semantically equivalent registries.
- Rich error types including offending action or capability id.
- Table-driven test set iterating all valid registry fixtures.
- Potential integration with domain model types to avoid duplication.
 - Export benchmark performance badges or integrate into CI threshold.
 - Nightly fuzz smoke in CI with short `-fuzztime` to catch regressions earlier.
 - Policy bundle permutation & semantic hash golden tests for parity.
 - Simplify hash golden updates via generated file or snapshot command.
 - Registry hash allocation pooling / strconv-based integer formatting for micro-optimizations.

## Performance Guardrails
Benchmark guardrails are enforced via `.github/workflows/bench-guardrail.yml` which compares current results to a baseline.

Key micro benchmarks (hashing/canonicalization) can be filtered with:

```bash
GUARD_TARGETS='BenchmarkCanonical(Registry|Policy)BundleHash' scripts/bench_guardrail.sh baseline.txt current.txt 25
```

Defaults:
 - Threshold: 20–25% slowdown triggers failure (configurable in workflow input).
 - Baseline: Nightly artifact if available; otherwise first run in PR becomes provisional baseline.

Current reference (Apple M3 Pro, single run; use only as guidance):

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| CanonicalPolicyBundleHash | ~1900 | ~1120 | 24 |
| CanonicalPolicyBundleHashMultiPermutation | ~3200 | ~1670 | 33 |
| CanonicalPolicyBundleHashMultiSemanticAdd | ~5700 | ~2240 | 40 |
| CanonicalRegistryHash | ~3800* | ~2850 | 41 |

Adjust thresholds cautiously; prefer relative (%) guardrails over absolute values to accommodate hardware variance.

*Optimization note (Oct 2025): Introduction of a pooled `strings.Builder` plus `strconv.Itoa` integer formatting reduced the `CanonicalRegistryHash` benchmark from a prior ~4000 ns/op (manual builder without pooling) to the current ~3790–3800 ns/op on Apple M3 Pro while keeping allocations and hash output stable. Golden hashes were preserved by maintaining byte-exact JSON formatting (see comment in `parse.go`). Minor run-to-run variance (±2–3%) is expected; do not treat single-run deltas within that band as regressions.

## CI Fuzz Smoke
The workflow `.github/workflows/fuzz-smoke.yml` runs short (10s) fuzz iterations for:

```
FuzzParseCapabilityRegistry
FuzzParsePolicyBundle
```

Adjust `-fuzztime` in the workflow to deepen coverage (e.g. `30s` or `2m`). For more exhaustive fuzzing, run locally:

```bash
go test -run=^$ -fuzz=FuzzParseCapabilityRegistry -fuzztime=2m ./web/testutil
go test -run=^$ -fuzz=FuzzParsePolicyBundle -fuzztime=2m ./web/testutil
```

Any new crash seeds will be stored in the testdata directory automatically by Go; commit deterministic repros as needed.

PRs welcome.
