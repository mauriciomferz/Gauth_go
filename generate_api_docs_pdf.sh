#!/bin/bash
#
# AgentAuth API Documentation PDF Generator
# Generates professional PDFs from API markdown documentation
#
# Usage: ./generate_api_docs_pdf.sh
#

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  AgentAuth API Documentation PDF Generator${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if pandoc is installed
if ! command -v pandoc &> /dev/null; then
    echo -e "${RED}Error: pandoc is not installed${NC}"
    echo -e "${YELLOW}Install with: brew install pandoc${NC}"
    exit 1
fi

# Check if LaTeX is available
if ! command -v pdflatex &> /dev/null && ! command -v xelatex &> /dev/null; then
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}LaTeX Engine Not Found${NC}"
    echo -e "${YELLOW}========================================${NC}"
    echo ""
    echo -e "${YELLOW}To generate PDFs, you need to install a LaTeX distribution:${NC}"
    echo ""
    echo -e "${GREEN}Option 1: BasicTeX (Recommended - Small, 100MB)${NC}"
    echo -e "  brew install --cask basictex"
    echo -e "  # After install, run:"
    echo -e "  sudo tlmgr update --self"
    echo -e "  sudo tlmgr install framed fvextra footnotebackref pagecolor mdframed needspace"
    echo ""
    echo -e "${GREEN}Option 2: Full MacTeX (Complete - 4GB)${NC}"
    echo -e "  brew install --cask mactex"
    echo ""
    echo -e "${GREEN}Option 3: Generate HTML instead (No LaTeX needed)${NC}"
    echo -e "  ./generate_api_docs_pdf.sh --html"
    echo ""
    exit 1
fi

# Check for HTML mode
if [ "$1" == "--html" ]; then
    PDF_ENGINE="html"
elif command -v xelatex &> /dev/null; then
    PDF_ENGINE="xelatex"
else
    PDF_ENGINE="pdflatex"
fi

# Navigate to docs directory
cd "$(dirname "$0")/docs"

if [ "$PDF_ENGINE" == "html" ]; then
    echo -e "${BLUE}Generating HTML files...${NC}"
    echo ""
    
    # Generate HTML files
    for file in API_EXAMPLES.md API_REFERENCE.md API_VERSIONING.md; do
        basename="${file%.md}"
        echo -e "${GREEN}→${NC} Converting $file to HTML..."
        pandoc "$file" \
          -s \
          -o "${basename}.html" \
          --toc \
          --toc-depth=3 \
          --css=https://cdnjs.cloudflare.com/ajax/libs/github-markdown-css/5.2.0/github-markdown.min.css \
          --metadata title="AgentAuth - $basename" \
          2>&1 && echo -e "  ${GREEN}✓${NC} ${basename}.html created" || echo -e "  ${RED}✗${NC} Failed"
    done
    
    echo ""
    echo -e "${GREEN}✓ HTML Generation Complete!${NC}"
    echo -e "${YELLOW}Generated files:${NC}"
    echo -e "  • API_EXAMPLES.html"
    echo -e "  • API_REFERENCE.html"
    echo -e "  • API_VERSIONING.html"
    echo ""
    echo -e "${BLUE}To convert HTML to PDF, open in browser and use Print → Save as PDF${NC}"
    exit 0
fi

echo -e "${BLUE}Step 1: Generating individual PDFs...${NC}"
echo ""

# Generate API_EXAMPLES.pdf
echo -e "${GREEN}→${NC} Converting API_EXAMPLES.md to PDF..."
pandoc API_EXAMPLES.md \
  -o API_EXAMPLES.pdf \
  --pdf-engine=$PDF_ENGINE \
  --toc \
  --toc-depth=3 \
  --number-sections \
  -V geometry:margin=1in \
  -V fontsize=11pt \
  -V colorlinks=true \
  -V linkcolor=blue \
  -V urlcolor=blue \
  -V toccolor=black \
  --metadata title="AgentAuth API Examples" \
  --metadata author="AgentAuth Community" \
  --metadata date="November 17, 2025" \
  2>&1

