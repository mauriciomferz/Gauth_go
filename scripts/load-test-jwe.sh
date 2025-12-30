#!/bin/bash
# AgentAuth JWE Encryption - Load Testing Script
# Tests performance and scalability under realistic load

set -e

# Configuration
TARGET_URL="${TARGET_URL:-http://localhost:8080}"
DURATION="${DURATION:-60}"  # seconds
CONCURRENT_USERS="${CONCURRENT_USERS:-100}"
RPS_TARGET="${RPS_TARGET:-1000}"  # requests per second

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "========================================"
echo "AgentAuth JWE Load Testing"
echo "========================================"
echo "Target: $TARGET_URL"
echo "Duration: ${DURATION}s"
echo "Concurrent Users: $CONCURRENT_USERS"
echo "Target RPS: $RPS_TARGET"
echo "========================================"

# Check dependencies
command -v vegeta >/dev/null 2>&1 || {
    echo -e "${RED}Error: vegeta is not installed${NC}"
    echo "Install: go install github.com/tsenart/vegeta@latest"
    exit 1
}

command -v jq >/dev/null 2>&1 || {
    echo -e "${YELLOW}Warning: jq is not installed (results will be less detailed)${NC}"
}

# Create output directory
mkdir -p load-test-results
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULT_DIR="load-test-results/$TIMESTAMP"
mkdir -p "$RESULT_DIR"

# Test 1: Token Issuance (with JWE encryption)
echo ""
echo "Test 1: Token Issuance (JWE Encryption)"
echo "========================================"

cat > "$RESULT_DIR/token-targets.txt" <<EOF
POST ${TARGET_URL}/api/v1/beta/authorize
Content-Type: application/json
@$RESULT_DIR/token-request.json

EOF

# Create sample token request
cat > "$RESULT_DIR/token-request.json" <<'EOF'
{
  "client_id": "test-client-001",
  "grant_type": "authorization_code",
  "code": "test-auth-code-123",
  "scope": "read:resource write:resource"
}
EOF

echo "Running token issuance test..."
vegeta attack \
    -targets="$RESULT_DIR/token-targets.txt" \
    -rate=$RPS_TARGET \
    -duration=${DURATION}s \
    -timeout=10s \
    > "$RESULT_DIR/token-results.bin"

echo "Generating report..."
vegeta report "$RESULT_DIR/token-results.bin" > "$RESULT_DIR/token-report.txt"
vegeta report -type=json "$RESULT_DIR/token-results.bin" > "$RESULT_DIR/token-report.json"

# Display summary
echo ""
cat "$RESULT_DIR/token-report.txt"

# Extract key metrics
if command -v jq >/dev/null 2>&1; then
    echo ""
    echo "Key Metrics:"
    LATENCY_P99=$(jq -r '.latencies.p99 / 1000000' "$RESULT_DIR/token-report.json")
    LATENCY_MEAN=$(jq -r '.latencies.mean / 1000000' "$RESULT_DIR/token-report.json")
    SUCCESS_RATE=$(jq -r '.success * 100' "$RESULT_DIR/token-report.json")
    
    echo "  Mean Latency: ${LATENCY_MEAN}ms"
    echo "  P99 Latency: ${LATENCY_P99}ms"
    echo "  Success Rate: ${SUCCESS_RATE}%"
    
    # Check if encryption overhead is acceptable
    if (( $(echo "$LATENCY_MEAN > 10" | bc -l) )); then
        echo -e "${YELLOW}  ⚠️  Mean latency > 10ms (encryption overhead high)${NC}"
    else
        echo -e "${GREEN}  ✓ Mean latency < 10ms (encryption overhead acceptable)${NC}"
    fi
fi

# Test 2: Token Validation (with JWE decryption)
echo ""
echo ""
echo "Test 2: Token Validation (JWE Decryption)"
echo "========================================"

# First, get a valid token
echo "Obtaining valid token..."
VALID_TOKEN=$(curl -s -X POST "${TARGET_URL}/api/v1/beta/authorize" \
    -H "Content-Type: application/json" \
    -d '{"client_id":"test-client-001","grant_type":"authorization_code","code":"test-code"}' \
    | jq -r '.access_token' 2>/dev/null || echo "")

