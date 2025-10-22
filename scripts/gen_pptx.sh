#!/usr/bin/env bash
set -euo pipefail

# gen_pptx.sh
# Generate PDF and PPTX presentation from deck markdown, converting Mermaid diagrams to PNG.
# Usage: ./scripts/gen_pptx.sh [options] [deck_markdown] [out_basename]
# Options:
#   --no-pdf                Skip PDF generation even if engine available
#   --engine=<name>         Force PDF engine (tectonic|weasyprint|wkhtmltopdf|pdflatex)
#   --reference-pptx=<file> Use custom PPTX reference template
#   --help                  Show help
# Default deck_markdown: docs/presentations/GAuth_Executive_Tech_Business_Deck.md
# Default out_basename: GAuth_Deck

DECK_MD=""
OUT_BASE=""
FORCE_ENGINE=""
NO_PDF=0
REFERENCE_PPTX=""
PPTX_ARGS=()

show_help(){
  cat <<'EOF'
gen_pptx.sh - Generate PPTX (and PDF) from GAuth markdown deck.

Usage: ./scripts/gen_pptx.sh [options] [deck_markdown] [out_basename]

Options:
  --no-pdf                Skip PDF output
  --engine=<name>         Force PDF engine (tectonic|weasyprint|wkhtmltopdf|pdflatex)
  --reference-pptx=<file> Use custom PPTX reference template for theming
  --help                  Show this help

Examples:
  ./scripts/gen_pptx.sh
  ./scripts/gen_pptx.sh --no-pdf docs/presentations/GAuth_Executive_Tech_Business_Deck.md GAuth_NoPDF
  ./scripts/gen_pptx.sh --engine=tectonic --reference-pptx=branding.pptx
EOF
}

ARGS=()
for arg in "$@"; do
  case "$arg" in
    --no-pdf) NO_PDF=1 ;;
    --engine=*) FORCE_ENGINE="${arg#*=}" ;;
    --reference-pptx=*) REFERENCE_PPTX="${arg#*=}" ;;
    --help) show_help; exit 0 ;;
    *) ARGS+=("$arg") ;;
  esac
done

DECK_MD=${ARGS[0]:-docs/presentations/GAuth_Executive_Tech_Business_Deck.md}
OUT_BASE=${ARGS[1]:-GAuth_Deck}

if [[ ! -f "$DECK_MD" ]]; then
  echo "Error: Deck markdown '$DECK_MD' not found" >&2
  exit 1
fi

command -v pandoc >/dev/null 2>&1 || { echo "Error: pandoc not found. Install via 'brew install pandoc'" >&2; exit 1; }
command -v mmdc >/dev/null 2>&1 || {
  echo "Mermaid CLI (mmdc) not found. Attempting global install..." >&2
  if command -v npm >/dev/null 2>&1; then
    npm install -g @mermaid-js/mermaid-cli || { echo "Failed to install mermaid-cli. Install manually." >&2; exit 1; }
  else
    echo "npm not available; cannot auto-install mermaid-cli." >&2
    exit 1
  fi
}

WORKDIR=$(mktemp -d)
trap 'rm -rf "$WORKDIR"' EXIT

DIAGRAM_DIR=docs/presentations/diagrams
mkdir -p "$DIAGRAM_DIR"

# Create a temp copy without front matter to simplify pandoc parsing
STRIPPED_MD="$WORKDIR/stripped.md"
python3 - "$DECK_MD" > "$STRIPPED_MD" <<'PYFRONT'
import sys
path=sys.argv[1]
lines=open(path,'r').read().splitlines()
if lines and lines[0].strip()=="---":
  # find closing ---
  for i in range(1,len(lines)):
    if lines[i].strip()=="---":
      lines=lines[i+1:]
      break
open(sys.stdout.fileno(),'w').write('\n'.join(lines))
PYFRONT

# Extract mermaid code fences and convert to PNG from stripped file
# We number diagrams sequentially.

awk -v WDIR="$WORKDIR" 'BEGIN{c=0; inside=0} {
  if ($0 ~ /^```mermaid/) { inside=1; c++; fname=sprintf("%s/diagram_%02d.mmd", WDIR, c); next }
  if (inside && $0 ~ /^```$/) { inside=0; next }
  if (inside) { print >> fname }
} END{ }' "$STRIPPED_MD"

diagram_files=$(ls "$WORKDIR"/diagram_*.mmd 2>/dev/null || true)
if [[ -z "$diagram_files" ]]; then
  diagram_count=0
else
  diagram_count=$(echo "$diagram_files" | wc -l | tr -d ' ')
fi
echo "Found $diagram_count mermaid diagrams"

if [[ $diagram_count -gt 0 ]]; then
  for f in "$WORKDIR"/diagram_*.mmd; do
    base=$(basename "$f" .mmd)
    out_png="$DIAGRAM_DIR/${base}.png"
    echo "Rendering $f -> $out_png"
    mmdc -i "$f" -o "$out_png" || { echo "Mermaid render failed for $f" >&2; exit 1; }
  done
