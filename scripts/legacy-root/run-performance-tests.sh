#!/usr/bin/env bash
###############################################
# K6 Performance Testing Runner Script
# Runs performance tests and generates reports
###############################################

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
API_URL="${API_URL:-http://localhost:8080}"
RESULTS_DIR="$ROOT_DIR/performance-tests/results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

mkdir -p "$RESULTS_DIR"

echo -e "${BLUE}╔═══════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     AgentAuth Performance Testing Suite       ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════╝${NC}"
echo ""

if ! command -v k6 &> /dev/null; then
  echo -e "${RED}❌ k6 is not installed${NC}"
  echo -e "${YELLOW}Install k6: https://k6.io/docs/getting-started/installation${NC}"
  echo ""
  echo "Quick install options:"
  echo "  macOS:    brew install k6"
  echo "  Linux:    https://k6.io/docs/getting-started/installation/"
  echo "  Windows:  choco install k6"
  exit 1
fi

echo -e "${YELLOW}🔍 Checking backend availability...${NC}"
if curl -s -f "${API_URL}/health" > /dev/null; then
  echo -e "${GREEN}✅ Backend is running at ${API_URL}${NC}"
else
  echo -e "${RED}❌ Backend not responding at ${API_URL}${NC}"
  exit 1
fi

echo ""

run_test() {
  local test_name=$1
  local test_file=$2
  local description=$3

  echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "${BLUE}🚀 Running: ${test_name}${NC}"
  echo -e "${BLUE}   ${description}${NC}"
  echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

  if k6 run --out json="${RESULTS_DIR}/${test_name}-${TIMESTAMP}.json" \
            --summary-export="${RESULTS_DIR}/${test_name}-${TIMESTAMP}-summary.json" \
            -e API_URL="${API_URL}" \
            "${test_file}"; then
    echo -e "${GREEN}✅ ${test_name} completed successfully${NC}"
    return 0
  else
    echo -e "${RED}❌ ${test_name} failed${NC}"
    return 1
  fi
}

TEST_TYPE="${1:-all}"

case "$TEST_TYPE" in
  health)
    run_test "health-check" "$ROOT_DIR/performance-tests/health-check.js" "Health endpoint performance test"
    ;;
  token)
    run_test "token-creation" "$ROOT_DIR/performance-tests/token-creation.js" "Token creation performance test"
    ;;
  metrics)
    run_test "metrics-endpoint" "$ROOT_DIR/performance-tests/metrics-endpoint.js" "Metrics endpoint performance test"
    ;;
  load)
    run_test "load-test" "$ROOT_DIR/performance-tests/load-test.js" "Full API workflow load test"
    ;;
  stress)
    run_test "stress-test" "$ROOT_DIR/performance-tests/stress-test.js" "Stress test to find breaking point"
    ;;
  all)
    echo -e "${GREEN}Running all performance tests...${NC}"
    echo ""

    run_test "health-check" "$ROOT_DIR/performance-tests/health-check.js" "Health endpoint performance test"
    run_test "token-creation" "$ROOT_DIR/performance-tests/token-creation.js" "Token creation performance test"
    run_test "metrics-endpoint" "$ROOT_DIR/performance-tests/metrics-endpoint.js" "Metrics endpoint performance test"
    run_test "load-test" "$ROOT_DIR/performance-tests/load-test.js" "Full API workflow load test"

    echo ""
    echo -e "${YELLOW}⚠️  Skipping stress test in 'all' mode${NC}"
    echo -e "${YELLOW}   Run stress test separately with: scripts/legacy-root/run-performance-tests.sh stress${NC}"
    ;;
  *)
    echo -e "${RED}Unknown test type: $TEST_TYPE${NC}"
    echo ""
    echo "Usage: $0 [health|token|metrics|load|stress|all]"
    exit 1
    ;;
esac

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Performance testing complete!${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${YELLOW}📊 Results saved to: ${RESULTS_DIR}/${NC}"
echo ""
