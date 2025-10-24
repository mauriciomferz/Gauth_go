#!/usr/bin/env bash
set -euo pipefail
# regenerate_compliance_matrix.sh
# Regenerates conformance artifacts and updates compliance matrix snapshot.
# Usage: ./scripts/regenerate_compliance_matrix.sh

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
ARTIFACT_DIR="$ROOT_DIR/artifacts"
REPORT_JSON="$ROOT_DIR/conformance/report.json"
MATRIX_SRC="$ROOT_DIR/docs/rfc0111_compliance_matrix.md"
TIMESTAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

mkdir -p "$ARTIFACT_DIR"

echo "[INFO] Running conformance analyzer..."
go run "$ROOT_DIR/cmd/conformance" \
  --json-out="$REPORT_JSON" \
  --markdown-out="$ARTIFACT_DIR/report.md" || { echo "[ERROR] Conformance run failed"; exit 1; }

if [[ -f "$MATRIX_SRC" ]]; then
  echo "[INFO] Copying compliance matrix snapshot..."
  cp "$MATRIX_SRC" "$ARTIFACT_DIR/rfc0111_compliance_matrix_snapshot.md"
  echo "\n> Snapshot Timestamp: $TIMESTAMP" >> "$ARTIFACT_DIR/rfc0111_compliance_matrix_snapshot.md"
fi

# Optional: append summary line to history trend if present
if [[ -f "$ARTIFACT_DIR/history_gap_matrix_status.jsonl" ]]; then
  echo "[INFO] Appending summary to history log..."
  COVERAGE=$(jq -r '.summary.coverage_percent' "$REPORT_JSON" 2>/dev/null || echo "NA")
  echo "{\"ts\":\"$TIMESTAMP\",\"coverage\":\"$COVERAGE\"}" >> "$ARTIFACT_DIR/history_conformance_runs.jsonl"
fi

echo "[INFO] Done. Artifacts written to $ARTIFACT_DIR"