fi

# Create a processed markdown with image references replacing mermaid fences
PROCESSED_MD="$WORKDIR/processed.md"
python3 - "$STRIPPED_MD" "$DIAGRAM_DIR" > "$PROCESSED_MD" <<'PYCODE'
import sys, re, os
deck = sys.argv[1]
diag_dir = sys.argv[2]
content = open(deck,'r').read().splitlines()
out=[]
in_mermaid=False
buf=[]
idx=0
for line in content:
    if line.startswith('```mermaid'):
        in_mermaid=True
        buf=[]
        continue
    if in_mermaid and line.strip()=='```':
        in_mermaid=False
        idx+=1
        png=f"diagram_{idx:02d}.png"
        out.append(f"![Diagram {idx}]({os.path.join(diag_dir, png)})")
        continue
    if in_mermaid:
        buf.append(line)
        continue
    out.append(line)
text='\n'.join(out)
# Remove front-matter block starting with --- if present to avoid pandoc YAML parse issues.
def strip_front_matter(t: str) -> str:
  lines=t.split('\n')
  if not lines or lines[0].strip()!='---':
    return t
  # find closing ---
  for i in range(1,len(lines)):
    if lines[i].strip()=='---':
      return '\n'.join(lines[i+1:])
  return t  # malformed, return as-is
text=strip_front_matter(text)
while text.startswith('---'):
  # handle recursive or duplicated separators
  new=strip_front_matter(text)
  if new==text:
    break
  text=new
open(sys.stdout.fileno(),'w').write(text)
PYCODE

## Force-remove any residual YAML front matter block at start of processed file
if [[ -s "$PROCESSED_MD" ]]; then
  if head -n1 "$PROCESSED_MD" | grep -q '^---$'; then
    # delete until next --- line
    awk 'BEGIN{skip=0} { if(NR==1 && $0=="---") {skip=1; next} if(skip && $0=="---") {skip=0; next} if(!skip){print}}' "$PROCESSED_MD" > "$PROCESSED_MD.tmp" && mv "$PROCESSED_MD.tmp" "$PROCESSED_MD"
  fi
  # If still starts with --- remove first line
  if head -n1 "$PROCESSED_MD" | grep -q '^---$'; then
    tail -n +2 "$PROCESSED_MD" > "$PROCESSED_MD.tmp" && mv "$PROCESSED_MD.tmp" "$PROCESSED_MD"
  fi
  # Prepend a synthetic title to ensure not empty / not front matter
  { echo "# GAuth Presentation"; echo; cat "$PROCESSED_MD"; } > "$PROCESSED_MD.tmp" && mv "$PROCESSED_MD.tmp" "$PROCESSED_MD"
fi

# Replace remaining slide separators '---' with non-metadata marker '***'
sed -i '' 's/^---$/***/g' "$PROCESSED_MD" || true

echo "Generating PDF and PPTX from processed markdown"
echo "--- Processed markdown first 20 lines (debug) ---"
head -n 20 "$PROCESSED_MD" || true
echo "--- End head ---"
echo "Running pandoc without yaml_metadata_block extension"
PDF_ENGINE=""
if [[ "$NO_PDF" -eq 0 ]]; then
  if [[ -n "$FORCE_ENGINE" ]]; then
    PDF_ENGINE="$FORCE_ENGINE"
  else
    if command -v wkhtmltopdf >/dev/null 2>&1; then
      PDF_ENGINE="wkhtmltopdf"
    elif command -v weasyprint >/dev/null 2>&1; then
      PDF_ENGINE="weasyprint"
    elif command -v tectonic >/dev/null 2>&1; then
      PDF_ENGINE="tectonic"
    elif command -v pdflatex >/dev/null 2>&1; then
      PDF_ENGINE="pdflatex"
    fi
  fi
  if [[ -n "$PDF_ENGINE" ]]; then
    echo "Using PDF engine: $PDF_ENGINE"
    pandoc -f markdown "$PROCESSED_MD" --pdf-engine="$PDF_ENGINE" -o "${OUT_BASE}.pdf" || echo "PDF generation failed with $PDF_ENGINE, skipping PDF."
  else
    echo "No PDF engine selected or available; skipping PDF output. Use --engine=<name> or install tectonic."
  fi
else
  echo "--no-pdf flag set; skipping PDF generation."
fi

if [[ -n "$REFERENCE_PPTX" ]]; then
  if [[ -f "$REFERENCE_PPTX" ]]; then
    PPTX_ARGS+=("--reference-doc=$REFERENCE_PPTX")
  else
    echo "Warning: reference PPTX '$REFERENCE_PPTX' not found; ignoring." >&2
  fi
fi
set +u
pandoc -f markdown "$PROCESSED_MD" "${PPTX_ARGS[@]}" -o "${OUT_BASE}.pptx"
set -u

echo "Output files: ${OUT_BASE}.pdf, ${OUT_BASE}.pptx"
echo "Diagram images in: $DIAGRAM_DIR"
echo "Done." 
