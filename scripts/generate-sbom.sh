#!/usr/bin/env bash
set -euo pipefail

# Demo SBOM generation script using Syft (beta demonstration).
# Requires Syft: https://github.com/anchore/syft
# Install (macOS/Homebrew): brew install syft
# Or (curl): curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin
# Usage:
#   scripts/generate-sbom.sh [IMAGE_NAME[:TAG]]
# If IMAGE is provided, generates SBOM for container image, otherwise for source directory.

IMAGE="${1:-}"
OUT_DIR="sbom"
mkdir -p "$OUT_DIR"
TS=$(date +%Y%m%d-%H%M%S)

if ! command -v syft >/dev/null 2>&1; then
  echo "syft not installed. See header comments for installation instructions." >&2
  exit 2
fi

if [ -n "$IMAGE" ]; then
  echo "Generating image SBOM for $IMAGE ..."
  syft "$IMAGE" -o json > "$OUT_DIR/sbom-image-$TS.json"
  syft "$IMAGE" -o spdx-json > "$OUT_DIR/sbom-image-$TS.spdx.json"
else
  echo "Generating source SBOM for current module ..."
  syft dir:. -o json > "$OUT_DIR/sbom-source-$TS.json"
  syft dir:. -o spdx-json > "$OUT_DIR/sbom-source-$TS.spdx.json"
fi

echo "SBOM artifacts written to $OUT_DIR/ (json + spdx-json)"
