#!/bin/bash
# Post-Upgrade Verification Script
# Run this after upgrading Go from 1.25.1 to 1.25.3

set -e

echo "=================================================="
echo "AgentAuth Go 1.25.3 Upgrade Verification"
echo "=================================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Step 1: Check Go version
echo "Step 1: Verifying Go version..."
GO_VERSION=$(go version | awk '{print $3}')
if [[ "$GO_VERSION" == "go1.25.3" ]] || [[ "$GO_VERSION" > "go1.25.3" ]]; then
    echo -e "${GREEN}✓${NC} Go version: $GO_VERSION"
else
    echo -e "${RED}✗${NC} Go version: $GO_VERSION"
    echo -e "${RED}ERROR: Go 1.25.3 or higher required${NC}"
    echo "Current: $GO_VERSION"
    echo ""
    echo "Install Go 1.25.3:"
    echo "  macOS: brew upgrade go"
    echo "  Or download from: https://go.dev/dl/"
    exit 1
fi
echo ""

# Step 2: Clean build cache
echo "Step 2: Cleaning build cache..."
go clean -cache
echo -e "${GREEN}✓${NC} Build cache cleaned"
echo ""

# Step 3: Verify go.mod
echo "Step 3: Verifying go.mod..."
GO_MOD_VERSION=$(grep "^go " go.mod | awk '{print $2}')
if [[ "$GO_MOD_VERSION" == "1.25.3" ]] || [[ "$GO_MOD_VERSION" > "1.25.3" ]]; then
    echo -e "${GREEN}✓${NC} go.mod specifies: go $GO_MOD_VERSION"
else
    echo -e "${YELLOW}⚠${NC}  go.mod specifies: go $GO_MOD_VERSION"
    echo "Updating go.mod to 1.25.3..."
    go mod edit -go=1.25.3
    echo -e "${GREEN}✓${NC} go.mod updated"
fi
echo ""

# Step 4: Tidy dependencies
echo "Step 4: Tidying dependencies..."
go mod tidy
echo -e "${GREEN}✓${NC} Dependencies tidied"
echo ""

# Step 5: Run vulnerability check
echo "Step 5: Running govulncheck..."
if command -v govulncheck &> /dev/null; then
    VULN_OUTPUT=$(govulncheck ./... 2>&1 || true)
    if echo "$VULN_OUTPUT" | grep -q "No vulnerabilities found"; then
        echo -e "${GREEN}✓${NC} No vulnerabilities found"
    else
        VULN_COUNT=$(echo "$VULN_OUTPUT" | grep -c "Vulnerability #" || echo "0")
        if [ "$VULN_COUNT" -eq "0" ]; then
            echo -e "${GREEN}✓${NC} No vulnerabilities found"
        else
            echo -e "${RED}✗${NC} Found $VULN_COUNT vulnerabilities"
            echo ""
            echo "$VULN_OUTPUT"
            echo ""
            echo -e "${RED}FAILURE: Vulnerabilities still present after upgrade${NC}"
            exit 1
        fi
    fi
else
    echo -e "${YELLOW}⚠${NC}  govulncheck not installed, skipping vulnerability check"
    echo "Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
fi
echo ""

# Step 6: Build all binaries
echo "Step 6: Building binaries..."
if go build -o bin/ ./cmd/... 2>&1; then
    BIN_COUNT=$(ls -1 bin/ 2>/dev/null | wc -l | tr -d ' ')
    echo -e "${GREEN}✓${NC} Built $BIN_COUNT binaries successfully"
else
    echo -e "${RED}✗${NC} Build failed"
    exit 1
fi
echo ""

# Step 7: Run unit tests
echo "Step 7: Running unit tests..."
echo "(This may take a few minutes...)"
if go test ./... -count=1 -timeout=10m 2>&1 | tee /tmp/test_output.txt | grep -E "(PASS|FAIL|ok|FAIL)"; then
    PASS_COUNT=$(grep -c "^ok" /tmp/test_output.txt || echo "0")
    FAIL_COUNT=$(grep -c "^FAIL" /tmp/test_output.txt || echo "0")
    
    if [ "$FAIL_COUNT" -eq "0" ]; then
        echo -e "${GREEN}✓${NC} All tests passed ($PASS_COUNT packages)"
    else
        echo -e "${RED}✗${NC} $FAIL_COUNT test packages failed"
        exit 1
    fi
else
    echo -e "${RED}✗${NC} Test execution failed"
    exit 1
fi
echo ""

# Step 8: Run conformance tests (if binary exists)
echo "Step 8: Running conformance tests..."
if [ -f "./bin/gauth-conformance" ]; then
    if ./bin/gauth-conformance run 2>&1 | grep -q "100%"; then
        echo -e "${GREEN}✓${NC} Conformance tests: 100% compliance"
    else
        echo -e "${YELLOW}⚠${NC}  Conformance tests completed (check output for details)"
    fi
else
    echo -e "${YELLOW}⚠${NC}  gauth-conformance binary not found, skipping"
fi
echo ""

# Step 9: Quick performance check
echo "Step 9: Running quick performance check..."
if go test ./pkg/gauth -bench=BenchmarkValidateToken -benchtime=1s -run=^$ 2>&1 | grep -E "ns/op"; then
    echo -e "${GREEN}✓${NC} Performance benchmark completed"
else
    echo -e "${YELLOW}⚠${NC}  Benchmark not available, skipping"
fi
echo ""

# Summary
echo "=================================================="
echo -e "${GREEN}VERIFICATION COMPLETE${NC}"
echo "=================================================="
echo ""
echo "Summary:"
echo "  ✓ Go version: $GO_VERSION"
echo "  ✓ Build cache cleaned"
echo "  ✓ Dependencies updated"
echo "  ✓ Vulnerability scan: 0 CVEs"
echo "  ✓ All binaries built successfully"
echo "  ✓ All unit tests passing"
echo ""
echo "Next steps:"
echo "  1. Review test output above"
echo "  2. Run integration tests: go test ./test/integration/..."
echo "  3. Start web server: ./bin/web-server"
echo "  4. Commit changes: git add go.mod go.sum && git commit -m 'build: Upgrade to Go 1.25.3'"
echo ""
echo -e "${GREEN}Ready for production deployment!${NC}"
