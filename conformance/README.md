---
title: Readme
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# Conformance Harness

Automated, evidence-based verification that the implementation aligns with RFC 0111 and RFC 0115 clauses.

## Current Components
- `harnesslib/scan.go`: Parses canonical RFC markdown into structured clause IDs (slugified + RFC prefix).
- `harnesslib/analyze.go`: Loads `clause_map.json`, builds an AST-driven symbol index (functions, types, vars, consts) and evaluates clause → symbol → test coverage.
- `harnesslib/report.go`: Emits JSON + Markdown including summary stats, failure categories, and symbol evidence (file:line).
- `harnesslib/symbol_index_test.go`: Unit test ensuring critical symbols are discoverable.
- Deprecated legacy placeholder directory: `conformance/harness/` (to be removed once no external imports rely on it).

## Key Features
| Feature | Status | Notes |
|---------|--------|-------|
| RFC Markdown scanning | Implemented | Heading → clause extraction with line numbers |
| Clause mapping (`clause_map.json`) | Implemented | Drives required symbols & test globs |
| AST symbol indexing | Implemented | Skips vendor; skips test files unless env override |
| Evidence collection | Implemented | `Evidence[symbol] = [file:line,...]` |
| Coverage percent | Implemented | `SymbolsFound / RequiredSymbols` |
| Failure categorization | Implemented | Missing clauses / symbols / tests tallied |
| Clause ID normalization | Implemented | Fallback matching strips digits & punctuation |
| Env-gated debug | Implemented | `CONFORMANCE_DEBUG=1` prints indexing stats |
| Env include tests | Implemented | `CONFORMANCE_INCLUDE_TESTS=1` indexes *_test.go |
| Markdown evidence table | Implemented | Locations rendered with `<br>` separators |
| GAP matrix integration | Implemented | Counts of Implemented / Partial / Missing loaded from `docs/conformance_gaps.json` |
| Overrides mechanism | Planned | Temporal exceptions (expiry enforced) |
| Symbol index caching | Planned | Performance improvement for CI |

## Environment Variables
- `CONFORMANCE_DEBUG=1` — emit debug line to stderr with symbol counts and key symbol probe.
- `CONFORMANCE_INCLUDE_TESTS=1` — include test files in symbol index (off by default).
- `GAP_MATRIX_PATH=/custom/path.json` — override default path (`docs/conformance_gaps.json`).
- (Planned) `CONFORMANCE_CSV_OUT=dir` — destination directory for auto-written CSV artifacts (manual helper functions exist now).

## Output Artifacts
After analyzer run (via CLI or test harness):
- `report.json` — machine-readable summary, evidence, counts.
- `report.md` — human-readable summary, failures, evidence table.
- `gap_matrix.csv` (manual) — GAP items flattened (Section,ID,Requirement,Status,Priority,Gap,Evidence)
- `symbol_evidence.csv` (manual) — Implementation symbols and locations

## Failure Categories
## GAP Matrix Semantics
The GAP matrix file (`docs/conformance_gaps.json`) enumerates higher-level requirements beyond direct clause-symbol mapping. Each item has a `status` of one of:
- `Implemented` — functionality present (may still have improvement gaps).
- `Partial` — foundational code exists but lacks breadth, robustness, or full semantics.
- `Missing` — no meaningful implementation present.

The analyzer aggregates these into summary fields: `gap_implemented`, `gap_partial`, `gap_missing`, `gap_total` and renders them in the report summary. This provides a macro compliance trajectory separate from fine-grained clause coverage.

Failures are prefixed for machine parsing:
- `clause missing:` — clause expected by mapping not found in scanned RFCs.
- `symbol missing:` — required symbol absent from implementation index.
- `tests missing:` — test glob did not match any files.

Summary section includes counts: `MissingClauses`, `MissingSymbols`, `MissingTests` plus total coverage percent.

## `clause_map.json` Structure
```jsonc
{
   "entries": [
      {
         "clause_prefix": "0115:9.-canonical-serialization",
         "symbols": ["CanonicalPOADigest"],
         "tests_glob": "**/canonical_test.go"
      }
   ]
}
```
Notes:
- `clause_prefix` should match (or be a prefix of) the scanned clause ID. Normalization fallback reduces sensitivity to numbering changes.
- `tests_glob` uses project-root glob semantics.

## CLI Usage
The conformance runner exposes a CLI at `./cmd/conformance`.

Basic run with markdown + JSON outputs:
```bash
go run ./cmd/conformance \
   --markdown-out=report.md \
   --json-out=report.json
```

Add a condensed symbol locations summary file:
```bash
go run ./cmd/conformance \
   --markdown-out=report.md \
   --json-out=report.json \
   --symbol-locations-out=symbol_locations.md
```

Enable CSV exports and append to a history file (auto-generates `history_trend.md` in the same directory):
```bash
go run ./cmd/conformance \
   --csv-out=artifacts \
   --history-file=artifacts/history.csv \
   --markdown-out=artifacts/report.md \
   --json-out=artifacts/report.json
```

