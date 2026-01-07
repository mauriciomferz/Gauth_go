#!/usr/bin/env bash
#
# API Documentation PDF Generator
# Generates PDFs from API markdown documentation.
#

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DOCS_DIR="$ROOT_DIR/docs"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  API Documentation PDF Generator${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

if ! command -v pandoc &> /dev/null; then
  echo -e "${RED}Error: pandoc is not installed${NC}"
  echo -e "${YELLOW}Install with: brew install pandoc${NC}"
  exit 1
fi

if [[ ! -d "$DOCS_DIR" ]]; then
  echo -e "${RED}Error: docs directory not found: $DOCS_DIR${NC}"
  exit 1
fi

# Check if LaTeX is available (unless HTML mode)
if [[ "${1:-}" != "--html" ]]; then
  if ! command -v pdflatex &> /dev/null && ! command -v xelatex &> /dev/null; then
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}LaTeX Engine Not Found${NC}"
    echo -e "${YELLOW}========================================${NC}"
    echo ""
    echo -e "${YELLOW}To generate PDFs, install a LaTeX distribution:${NC}"
    echo ""
    echo -e "${GREEN}Option 1: BasicTeX (Recommended)${NC}"
    echo -e "  brew install --cask basictex"
    echo ""
    echo -e "${GREEN}Option 2: Full MacTeX${NC}"
    echo -e "  brew install --cask mactex"
    echo ""
    echo -e "${GREEN}Option 3: Generate HTML instead${NC}"
    echo -e "  scripts/legacy-root/generate_api_docs_pdf.sh --html"
    echo ""
    exit 1
  fi
fi

# Engine selection
if [[ "${1:-}" == "--html" ]]; then
  PDF_ENGINE="html"
elif command -v xelatex &> /dev/null; then
  PDF_ENGINE="xelatex"
else
  PDF_ENGINE="pdflatex"
fi

cd "$DOCS_DIR"

if [[ "$PDF_ENGINE" == "html" ]]; then
  echo -e "${BLUE}Generating HTML files...${NC}"
  echo ""

  for file in API_EXAMPLES.md API_REFERENCE.md API_VERSIONING.md; do
    basename="${file%.md}"
    echo -e "${GREEN}→${NC} Converting $file to HTML..."
    pandoc "$file" \
      -s \
      -o "${basename}.html" \
      --toc \
      --toc-depth=3 \
      --css=https://cdnjs.cloudflare.com/ajax/libs/github-markdown-css/5.2.0/github-markdown.min.css \
      --metadata title="API Docs - $basename" \
      2>&1 && echo -e "  ${GREEN}✓${NC} ${basename}.html created" || echo -e "  ${RED}✗${NC} Failed"
  done

  echo ""
  echo -e "${GREEN}✓ HTML Generation Complete!${NC}"
  exit 0
fi

echo -e "${BLUE}Step 1: Generating individual PDFs...${NC}"
echo ""

pandoc API_EXAMPLES.md \
  -o API_EXAMPLES.pdf \
  --pdf-engine="$PDF_ENGINE" \
  --toc \
  --toc-depth=3 \
  --number-sections \
  -V geometry:margin=1in \
  -V fontsize=11pt \
  --metadata title="API Examples" \
  --metadata date="November 17, 2025" \
  2>&1

echo -e "${BLUE}Step 2: Generating combined PDF...${NC}"
echo ""

pandoc \
  API_EXAMPLES.md \
  API_REFERENCE.md \
  API_VERSIONING.md \
  -o API_Complete_Documentation.pdf \
  --pdf-engine="$PDF_ENGINE" \
  --toc \
  --toc-depth=3 \
  --number-sections \
  -V documentclass=report \
  -V geometry:margin=1in \
  -V fontsize=11pt \
  --metadata title="Complete API Documentation" \
  --metadata date="November 17, 2025" \
  2>&1

echo ""
echo -e "${GREEN}✓ Documentation generation complete.${NC}"
echo -e "${YELLOW}Output directory:${NC} $DOCS_DIR"
echo -e "${YELLOW}PDF Engine Used:${NC} $PDF_ENGINE"
echo ""
