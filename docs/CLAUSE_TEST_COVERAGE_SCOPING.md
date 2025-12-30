---
title: Clause Test Coverage Scoping
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# AAP-001 / AAP-002 Clause-to-Test Coverage Scoping

> Draft: 2025-10-20
> Purpose: Establish a structured mapping framework linking specification clauses to concrete test cases and code artifacts to enable measurable compliance progression.

## 1. Objectives
- Provide a machine-readable index (JSON) enumerating RFC clauses and expected test coverage.
- Enable automated gap detection: clauses with zero associated tests flagged P0.
- Surface artifact types: unit test, integration test, property test, fuzz test, benchmark.
- Prepare for future CI gate (minimum clause coverage percentage).

## 2. Proposed Data Model (JSON)
```json
{
  "generated": "2025-10-20T00:00:00Z",
  "rfc": {
    "0111": {
      "sections": [
        {"id": "0111:authz-combining", "title": "Combining Algorithms", "expected": ["deny_overrides", "permit_overrides", "first_applicable"], "tests": []},
        {"id": "0111:jti-format", "title": "JTI Format Validation", "expected": ["uuidv4"], "tests": []},
        {"id": "0111:replay-protection", "title": "Replay Protection", "expected": ["first-seen", "fail-open", "latency-metrics"], "tests": []}
      ]
    },
    "0115": {
      "sections": [
        {"id": "0115:poa-structure", "title": "PoA Structural Fields", "expected": ["parties", "scope", "requirements"], "tests": []},
        {"id": "0115:poa-semantic", "title": "Semantic Validation Rules", "expected": ["non-self-delegation", "transaction-currency", "temporal-invariants", "numeric-limits"], "tests": []},
        {"id": "0115:canonical-digest", "title": "Canonical Digest Stability", "expected": ["stable-permutation", "domain-separation"], "tests": []}
      ]
    }
  }
}
```

## 3. Script Plan (`scripts/generate_clause_coverage.go`)
### Responsibilities
- Load static JSON template (e.g. `docs/coverage_template.json`).
- Scan `./pkg` and `./test` for test files containing markers (`//clause:0111:authz-combining`).
- Populate `tests` arrays with file paths + test function names.
- Emit `docs/CLAUSE_TEST_COVERAGE.json` with filled data.
- Output summary: total clauses, covered clauses, coverage percentage.

### Edge Cases
- Multiple markers in same test function (associate once per clause).
- Missing markers → leave `tests` empty (declared gap).
- Deprecated clauses (future) flagged via `deprecated": true` to exclude from denominator.

### Error Modes
- Missing template file → exit non-zero.
- Malformed marker format → log warning, skip.
- Write failure (permissions) → exit non-zero.

## 4. Marking Convention
Add a comment at top of test functions:
```go
//clause:0111:authz-combining
func TestDenyOverrides(t *testing.T) { ... }
```
Multiple clauses:
```go
//clause:0115:canonical-digest
//clause:0115:poa-semantic
func TestCanonicalDigestStable(t *testing.T) { ... }
```

## 5. CI Integration
- Add `make coverage-clause` target running `go run scripts/generate_clause_coverage.go`.
- CI fails if coverage < configured threshold (initially >0%). Threshold raised over time.

## 6. Initial Target Clauses (Subset)
| RFC | Clause ID | Priority | Current Coverage | Notes |
|-----|-----------|----------|------------------|-------|
| 0111 | 0111:authz-combining | P0 | TBD | Strategies tested; need markers |
| 0111 | 0111:jti-format | P0 | TBD | Regex test exists; add marker |
| 0111 | 0111:replay-protection | P1 | TBD | Add markers to replay tests |
| 0115 | 0115:poa-structure | P0 | TBD | Structural tests in `poa_test.go` |
| 0115 | 0115:poa-semantic | P0 | TBD | `BasicPoAValidator` tests |
| 0115 | 0115:canonical-digest | P1 | TBD | Canonical digest property tests |

## 7. Roadmap
1. Implement generator script with minimal clause subset.
2. Add markers to existing tests iteratively.
3. Expand clause list referencing GAP_MATRIX rows (1:1 mapping where feasible).
4. Introduce property/fuzz test clauses post harness addition.

## 8. Next Steps

## 9. Run Instructions (Initial)
1. Add markers to tests (example):
  ```go
  //clause:0111:authz-combining
  func TestDenyOverridesStrategy(t *testing.T) { /* ... */ }
  ```
2. Execute generator:
  ```bash
  go run ./cmd/coverage > /dev/null
  ```
3. Inspect output file:
  ```bash
  jq '.rfc["0111"].sections[] | {id:.id, tests_count:(.tests|length)}' docs/CLAUSE_TEST_COVERAGE.json
  ```
4. CI integration suggestion (Makefile target):
  ```make
  clause-coverage:
	go run ./cmd/coverage || exit 1
	@echo "Generated docs/CLAUSE_TEST_COVERAGE.json"
  ```
5. Failing threshold example (bash snippet):
  ```bash
  pct=$(jq -r '(.rfc["0111"].sections + .rfc["0115"].sections | map(select(.tests|length>0) | length) / (.rfc["0111"].sections + .rfc["0115"].sections | length) * 100' docs/CLAUSE_TEST_COVERAGE.json)
  echo "Clause coverage: ${pct}%"
  awk "BEGIN {exit (${pct} < 5)}"  # require at least 5%
  ```

## 10. Future Enhancements
- Add `deprecated:true` flag for retired clauses.
- Support multi-file mapping summary (list unique functions per clause).
- Emit coverage badge (SVG) based on percentage.
- Add property/fuzz test classification (suffix markers: `//clause:0115:canonical-digest:fuzz`).
- Integrate with GAP matrix: auto-add clause entries for each row without status Missing.

---
Draft prepared for implementation.