Apply gating thresholds (non-zero / non-negative values activate checks). Any violation exits with status code 2:
```bash
go run ./cmd/conformance \
   --min-symbol-coverage=85 \
   --max-gap-missing=12 \
   --max-missing-symbols=5 \
   --max-missing-tests=3
```

Generate an explicit trend dashboard file with a custom moving average window:
```bash
go run ./cmd/conformance \
   --history-file=artifacts/history.csv \
   --trend-markdown-out=artifacts/history_trend.md \
   --trend-window=15
```

### CLI Flags
| Flag | Purpose | Default |
|------|---------|---------|
| `--markdown-out` | Write markdown report | (empty) |
| `--json-out` | Write JSON report | (empty) |
| `--csv-out` | Directory for `gap_matrix.csv` & `symbol_evidence.csv` | (empty) |
| `--history-file` | Append run stats to history CSV | (empty) |
| `--trend-markdown-out` | Write trend dashboard markdown | (empty; auto `history_trend.md` if `--csv-out` set) |
| `--trend-window` | Coverage moving average window size | 10 |
| `--min-symbol-coverage` | Minimum required coverage percent | 0 (disabled) |
| `--max-gap-missing` | Max allowed GAP items with status Missing | -1 (disabled) |
| `--max-missing-symbols` | Max allowed missing required symbols | -1 (disabled) |
| `--max-missing-tests` | Max allowed missing test globs | -1 (disabled) |
| `--symbol-locations-out` | Write condensed symbol→locations markdown | (empty) |

Exit Codes:
- 0: Success (thresholds satisfied or disabled)
- 2: Threshold violation(s) (messages printed to stderr)

### History CSV Schema
Appended rows use the header:
```
timestamp,coverage,gap_missing,gap_partial,gap_implemented,missing_symbols,missing_tests
```
Each field:
- `timestamp`: RFC3339 UTC time
- `coverage`: symbol coverage percent (two decimals)
- `gap_missing`: count of GAP items with status Missing
- `gap_partial`: count of GAP items with status Partial
- `gap_implemented`: count of GAP items with status Implemented
- `missing_symbols`: number of required symbols not found
- `missing_tests`: number of test globs unmatched

### Trend Dashboard
When a history file is provided, the CLI can render a dashboard summarizing trajectory:
- Coverage latest value, moving average over `--trend-window`
- Delta (latest minus previous) for coverage, gap missing, gap implemented
- Recent runs table (last 20 entries) with full per-run metrics
Autogenerated path precedence:
1. Explicit `--trend-markdown-out`
2. `history_trend.md` inside `--csv-out` directory (if both `--csv-out` and `--history-file` are set)
3. Omitted if neither location resolved

Use this artifact to visualize regression or improvement trends in CI (e.g., publish as build artifact or comment on PR).

### CSV Export (Programmatic API)
Programmatic usage after building a `Report`:
```go
ar := Analyze(clauses)
rep := BuildReport(ar)
// GAP items CSV in-memory
csvGap := rep.GapItemsCSV()
// Symbol evidence CSV in-memory
csvSym := rep.SymbolEvidenceCSV()
// Write to files
_ = WriteGapCSV("gap_matrix.csv", rep)
_ = WriteSymbolEvidenceCSV("symbol_evidence.csv", rep)
```
Fields:
- Gap CSV columns: Section, ID, Requirement, Status, Priority, Gap, Evidence (pipe-separated evidence refs)
- Symbol CSV columns: Symbol, Locations (pipe-separated file:line)

Note: Quoting handled by `encoding/csv`; embedded commas appear inside quoted fields.

## Roadmap (Updated)
Phase 0 (DONE): Foundational scanner, analyzer, initial mapping, evidence, coverage.
Phase 1: GAP matrix integration, override file, richer status (Implemented / Partial / Missing).
Phase 2: Caching + performance, parallel symbol parsing, fuzz/spec invariants.
Phase 3 (DONE): Policy-driven CI gating thresholds (minimum coverage %, max missing tests).
Phase 4: Auto generation of mapping from inline spec annotations.

## Future Enhancements
- Add JSON `failure_categories` object separating arrays of each failure type for direct CI parsing.
- Add additional CLI flags: `--fail-on-missing-tests` convenience, `--min-coverage=90` alias.
- Integrate GAP matrix & policy engine to compute compliance tier.
- Persist historical trend (coverage over commits) for regression detection.

## Contributing
1. Update `clause_map.json` for new or reworded clauses.
2. Add symbols/tests; run `go test ./conformance/harnesslib` to verify.
3. Use `CONFORMANCE_DEBUG=1` when debugging missing symbols.
4. Avoid adding transient or flaky test globs—prefer deterministic unit tests.

---
This harness is in active development; interfaces may evolve before a stable v1 tagging.

