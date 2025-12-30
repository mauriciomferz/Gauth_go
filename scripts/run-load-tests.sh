#!/bin/bash
# Load Testing Script for AgentAuth
# Requires: k6 (https://k6.io/docs/getting-started/installation/)

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BASE_URL="${BASE_URL:-http://localhost:8080}"
TEST_SCRIPT="tests/load/k6-load-test.js"
OUTPUT_DIR="tests/load/results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Check if k6 is installed
if ! command -v k6 &> /dev/null; then
    echo -e "${RED}❌ k6 is not installed${NC}"
    echo -e "${YELLOW}Install k6:${NC}"
    echo "  macOS:   brew install k6"
    echo "  Linux:   sudo apt-get install k6"
    echo "  Or visit: https://k6.io/docs/getting-started/installation/"
    exit 1
fi

# Check if server is running
echo -e "${BLUE}🔍 Checking if server is running...${NC}"
if ! curl -s "${BASE_URL}/healthz" > /dev/null 2>&1; then
    echo -e "${RED}❌ Server is not running at ${BASE_URL}${NC}"
    echo -e "${YELLOW}Start the server first:${NC}"
    echo "  go run ./cmd/web-server"
    exit 1
fi
echo -e "${GREEN}✅ Server is running${NC}"

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Function to run a specific test scenario
run_test() {
    local scenario=$1
    local description=$2
    
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════${NC}"
    echo -e "${BLUE}  Running: ${description}${NC}"
    echo -e "${BLUE}═══════════════════════════════════════${NC}"
    echo ""
    
    local output_file="${OUTPUT_DIR}/${scenario}_${TIMESTAMP}.json"
    local html_output="${OUTPUT_DIR}/${scenario}_${TIMESTAMP}.html"
    
    # Run k6 test
    BASE_URL="${BASE_URL}" k6 run \
        --out json="${output_file}" \
        --summary-export="${output_file}.summary.json" \
        "${TEST_SCRIPT}"
    
    echo ""
    echo -e "${GREEN}✅ Test completed${NC}"
    echo -e "${YELLOW}Results saved to: ${output_file}${NC}"
    
    # Generate HTML report if jq is available
    if command -v jq &> /dev/null; then
        generate_html_report "${output_file}.summary.json" "${html_output}"
        echo -e "${YELLOW}HTML report: ${html_output}${NC}"
    fi
}