if [ -z "$VALID_TOKEN" ]; then
    echo -e "${YELLOW}Warning: Could not obtain valid token, skipping validation test${NC}"
else
    cat > "$RESULT_DIR/validation-targets.txt" <<EOF
GET ${TARGET_URL}/api/v1/beta/resource
Authorization: Bearer ${VALID_TOKEN}

EOF

    echo "Running token validation test..."
    vegeta attack \
        -targets="$RESULT_DIR/validation-targets.txt" \
        -rate=$RPS_TARGET \
        -duration=${DURATION}s \
        -timeout=10s \
        > "$RESULT_DIR/validation-results.bin"

    echo "Generating report..."
    vegeta report "$RESULT_DIR/validation-results.bin" > "$RESULT_DIR/validation-report.txt"
    vegeta report -type=json "$RESULT_DIR/validation-results.bin" > "$RESULT_DIR/validation-report.json"

    # Display summary
    echo ""
    cat "$RESULT_DIR/validation-report.txt"

    # Extract key metrics
    if command -v jq >/dev/null 2>&1; then
        echo ""
        echo "Key Metrics:"
        LATENCY_P99=$(jq -r '.latencies.p99 / 1000000' "$RESULT_DIR/validation-report.json")
        LATENCY_MEAN=$(jq -r '.latencies.mean / 1000000' "$RESULT_DIR/validation-report.json")
        SUCCESS_RATE=$(jq -r '.success * 100' "$RESULT_DIR/validation-report.json")
        
        echo "  Mean Latency: ${LATENCY_MEAN}ms"
        echo "  P99 Latency: ${LATENCY_P99}ms"
        echo "  Success Rate: ${SUCCESS_RATE}%"
    fi
fi

# Test 3: Mixed Workload (50% issuance, 50% validation)
echo ""
echo ""
echo "Test 3: Mixed Workload"
echo "========================================"

cat > "$RESULT_DIR/mixed-targets.txt" <<EOF
POST ${TARGET_URL}/api/v1/beta/authorize
Content-Type: application/json
@$RESULT_DIR/token-request.json

GET ${TARGET_URL}/api/v1/beta/resource
Authorization: Bearer ${VALID_TOKEN}

EOF

echo "Running mixed workload test..."
vegeta attack \
    -targets="$RESULT_DIR/mixed-targets.txt" \
    -rate=$RPS_TARGET \
    -duration=${DURATION}s \
    -timeout=10s \
    > "$RESULT_DIR/mixed-results.bin"

echo "Generating report..."
vegeta report "$RESULT_DIR/mixed-results.bin" > "$RESULT_DIR/mixed-report.txt"
vegeta report -type=json "$RESULT_DIR/mixed-results.bin" > "$RESULT_DIR/mixed-report.json"

# Display summary
echo ""
cat "$RESULT_DIR/mixed-report.txt"

# Test 4: Stress Test (ramp up to failure)
echo ""
echo ""
echo "Test 4: Stress Test (Finding Breaking Point)"
echo "========================================"

STRESS_RATES=(100 500 1000 2000 5000 10000)
echo "Testing at rates: ${STRESS_RATES[*]} req/s"

for RATE in "${STRESS_RATES[@]}"; do
    echo ""
    echo "Testing at ${RATE} req/s..."
    
    vegeta attack \
        -targets="$RESULT_DIR/token-targets.txt" \
        -rate=$RATE \
        -duration=30s \
        -timeout=10s \
        > "$RESULT_DIR/stress-${RATE}-results.bin"
    
    vegeta report -type=json "$RESULT_DIR/stress-${RATE}-results.bin" > "$RESULT_DIR/stress-${RATE}-report.json"
    
    if command -v jq >/dev/null 2>&1; then
        SUCCESS_RATE=$(jq -r '.success * 100' "$RESULT_DIR/stress-${RATE}-report.json")
        LATENCY_P99=$(jq -r '.latencies.p99 / 1000000' "$RESULT_DIR/stress-${RATE}-report.json")
        
        echo "  Rate: ${RATE} req/s, Success: ${SUCCESS_RATE}%, P99: ${LATENCY_P99}ms"
        
        # If success rate drops below 95%, we've found the limit
        if (( $(echo "$SUCCESS_RATE < 95" | bc -l) )); then
            echo -e "${YELLOW}  ⚠️  Breaking point reached at ${RATE} req/s${NC}"
            break
        fi
    fi