if [ $? -eq 0 ]; then
    echo -e "  ${GREEN}✓${NC} API_EXAMPLES.pdf created"
else
    echo -e "  ${RED}✗${NC} Failed to create API_EXAMPLES.pdf"
fi

# Generate API_REFERENCE.pdf
echo -e "${GREEN}→${NC} Converting API_REFERENCE.md to PDF..."
pandoc API_REFERENCE.md \
  -o API_REFERENCE.pdf \
  --pdf-engine=$PDF_ENGINE \
  --toc \
  --toc-depth=3 \
  --number-sections \
  -V geometry:margin=1in \
  -V fontsize=11pt \
  -V colorlinks=true \
  -V linkcolor=blue \
  -V urlcolor=blue \
  -V toccolor=black \
  --metadata title="AgentAuth API Reference" \
  --metadata author="AgentAuth Community" \
  --metadata date="November 17, 2025" \
  2>&1

if [ $? -eq 0 ]; then
    echo -e "  ${GREEN}✓${NC} API_REFERENCE.pdf created"
else
    echo -e "  ${RED}✗${NC} Failed to create API_REFERENCE.pdf"
fi

# Generate API_VERSIONING.pdf
echo -e "${GREEN}→${NC} Converting API_VERSIONING.md to PDF..."
pandoc API_VERSIONING.md \
  -o API_VERSIONING.pdf \
  --pdf-engine=$PDF_ENGINE \
  --toc \
  --toc-depth=3 \
  --number-sections \
  -V geometry:margin=1in \
  -V fontsize=11pt \
  -V colorlinks=true \
  -V linkcolor=blue \
  -V urlcolor=blue \
  -V toccolor=black \
  --metadata title="AgentAuth API Versioning" \
  --metadata author="AgentAuth Community" \
  --metadata date="November 17, 2025" \
  2>&1

if [ $? -eq 0 ]; then
    echo -e "  ${GREEN}✓${NC} API_VERSIONING.pdf created"
else
    echo -e "  ${RED}✗${NC} Failed to create API_VERSIONING.pdf"
fi

echo ""
echo -e "${BLUE}Step 2: Generating combined PDF...${NC}"
echo ""

# Generate combined PDF
echo -e "${GREEN}→${NC} Creating AgentAuth_API_Complete_Documentation.pdf..."
pandoc \
  API_EXAMPLES.md \
  API_REFERENCE.md \
  API_VERSIONING.md \
  -o AgentAuth_API_Complete_Documentation.pdf \
  --pdf-engine=$PDF_ENGINE \
  --toc \
  --toc-depth=3 \
  --number-sections \
  -V documentclass=report \
  -V geometry:margin=1in \
  -V fontsize=11pt \
  -V colorlinks=true \
  -V linkcolor=blue \
  -V urlcolor=blue \
  -V toccolor=black \
  --metadata title="AgentAuth 1.0 - Complete API Documentation" \
  --metadata subtitle="RFC-0111/0115 Compliant Authorization Framework" \
  --metadata author="AgentAuth Community" \
  --metadata date="November 17, 2025" \
  2>&1

if [ $? -eq 0 ]; then
    echo -e "  ${GREEN}✓${NC} AgentAuth_API_Complete_Documentation.pdf created"
else
    echo -e "  ${RED}✗${NC} Failed to create combined PDF"
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}✓ PDF Generation Complete!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${YELLOW}Generated files in docs/ directory:${NC}"
echo -e "  • API_EXAMPLES.pdf"
echo -e "  • API_REFERENCE.pdf"
echo -e "  • API_VERSIONING.pdf"
echo -e "  • AgentAuth_API_Complete_Documentation.pdf (combined)"
echo ""
echo -e "${BLUE}PDF Engine Used:${NC} $PDF_ENGINE"
echo ""
