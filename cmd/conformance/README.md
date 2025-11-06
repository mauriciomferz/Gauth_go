# Conformance CLI Tool

Automated conformance analysis tool for RFC 0111 and RFC 0115 compliance verification.

## Overview

The conformance CLI tool analyzes your GAuth implementation against RFC 0111 (GAuth Core) and RFC 0115 (Power of Attorney) specifications. It:

- Scans the codebase for required symbols and test coverage
- Generates compliance reports in multiple formats (JSON, Markdown, CSV)
- Tracks conformance history over time
- Generates trend analysis dashboards
- Enforces quality gates via threshold checks

## Installation

Build the binary:

```bash
go build -o bin/conformance ./cmd/conformance
```

Or run directly:

```bash
go run ./cmd/conformance [flags]
```

## Usage

### Basic Analysis

Run conformance analysis and print summary:

```bash
./bin/conformance
```

### Generate Reports

Create Markdown and JSON reports:

```bash
./bin/conformance \
  --markdown-out=artifacts/report.md \
  --json-out=artifacts/report.json
```

### Generate CSV Artifacts

Export gap matrix and symbol evidence:

```bash
./bin/conformance --csv-out=artifacts
```

This creates:
- `artifacts/gap_matrix.csv` - Status of all RFC clauses
- `artifacts/symbol_evidence.csv` - Symbol locations and evidence

### History Tracking

Track conformance over time:

```bash
./bin/conformance \
  --history-file=artifacts/history.csv \
  --trend-markdown-out=artifacts/history_trend.md \
  --trend-window=15
```

Each run appends a timestamped entry to the history file. The trend markdown shows:
- Latest coverage percentage
- Moving average over window (default 15 runs)
- Coverage delta (change from previous run)
- Recent runs table (last 20 entries)

### Threshold Enforcement (Quality Gates)

Fail the build if conformance is below thresholds:

```bash
./bin/conformance \
  --min-coverage=85.0 \
  --max-missing-symbols=10 \
  --max-missing-tests=5
```

Exit codes:
- `0` - All checks passed
- `1` - Analysis error
- `2` - Threshold violation

Use `-1` to disable a threshold (default).

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--markdown-out` | string | - | Path to write markdown report |
| `--json-out` | string | - | Path to write JSON report |
| `--csv-out` | string | - | Directory to write CSV artifacts |
| `--history-file` | string | - | Path to history CSV (appends run data) |
| `--trend-markdown-out` | string | - | Path to write trend dashboard |
| `--trend-window` | int | 15 | Number of recent runs for moving average |
| `--min-coverage` | float | -1 | Minimum coverage percentage (0-100) |
| `--max-missing-symbols` | int | -1 | Maximum allowed missing symbols |
| `--max-missing-tests` | int | -1 | Maximum allowed missing tests |

## Output Examples

### Summary Output

```
🔍 GAuth RFC 0111/0115 Conformance Analyzer
============================================
✅ Analysis complete: 68.8% coverage (33/48 symbols)

📝 Generated files:
   - artifacts/report.md
   - artifacts/report.json
   - artifacts/gap_matrix.csv
   - artifacts/symbol_evidence.csv
   - artifacts/history.csv
   - artifacts/history_trend.md

✅ All checks passed
```

### Threshold Violation

```
🔍 GAuth RFC 0111/0115 Conformance Analyzer
============================================
✅ Analysis complete: 68.8% coverage (33/48 symbols)

❌ Threshold violations:
   - Coverage 68.8% below minimum 90.0%
   - 15 missing symbols exceeds maximum 10

[Exit code 2]
```

## CI/CD Integration

### GitHub Actions Example

```yaml
- name: Run Conformance Analysis
  run: |
    go run ./cmd/conformance \
      --markdown-out=artifacts/report.md \
      --json-out=artifacts/report.json \
      --csv-out=artifacts \
      --history-file=artifacts/history.csv \
      --trend-markdown-out=artifacts/history_trend.md \
      --min-coverage=85.0 \
      --max-missing-symbols=10

- name: Upload Artifacts
  uses: actions/upload-artifact@v3
  if: always()
  with:
    name: conformance-reports
    path: artifacts/
```

### Makefile Example

```makefile
.PHONY: conformance
conformance:
	@echo "Running conformance analysis..."
	@go run ./cmd/conformance \
		--markdown-out=artifacts/report.md \
		--json-out=artifacts/report.json \
		--csv-out=artifacts \
		--history-file=artifacts/history.csv \
		--trend-markdown-out=artifacts/history_trend.md

.PHONY: conformance-gate
conformance-gate:
	@echo "Running conformance with quality gates..."
	@go run ./cmd/conformance \
		--min-coverage=85.0 \
		--max-missing-symbols=10 \
		--max-missing-tests=5
```

## History CSV Format

The history CSV tracks conformance metrics over time:

```csv
timestamp,coverage,gap_missing,gap_partial,gap_implemented,missing_symbols,missing_tests
2025-11-06T20:58:32+01:00,68.75,19,16,8,15,3
```

Columns:
- `timestamp` - RFC3339 timestamp of the run
- `coverage` - Coverage percentage (0-100)
- `gap_missing` - Number of missing GAP items
- `gap_partial` - Number of partially implemented GAP items
- `gap_implemented` - Number of implemented GAP items
- `missing_symbols` - Number of missing required symbols
- `missing_tests` - Number of missing required tests

## Trend Dashboard

The trend markdown provides a visual dashboard of conformance over time:

```markdown
# Conformance Trend Dashboard

## Summary
Total Runs: 24

Coverage Latest: 68.75%
Coverage Moving Avg: 97.50%
Coverage Δ (latest-prev): 0.00%

Gap Missing Latest: 19 (Δ 0)
Gap Implemented Latest: 8 (Δ 0)

## Recent Runs
| Timestamp | Coverage | GapMissing | GapPartial | GapImplemented | MissingSymbols | MissingTests |
|-----------|----------|-----------|-----------|---------------|---------------|-------------|
| 2025-11-06T20:58:20+01:00 | 68.75 | 19 | 16 | 8 | 15 | 3 |
| 2025-11-06T20:58:32+01:00 | 68.75 | 19 | 16 | 8 | 15 | 3 |
```

## Implementation Details

The tool:

1. **Loads Clause Mapping**: Reads `conformance/clause_map.json` to get RFC clause definitions
2. **Builds Symbol Index**: Scans the codebase using Go AST to find all exported symbols
3. **Analyzes Coverage**: Matches required symbols from clause_map against the symbol index
4. **Collects Evidence**: Records file:line locations for all found symbols
5. **Generates Reports**: Formats results in requested output formats
6. **Tracks History**: Appends metrics to CSV and generates trend dashboards
7. **Enforces Gates**: Checks thresholds and exits with appropriate code

## Related Documentation

- [Conformance Harness README](../../conformance/README.md)
- [Clause Mapping](../../conformance/clause_map.json)
- [GAP Matrix](../../docs/conformance_gaps.json)

## License

See [LICENSE](../../LICENSE) file in the repository root.