done

# Generate HTML report
echo ""
echo ""
echo "Generating visualizations..."

if command -v vegeta >/dev/null 2>&1; then
    vegeta plot "$RESULT_DIR/token-results.bin" > "$RESULT_DIR/token-latency-plot.html"
    vegeta plot "$RESULT_DIR/validation-results.bin" > "$RESULT_DIR/validation-latency-plot.html"
    vegeta plot "$RESULT_DIR/mixed-results.bin" > "$RESULT_DIR/mixed-latency-plot.html"
    echo -e "${GREEN}✓ Latency plots generated${NC}"
fi

# Generate summary report
cat > "$RESULT_DIR/SUMMARY.md" <<EOF
# AgentAuth JWE Load Test Summary

**Date**: $(date)
**Target**: $TARGET_URL
**Duration**: ${DURATION}s per test
**Target RPS**: $RPS_TARGET

---

## Test 1: Token Issuance (JWE Encryption)

$(cat "$RESULT_DIR/token-report.txt")

## Test 2: Token Validation (JWE Decryption)

$(cat "$RESULT_DIR/validation-report.txt" 2>/dev/null || echo "Test skipped (no valid token)")

## Test 3: Mixed Workload

$(cat "$RESULT_DIR/mixed-report.txt")

---

## Performance Analysis

### Encryption Overhead

- **Target**: < 10ms total latency (encryption + serialization)
- **Actual**: See reports above

### Throughput

- **Target**: 1000 req/s per instance
- **Actual**: See stress test results

### Scalability

- **Breaking Point**: See stress test section

---

## Recommendations

1. If P99 latency > 10ms:
   - Increase CPU allocation
   - Enable hardware AES-NI acceleration
   - Scale horizontally (more instances)

2. If success rate < 99.9%:
   - Check error logs
   - Verify database connection pool size
   - Check network latency

3. If memory usage high:
   - Implement key caching
   - Optimize token size
   - Enable compression

---

## Next Steps

- [ ] Review detailed reports in this directory
- [ ] Compare with baseline (non-JWE) performance
- [ ] Adjust resource allocation if needed
- [ ] Run tests in production-like environment
- [ ] Set up continuous performance monitoring

EOF

echo ""
echo "========================================"
echo -e "${GREEN}Load testing complete!${NC}"
echo "========================================"
echo "Results saved to: $RESULT_DIR"
echo ""
echo "Key files:"
echo "  - SUMMARY.md (human-readable summary)"
echo "  - *-report.txt (detailed text reports)"
echo "  - *-report.json (machine-readable results)"
echo "  - *-latency-plot.html (visual latency charts)"
echo ""
echo "Open summary: cat $RESULT_DIR/SUMMARY.md"
echo "Open plots: open $RESULT_DIR/*-plot.html"
echo ""

# Check for critical issues
if command -v jq >/dev/null 2>&1; then
    TOKEN_SUCCESS=$(jq -r '.success * 100' "$RESULT_DIR/token-report.json")
    TOKEN_P99=$(jq -r '.latencies.p99 / 1000000' "$RESULT_DIR/token-report.json")
    
    echo "Health Check:"
    if (( $(echo "$TOKEN_SUCCESS < 99" | bc -l) )); then
        echo -e "${RED}  ✗ Token issuance success rate < 99% (${TOKEN_SUCCESS}%)${NC}"
    else
        echo -e "${GREEN}  ✓ Token issuance success rate OK (${TOKEN_SUCCESS}%)${NC}"
    fi
    
    if (( $(echo "$TOKEN_P99 > 50" | bc -l) )); then
        echo -e "${YELLOW}  ⚠️  Token issuance P99 latency high (${TOKEN_P99}ms)${NC}"
    else
        echo -e "${GREEN}  ✓ Token issuance P99 latency OK (${TOKEN_P99}ms)${NC}"
    fi
fi
