#!/bin/bash
# RFC Standards Rename Script
# Renames RFC-111 to AgentAuth-RFC-001 and RFC-115 to AgentAuth-RFC-002
# This eliminates namespace collision with IETF standards

set -e

echo "=========================================="
echo "AgentAuth RFC Standards Rename Script"
echo "=========================================="
echo ""
echo "Purpose: Rename internal RFC references to avoid IETF collision"
echo "- RFC-111 → AgentAuth-RFC-001"
echo "- RFC-115 → AgentAuth-RFC-002"
echo ""
echo "Affected:"
echo "- Package names (pkg/rfc0111 → pkg/gauth_rfc_001)"
echo "- Import statements"
echo "- Documentation references"
echo "- Example code"
echo ""

# Confirmation prompt
read -p "Proceed with rename? (y/n): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
fi

echo ""
echo "Step 1: Renaming package declarations..."
echo "----------------------------------------"

# Rename package declarations in Go files
find . -type f -name "*.go" -not -path "*/vendor/*" -not -path "*/.git/*" -exec sed -i '' 's/package rfc0111/package gauth_rfc_001/g' {} +
echo "✓ Updated package rfc0111 → gauth_rfc_001"

# Note: rfc115 doesn't appear to exist in current codebase
# find . -type f -name "*.go" -not -path "*/vendor/*" -not -path "*/.git/*" -exec sed -i '' 's/package rfc115/package gauth_rfc_002/g' {} +
# echo "✓ Updated package rfc115 → gauth_rfc_002"

echo ""
echo "Step 2: Updating import statements..."
echo "--------------------------------------"

# Update import statements
find . -type f -name "*.go" -not -path "*/vendor/*" -not -path "*/.git/*" -exec sed -i '' 's|"github.com/mauriciomferz/Gauth_go/pkg/rfc0111"|"github.com/mauriciomferz/Gauth_go/pkg/gauth_rfc_001"|g' {} +
echo "✓ Updated imports: pkg/rfc0111 → pkg/gauth_rfc_001"

# find . -type f -name "*.go" -not -path "*/vendor/*" -not -path "*/.git/*" -exec sed -i '' 's|"github.com/mauriciomferz/Gauth_go/pkg/rfc115"|"github.com/mauriciomferz/Gauth_go/pkg/gauth_rfc_002"|g' {} +
# echo "✓ Updated imports: pkg/rfc115 → pkg/gauth_rfc_002"

echo ""
echo "Step 3: Updating documentation..."
echo "----------------------------------"

# Update documentation references (preserve historical IETF context)
find . -type f \( -name "*.md" -o -name "*.txt" \) -not -path "*/vendor/*" -not -path "*/.git/*" -exec sed -i '' 's/RFC 111/AgentAuth-RFC-001 (formerly RFC 111)/g' {} +
find . -type f \( -name "*.md" -o -name "*.txt" \) -not -path "*/vendor/*" -not -path "*/.git/*" -exec sed -i '' 's/RFC-111/AgentAuth-RFC-001/g' {} +
find . -type f \( -name "*.md" -o -name "*.txt" \) -not -path "*/vendor/*" -not -path "*/.git/*" -exec sed -i '' 's/RFC 115/AgentAuth-RFC-002 (formerly RFC 115)/g' {} +
find . -type f \( -name "*.md" -o -name "*.txt" \) -not -path "*/vendor/*" -not -path "*/.git/*" -exec sed -i '' 's/RFC-115/AgentAuth-RFC-002/g' {} +
echo "✓ Updated documentation references"

# Update JSON examples
find . -type f -name "*.json" -not -path "*/vendor/*" -not -path "*/.git/*" -exec sed -i '' 's/"rfc111/"gauth_rfc_001/g' {} +
find . -type f -name "*.json" -not -path "*/vendor/*" -not -path "*/.git/*" -exec sed -i '' 's/"rfc115/"gauth_rfc_002/g' {} +
echo "✓ Updated JSON examples"

echo ""
echo "Step 4: Renaming directories..."
echo "--------------------------------"

# Rename directories (if they exist)
if [ -d "pkg/rfc0111" ]; then
    mv pkg/rfc0111 pkg/gauth_rfc_001
    echo "✓ Renamed directory: pkg/rfc0111 → pkg/gauth_rfc_001"
else
    echo "⚠  Directory pkg/rfc0111 not found (may already be renamed)"
fi

if [ -d "pkg/rfc115" ]; then
    mv pkg/rfc115 pkg/gauth_rfc_002
    echo "✓ Renamed directory: pkg/rfc115 → pkg/gauth_rfc_002"
else
    echo "⚠  Directory pkg/rfc115 not found (may not exist)"
fi

echo ""
echo "Step 5: Updating .gitignore..."
echo "-------------------------------"

# Update gitignore entries
if [ -f ".gitignore" ]; then
    sed -i '' 's/rfc111/gauth_rfc_001/g' .gitignore
    sed -i '' 's/rfc115/gauth_rfc_002/g' .gitignore
    echo "✓ Updated .gitignore"
fi

echo ""
echo "Step 6: Running go mod tidy..."
echo "-------------------------------"

go mod tidy
echo "✓ Updated Go module dependencies"

echo ""
echo "Step 7: Building project to verify changes..."
echo "----------------------------------------------"

if go build ./...; then
    echo "✓ Build successful"
else
    echo "✗ Build failed - please review errors above"
    exit 1
fi

echo ""
echo "=========================================="
echo "✅ RFC Rename Completed Successfully"
echo "=========================================="
echo ""
echo "Summary of changes:"
echo "- Package rfc0111 → gauth_rfc_001"
echo "- All imports updated"
echo "- Documentation updated with historical context"
echo "- Directory structure renamed"
echo "- Project builds successfully"
echo ""
echo "Next steps:"
echo "1. Review changes: git diff"
echo "2. Run tests: go test ./..."
echo "3. Update external documentation referencing RFC-111/115"
echo "4. Commit changes: git add . && git commit -m 'Rename RFC-111/115 to AgentAuth-RFC-001/002'"
echo ""
echo "Note: This rename eliminates collision with IETF standards:"
echo "- IETF RFC 111: Network Control Protocol (1971)"
echo "- IETF RFC 115: Network Information Center Procedures (1971)"
echo ""