# Function to generate HTML report
generate_html_report() {
    local json_file=$1
    local html_file=$2
    
    cat > "${html_file}" <<'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>K6 Load Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 40px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 3px solid #7d64ff; padding-bottom: 10px; }
        h2 { color: #555; margin-top: 30px; }
        .metric-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin: 20px 0; }
        .metric-card { background: #f8f9fa; padding: 20px; border-radius: 6px; border-left: 4px solid #7d64ff; }
        .metric-label { font-size: 14px; color: #666; text-transform: uppercase; }
        .metric-value { font-size: 32px; font-weight: bold; color: #333; margin: 10px 0; }
        .metric-unit { font-size: 14px; color: #999; }
        .success { border-left-color: #28a745; }
        .warning { border-left-color: #ffc107; }
        .danger { border-left-color: #dc3545; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background: #7d64ff; color: white; }
        tr:hover { background: #f5f5f5; }
        .timestamp { color: #999; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 K6 Load Test Report</h1>
        <p class="timestamp">Generated: TIMESTAMP_PLACEHOLDER</p>
        
        <h2>📊 Key Metrics</h2>
        <div class="metric-grid">
            <div class="metric-card success">
                <div class="metric-label">Total Requests</div>
                <div class="metric-value">REQUESTS_PLACEHOLDER</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Request Rate</div>
                <div class="metric-value">RATE_PLACEHOLDER</div>
                <div class="metric-unit">req/s</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Failed Requests</div>
                <div class="metric-value">FAILED_PLACEHOLDER</div>
            </div>
            <div class="metric-card warning">
                <div class="metric-label">Error Rate</div>
                <div class="metric-value">ERROR_RATE_PLACEHOLDER</div>
                <div class="metric-unit">%</div>
            </div>
        </div>
        
        <h2>⏱️ Response Times</h2>
        <table>
            <tr>
                <th>Metric</th>
                <th>Value</th>
            </tr>
            <tr>
                <td>Average</td>
                <td>AVG_PLACEHOLDER ms</td>
            </tr>
            <tr>
                <td>Minimum</td>
                <td>MIN_PLACEHOLDER ms</td>
            </tr>
            <tr>
                <td>Maximum</td>
                <td>MAX_PLACEHOLDER ms</td>
            </tr>
            <tr>
                <td>95th Percentile</td>
                <td>P95_PLACEHOLDER ms</td>
            </tr>
            <tr>
                <td>99th Percentile</td>
                <td>P99_PLACEHOLDER ms</td>
            </tr>
        </table>
        
        <h2>📈 Throughput</h2>
        <div class="metric-grid">
            <div class="metric-card">
                <div class="metric-label">Data Received</div>
                <div class="metric-value">DATA_RX_PLACEHOLDER</div>
                <div class="metric-unit">MB</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">Data Sent</div>
                <div class="metric-value">DATA_TX_PLACEHOLDER</div>
                <div class="metric-unit">MB</div>
            </div>
        </div>
    </div>
</body>
</html>
EOF
    
    # Replace placeholders with actual values from JSON
    if [ -f "${json_file}" ]; then
        local timestamp=$(date)
        sed -i.bak "s/TIMESTAMP_PLACEHOLDER/${timestamp}/g" "${html_file}"
        rm "${html_file}.bak" 2>/dev/null || true
    fi
}

# Main menu
echo -e "${BLUE}╔═══════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     AgentAuth Load Testing Suite          ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}Target Server:${NC} ${BASE_URL}"
echo ""
echo "Select test scenario:"
echo "  1) Quick Test (10 VUs, 30s)"
echo "  2) Baseline Test (10 VUs, 2m)"
echo "  3) Ramp-Up Test (0→200 VUs, 7m)"
echo "  4) Spike Test (50→500 VUs, 3m)"
echo "  5) Full Test Suite (All scenarios)"
echo "  6) Custom Test"
echo ""
read -p "Enter choice [1-6]: " choice

case $choice in
    1)
        echo -e "${GREEN}Running Quick Test...${NC}"
        BASE_URL="${BASE_URL}" k6 run --vus 10 --duration 30s "${TEST_SCRIPT}"
        ;;
    2)
        run_test "baseline" "Baseline Test"
        ;;
    3)
        run_test "rampup" "Ramp-Up Test"
        ;;
    4)
        run_test "spike" "Spike Test"
        ;;
    5)
        echo -e "${GREEN}Running Full Test Suite...${NC}"
        run_test "baseline" "Baseline Test"
        run_test "rampup" "Ramp-Up Test"
        run_test "spike" "Spike Test"
        ;;
    6)
        read -p "Enter number of VUs: " vus
        read -p "Enter duration (e.g., 30s, 5m): " duration
        echo -e "${GREEN}Running Custom Test: ${vus} VUs for ${duration}${NC}"
        BASE_URL="${BASE_URL}" k6 run --vus "${vus}" --duration "${duration}" "${TEST_SCRIPT}"
        ;;
    *)
        echo -e "${RED}Invalid choice${NC}"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}═══════════════════════════════════════${NC}"
echo -e "${GREEN}  Load Testing Complete!${NC}"
echo -e "${GREEN}═══════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}📊 View results in: ${OUTPUT_DIR}${NC}"
echo ""
echo -e "${BLUE}💡 Tip: View Prometheus metrics at ${BASE_URL}/metrics${NC}"
echo -e "${BLUE}💡 Tip: Check Grafana dashboards for detailed analysis${NC}"
echo ""
