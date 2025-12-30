#!/bin/bash
# Performance Testing Script for AgentAuth QA Enhancements
# Task 8: Performance Testing & Optimization

set -e

echo "=================================="
echo "AgentAuth Performance Testing Suite"
echo "Task 8: QA Enhancement Initiative"
echo "=================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BENCH_TIME="${BENCH_TIME:-3s}"
OUTPUT_DIR="${OUTPUT_DIR:-./performance-results}"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Create output directory
mkdir -p "$OUTPUT_DIR"

echo "Configuration:"
echo "  Benchmark time: $BENCH_TIME"
echo "  Output directory: $OUTPUT_DIR"
echo "  Timestamp: $TIMESTAMP"
echo ""

# Function to run benchmarks
run_bench() {
    local name=$1
    local pattern=$2
    local output="$OUTPUT_DIR/${name}_${TIMESTAMP}.txt"
    
    echo -e "${YELLOW}Running: $name${NC}"
    go test -bench="$pattern" -benchmem -benchtime="$BENCH_TIME" ./pkg/gauth 2>&1 | tee "$output"
    
    # Check if benchmarks passed
    if grep -q "PASS" "$output"; then
        echo -e "${GREEN}✅ $name completed successfully${NC}"
    else
        echo -e "${RED}❌ $name failed${NC}"
        return 1
    fi
    echo ""
}

# 1. Jurisdiction Performance Tests
echo "=== 1. Formal Requirements Service Benchmarks ==="
run_bench "jurisdiction_lookup" "BenchmarkJurisdiction"

# 2. Monitoring Service Tests
echo "=== 2. Monitoring Service Benchmarks ==="
run_bench "monitoring_metrics" "BenchmarkMetrics"

# 3. Concurrent Performance Tests
echo "=== 3. Concurrent Performance Benchmarks ==="
run_bench "concurrent" "BenchmarkConcurrent"

# 4. Service Initialization Tests
echo "=== 4. Service Initialization Benchmarks ==="
run_bench "service_init" "BenchmarkService"

# 5. Cache Performance Tests
echo "=== 5. Cache Performance Benchmarks ==="
run_bench "cache" "BenchmarkCache"

# 6. Prometheus Metrics Tests
echo "=== 6. Prometheus Metrics Benchmarks ==="
run_bench "prometheus" "BenchmarkHistogram|BenchmarkCounter|BenchmarkGauge"

# 7. All Benchmarks (comprehensive)
echo "=== 7. Comprehensive Benchmark Suite ==="
COMPREHENSIVE_OUTPUT="$OUTPUT_DIR/comprehensive_${TIMESTAMP}.txt"
echo -e "${YELLOW}Running comprehensive benchmarks...${NC}"
go test -bench=. -benchmem -benchtime="$BENCH_TIME" ./pkg/gauth 2>&1 | tee "$COMPREHENSIVE_OUTPUT"

if grep -q "PASS" "$COMPREHENSIVE_OUTPUT"; then
    echo -e "${GREEN}✅ Comprehensive benchmarks completed${NC}"
else
    echo -e "${RED}❌ Comprehensive benchmarks failed${NC}"
    exit 1
fi

# 8. Generate performance summary
echo ""
echo "=== 8. Performance Summary ==="
SUMMARY_FILE="$OUTPUT_DIR/summary_${TIMESTAMP}.txt"

{
    echo "AgentAuth Performance Test Summary"
    echo "=============================="
    echo "Date: $(date)"
    echo "Go Version: $(go version)"
    echo "Architecture: $(go env GOOS)/$(go env GOARCH)"
    echo ""
    echo "Benchmark Results:"
    echo "------------------"
    grep "^Benchmark" "$COMPREHENSIVE_OUTPUT" || true
    echo ""
    echo "Performance Metrics:"
    echo "-------------------"
    echo "Total benchmarks run: $(grep -c "^Benchmark" "$COMPREHENSIVE_OUTPUT" || echo "0")"
    echo "Passed: $(grep -c "PASS" "$COMPREHENSIVE_OUTPUT" || echo "0")"
    echo ""
    echo "Top 10 Fastest Operations:"
    grep "^Benchmark" "$COMPREHENSIVE_OUTPUT" | sort -k3 -n | head -10 || true
    echo ""
    echo "Top 10 Most Memory Efficient:"
    grep "^Benchmark" "$COMPREHENSIVE_OUTPUT" | sort -k5 -n | head -10 || true
} > "$SUMMARY_FILE"

cat "$SUMMARY_FILE"

# 9. CPU Profiling (optional, if requested)
if [ "$CPU_PROFILE" = "1" ]; then
    echo ""
    echo "=== 9. CPU Profiling ==="
    echo -e "${YELLOW}Generating CPU profile...${NC}"
    go test -bench=BenchmarkComprehensive -cpuprofile="$OUTPUT_DIR/cpu_${TIMESTAMP}.prof" -benchtime="$BENCH_TIME" ./pkg/gauth
    echo -e "${GREEN}✅ CPU profile saved: $OUTPUT_DIR/cpu_${TIMESTAMP}.prof${NC}"
    echo "View with: go tool pprof $OUTPUT_DIR/cpu_${TIMESTAMP}.prof"
fi

# 10. Memory Profiling (optional, if requested)
if [ "$MEM_PROFILE" = "1" ]; then
    echo ""
    echo "=== 10. Memory Profiling ==="
    echo -e "${YELLOW}Generating memory profile...${NC}"
    go test -bench=BenchmarkConcurrent -memprofile="$OUTPUT_DIR/mem_${TIMESTAMP}.prof" -benchmem -benchtime="$BENCH_TIME" ./pkg/gauth
    echo -e "${GREEN}✅ Memory profile saved: $OUTPUT_DIR/mem_${TIMESTAMP}.prof${NC}"
    echo "View with: go tool pprof $OUTPUT_DIR/mem_${TIMESTAMP}.prof"
fi

echo ""
echo "=================================="
echo -e "${GREEN}✅ Performance testing complete!${NC}"
echo "=================================="
echo ""
echo "Results saved to: $OUTPUT_DIR"
echo "Summary: $SUMMARY_FILE"
echo ""
echo "Quick stats:"
grep "^Benchmark" "$COMPREHENSIVE_OUTPUT" | wc -l | xargs echo "  Total benchmarks:"
echo ""
echo "Next steps:"
echo "  1. Review summary: cat $SUMMARY_FILE"
echo "  2. Compare with baseline: benchstat baseline.txt $COMPREHENSIVE_OUTPUT"
echo "  3. Generate CPU profile: CPU_PROFILE=1 $0"
echo "  4. Generate memory profile: MEM_PROFILE=1 $0"
echo ""
