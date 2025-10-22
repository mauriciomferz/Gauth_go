#!/usr/bin/env bash
set -euo pipefail

# Scan for JavaScript patterns that would require loosening CSP (unsafe-eval/inline string execution)
# Exits non-zero if any forbidden pattern is found.
# Safe in CI: produces deterministic output.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

patterns=(
  "eval("
  "new Function"
  "setTimeout(\""   # setTimeout("...
  "setTimeout('"     # setTimeout('...
  "setInterval(\""  # setInterval("...
  "setInterval('"    # setInterval('...
)

# Assemble grep arguments
fail=0
report_file="/tmp/unsafe_js_scan_$$.txt"
: > "$report_file"

for pat in "${patterns[@]}"; do
  if grep -RFn --include='*.js' --color=never -e "$pat" . > /tmp/grep_out_$$ 2>/dev/null; then
    echo "[FOUND] Pattern: $pat" | tee -a "$report_file"
    cat /tmp/grep_out_$$ | tee -a "$report_file"
    echo | tee -a "$report_file"
    fail=1
  fi
  rm -f /tmp/grep_out_$$
done

if [[ $fail -eq 1 ]]; then
  echo "\nFAIL: Unsafe JavaScript execution patterns detected. Refactor instead of relaxing CSP." | tee -a "$report_file"
  exit 1
fi

echo "Scan complete: no unsafe JavaScript execution patterns detected." | tee -a "$report_file"
