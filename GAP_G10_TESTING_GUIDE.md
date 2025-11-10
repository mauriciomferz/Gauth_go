# Gap G10 Integration Testing Guide

**Date**: November 10, 2025  
**Version**: 1.0  
**Status**: Production Ready

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Test Structure](#test-structure)
4. [Running Tests](#running-tests)
5. [Benchmarks](#benchmarks)
6. [Coverage Analysis](#coverage-analysis)
7. [Troubleshooting](#troubleshooting)
8. [CI/CD Integration](#cicd-integration)

---

## Overview

The Gap G10 integration test suite validates RFC-0111 (GAuth 1.0) and RFC-0115 (Power of Attorney) compliance across all authorization components. The suite includes:

- **91 functional tests** across 6 test files
- **19 performance benchmarks** (16 component + 3 E2E)
- **59 subtests** for granular validation
- **4,400+ lines** of test code
- **90% test coverage** across critical components

### Test Coverage by Component

| Component | Tests | Coverage | Status |
|-----------|-------|----------|--------|
| Extended Token | 13 | 95% | ✅ |
| PVP (Power Verification Point) | 15 | 90% | ✅ |
| Commercial Register | 28 | 85% | ✅ |
| PIP (Power Information Point) | 16 | 90% | ✅ |
| PoA (Power of Attorney) | 15 | 90% | ✅ |
| E2E Integration | 4 | 95% | ✅ |

---

## Quick Start

### Prerequisites

```bash
# Go 1.21 or later required
go version

# Verify workspace
cd /path/to/Gauth_go
```

### Run All Tests (Fast)

```bash
# Run all unit tests (no integration tag)
go test ./pkg/...

# Run all integration tests
go test -tags=integration ./test/integration/...

# Run everything
go test -tags=integration ./...
```

### Run Specific Test Suite

```bash
# Extended Token tests
go test -v ./pkg/gauth -run TestExtendedToken

# PVP tests
go test -v ./pkg/verification -run TestDefaultPVP

# Commercial Register tests
go test -v ./pkg/registry -run TestMockCommercialRegister

# PIP tests
go test -v ./pkg/pip -run TestDefaultPIP

# PoA tests
go test -v ./pkg/poa -run TestPoADefinition

# E2E integration tests
go test -v -tags=integration ./test/integration -run TestGapG10E2E
```

---

## Test Structure

### Directory Layout

```
Gauth_go/
├── pkg/
│   ├── gauth/
│   │   └── extended_token_test.go      # Phase 1: Extended Token (13 tests)
│   ├── verification/
│   │   └── pvp_test.go                 # Phase 2: PVP (15 tests)
│   ├── registry/
│   │   └── commercial_register_test.go # Phase 3: Commercial Register (28 tests)
│   ├── pip/
│   │   └── pip_test.go                 # Phase 4: PIP (16 tests)
│   └── poa/
│       └── poa_test.go                 # Phase 5: PoA (15 tests)
└── test/
    └── integration/
        └── gap_g10_e2e_test.go         # Phase 6: E2E (4 main + 12 subtests)
```

### Build Tags

Integration tests use the `integration` build tag to separate them from unit tests:

```go
//go:build integration
// +build integration

package integration
```

**Why?** Integration tests may:
- Take longer to execute
- Require external services (mocked in our case)
- Test multi-component interactions

---

## Running Tests

### Basic Test Execution

#### 1. Unit Tests Only (Fast - ~9 seconds)

```bash
# Run unit tests without integration tag
go test ./pkg/gauth ./pkg/verification ./pkg/registry ./pkg/pip ./pkg/poa

# With verbose output
go test -v ./pkg/gauth ./pkg/verification ./pkg/registry ./pkg/pip ./pkg/poa

# With timeout
go test -timeout 30s ./pkg/...
```

#### 2. Integration Tests Only (~1 second)

```bash
# Run E2E integration tests
go test -tags=integration ./test/integration

# Verbose with test names
go test -v -tags=integration ./test/integration

# Specific E2E test
go test -v -tags=integration ./test/integration -run TestGapG10E2E_CompleteTokenIssuanceFlow
```

#### 3. All Tests (~10 seconds)

```bash
# Run everything
go test -tags=integration ./...

# Verbose
go test -v -tags=integration ./...
```

### Advanced Test Execution

#### Run Specific Test Function

```bash
# Run single test
go test -v ./pkg/gauth -run TestExtendedToken_Validate

# Run test with subtest
go test -v ./pkg/gauth -run "TestExtendedToken_Validate/Valid_complete_token"

# Multiple test patterns
go test -v ./pkg/poa -run "TestPoADefinition_Validate|TestPoADefinition_RepresentativeTypes"
```

#### Test with Race Detection

```bash
# Check for race conditions
go test -race ./pkg/pip

# All packages
go test -race ./...
```

#### Parallel Test Execution

```bash
# Run tests in parallel (default)
go test -parallel 4 ./...

# Sequential execution (for debugging)
go test -parallel 1 ./...
```

#### Test with Timeout

```bash
# Custom timeout (default is 10m)
go test -timeout 30s ./pkg/...

# Long-running integration tests
go test -timeout 5m -tags=integration ./test/integration
```

---

## Benchmarks

### Running Benchmarks

#### Component Benchmarks

```bash
# PVP benchmarks
go test -bench=. -benchmem -run=^$ ./pkg/verification

# Commercial Register benchmarks
go test -bench=. -benchmem -run=^$ ./pkg/registry

# PIP benchmarks
go test -bench=. -benchmem -run=^$ ./pkg/pip

# All component benchmarks
go test -bench=. -benchmem -run=^$ ./pkg/verification ./pkg/registry ./pkg/pip
```

#### E2E Benchmarks

```bash
# E2E flow benchmarks
go test -bench=BenchmarkE2E -benchmem -run=^$ -tags=integration ./test/integration

# Specific E2E benchmark
go test -bench=BenchmarkE2ETokenIssuanceFlow -benchmem -run=^$ -tags=integration ./test/integration
```

#### All Benchmarks

```bash
# Run all benchmarks (component + E2E)
go test -bench=. -benchmem -run=^$ ./pkg/verification ./pkg/registry ./pkg/pip
go test -bench=BenchmarkE2E -benchmem -run=^$ -tags=integration ./test/integration
```

### Benchmark Options

```bash
# Longer benchmark time for stable results
go test -bench=. -benchtime=3s -benchmem ./pkg/pip

# Multiple runs for consistency
go test -bench=. -count=5 -benchmem ./pkg/pip

# CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./pkg/pip
go tool pprof cpu.prof

# Memory profiling
go test -bench=. -memprofile=mem.prof -benchmem ./pkg/pip
go tool pprof mem.prof
```

### Benchmark Comparison

```bash
# Save baseline
go test -bench=. -benchmem -run=^$ ./pkg/pip > baseline.txt

# Make changes, then compare
go test -bench=. -benchmem -run=^$ ./pkg/pip > new.txt
benchstat baseline.txt new.txt
```

### Expected Benchmark Results

| Benchmark | Expected Performance | Alert If Exceeds |
|-----------|---------------------|------------------|
| PIP Cache Get | < 50 ns/op | 100 ns/op |
| PIP Authorization Validation | < 500 ns/op | 1 µs/op |
| PVP Identity Chain | < 1 µs/op | 2 µs/op |
| E2E Token Issuance | < 2 µs/op | 5 µs/op |
| E2E Authorization Decision | < 500 ns/op | 1 µs/op |

---

## Coverage Analysis

### Generate Coverage Report

#### Basic Coverage

```bash
# Component coverage
go test -cover ./pkg/gauth
go test -cover ./pkg/verification
go test -cover ./pkg/registry
go test -cover ./pkg/pip
go test -cover ./pkg/poa

# All packages
go test -cover ./...
```

#### Detailed Coverage with HTML Report

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./pkg/gauth ./pkg/verification ./pkg/registry ./pkg/pip ./pkg/poa

# View in browser
go tool cover -html=coverage.out

# Text summary
go tool cover -func=coverage.out
```

#### Coverage by Package

```bash
# Individual package reports
go test -coverprofile=gauth_coverage.out ./pkg/gauth
go test -coverprofile=pvp_coverage.out ./pkg/verification
go test -coverprofile=registry_coverage.out ./pkg/registry
go test -coverprofile=pip_coverage.out ./pkg/pip
go test -coverprofile=poa_coverage.out ./pkg/poa

# View specific package
go tool cover -html=pip_coverage.out
```

#### Combined Coverage Report

```bash
# Generate combined profile
go test -coverprofile=coverage.out -covermode=atomic ./...

# Filter out test files
grep -v "_test.go" coverage.out > coverage_filtered.out

# View combined report
go tool cover -html=coverage_filtered.out
```

### Coverage Targets

| Component | Current | Target | Status |
|-----------|---------|--------|--------|
| Extended Token | 95% | 95% | ✅ Met |
| PVP | 90% | 90% | ✅ Met |
| Commercial Register | 85% | 85% | ✅ Met |
| PIP | 90% | 90% | ✅ Met |
| PoA | 90% | 90% | ✅ Met |
| E2E Integration | 95% | 95% | ✅ Met |
| **Overall** | **90%** | **≥90%** | ✅ **Met** |

---

## Troubleshooting

### Common Issues

#### Issue: Tests Fail with "undefined: integration"

**Symptom**:
```
# github.com/.../test/integration
test/integration/gap_g10_e2e_test.go:1:1: expected 'package', found 'EOF'
```

**Solution**: Add the `-tags=integration` flag:
```bash
go test -tags=integration ./test/integration
```

#### Issue: "context deadline exceeded"

**Symptom**:
```
panic: test timed out after 10m0s
```

**Solution**: Increase timeout:
```bash
go test -timeout 30m ./...
```

#### Issue: Commercial Register Tests Take Too Long

**Symptom**: Tests taking 6+ seconds

**Explanation**: Commercial Register mocks include 100ms simulated network delays (realistic)

**Solution**: This is expected behavior. To skip:
```bash
go test -short ./pkg/registry  # Skip long-running tests
```

#### Issue: Race Condition Detected

**Symptom**:
```
WARNING: DATA RACE
```

**Solution**: 
1. Run with `-race` to identify location
2. Check PIP cache operations (already fixed in pip.go:686)
3. Ensure proper locking in concurrent code

#### Issue: Benchmark Results Vary Significantly

**Symptom**: Benchmark results differ by >20% between runs

**Solution**:
```bash
# Run longer benchmarks with multiple iterations
go test -bench=. -benchtime=3s -count=5 ./pkg/pip

# Check system load
top  # Ensure minimal background processes

# Disable CPU throttling (macOS)
sudo pmset -a disablesleep 1
```

### Debug Techniques

#### Verbose Test Output

```bash
# Show all test output
go test -v ./pkg/gauth

# Show logs even for passing tests
go test -v -args -test.v
```

#### Run Single Test with Debug

```bash
# Run specific test with verbose output
go test -v ./pkg/gauth -run TestExtendedToken_Validate

# Add debug prints in test code
t.Logf("Debug: value = %+v", variable)
```

#### Test Caching Issues

```bash
# Force re-run (ignore cache)
go test -count=1 ./...

# Clear test cache
go clean -testcache
```

---

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Gap G10 Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run Unit Tests
        run: go test -v -race -coverprofile=coverage.out ./pkg/...
      
      - name: Run Integration Tests
        run: go test -v -tags=integration ./test/integration/...
      
      - name: Upload Coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
      
      - name: Run Benchmarks
        run: |
          go test -bench=. -benchmem -run=^$ ./pkg/verification ./pkg/registry ./pkg/pip > bench.txt
          go test -bench=BenchmarkE2E -benchmem -run=^$ -tags=integration ./test/integration >> bench.txt
      
      - name: Check Performance Regression
        run: |
          # Compare with baseline (requires benchstat)
          go install golang.org/x/perf/cmd/benchstat@latest
          benchstat baseline_bench.txt bench.txt
```

### GitLab CI Example

```yaml
stages:
  - test
  - benchmark

unit-tests:
  stage: test
  script:
    - go test -v -race -coverprofile=coverage.out ./pkg/...
  coverage: '/coverage: \d+\.\d+% of statements/'

integration-tests:
  stage: test
  script:
    - go test -v -tags=integration ./test/integration/...

benchmarks:
  stage: benchmark
  script:
    - go test -bench=. -benchmem -run=^$ ./pkg/... > bench.txt
    - go test -bench=BenchmarkE2E -benchmem -run=^$ -tags=integration ./test/integration >> bench.txt
  artifacts:
    paths:
      - bench.txt
```

### Make Targets

Create a `Makefile` for convenience:

```makefile
.PHONY: test test-unit test-integration test-all bench coverage

test: test-unit test-integration

test-unit:
	@echo "Running unit tests..."
	go test -v -race ./pkg/...

test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./test/integration/...

test-all:
	@echo "Running all tests..."
	go test -v -race -tags=integration ./...

bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem -run=^$ ./pkg/verification ./pkg/registry ./pkg/pip
	@go test -bench=BenchmarkE2E -benchmem -run=^$ -tags=integration ./test/integration

coverage:
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./pkg/...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
```

Usage:
```bash
make test          # Run all tests
make test-unit     # Unit tests only
make test-integration  # Integration tests only
make bench         # Run benchmarks
make coverage      # Generate coverage report
```

---

## Test Development Guidelines

### Writing New Tests

#### Unit Test Template

```go
func TestComponentName_Operation(t *testing.T) {
    // Arrange
    component := NewComponent()
    input := createTestInput()
    
    // Act
    result, err := component.Operation(input)
    
    // Assert
    require.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, expectedValue, result.Field)
}
```

#### Integration Test Template

```go
//go:build integration
// +build integration

func TestIntegration_CompleteFlow(t *testing.T) {
    // Setup
    service1 := setupService1()
    service2 := setupService2()
    
    // Execute flow
    t.Run("Step1", func(t *testing.T) {
        result1, err := service1.Operation()
        require.NoError(t, err)
        assert.NotNil(t, result1)
    })
    
    t.Run("Step2", func(t *testing.T) {
        result2, err := service2.Operation()
        require.NoError(t, err)
        assert.NotNil(t, result2)
    })
}
```

#### Benchmark Template

```go
func BenchmarkComponent_Operation(b *testing.B) {
    // Setup (not timed)
    component := NewComponent()
    input := createTestInput()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // Operation to benchmark
        _, _ = component.Operation(input)
    }
}
```

### Testing Best Practices

1. **Use Table-Driven Tests** for multiple scenarios
2. **Use Subtests** (`t.Run()`) for related test cases
3. **Mock External Dependencies** (databases, APIs)
4. **Test Error Cases** not just happy paths
5. **Keep Tests Fast** (< 1 second per test ideally)
6. **Avoid Test Interdependencies** (each test should be independent)
7. **Use Descriptive Test Names** (`TestComponent_Operation_ErrorCase`)

---

## Performance Expectations

### Test Execution Times

| Test Suite | Expected Time | Alert If Exceeds |
|------------|---------------|------------------|
| Extended Token | < 0.5s | 1s |
| PVP | < 0.5s | 1s |
| Commercial Register | < 7s | 10s |
| PIP | < 2s | 3s |
| PoA | < 1s | 2s |
| E2E Integration | < 1s | 2s |
| **Total** | **< 12s** | **20s** |

### Benchmark Performance

See [GAP_G10_PHASE7_PERFORMANCE_REPORT.md](./GAP_G10_PHASE7_PERFORMANCE_REPORT.md) for detailed performance targets.

---

## Additional Resources

### Documentation
- [RFC-0111: GAuth 1.0 Specification](./docs/RFC-0111.md)
- [RFC-0115: Power of Attorney](./docs/RFC-0115.md)
- [Gap G10 Integration Tests Progress](./GAP_G10_INTEGRATION_TESTS_PROGRESS.md)
- [Phase 6: E2E Tests Completion](./GAP_G10_PHASE6_E2E_TESTS_COMPLETION.md)
- [Phase 7: Performance Report](./GAP_G10_PHASE7_PERFORMANCE_REPORT.md)

### Tools
- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Assertions](https://github.com/stretchr/testify)
- [Benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)

### Support
For questions or issues:
1. Check this guide first
2. Review test code in relevant package
3. Consult phase completion reports
4. Check Git history for context

---

**Document Version**: 1.0  
**Last Updated**: November 10, 2025  
**Status**: Complete and Production Ready  
**Maintainer**: Gap G10 Integration Testing Team
