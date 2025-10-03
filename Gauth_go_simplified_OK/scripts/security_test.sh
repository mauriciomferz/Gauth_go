#!/bin/bash

# GAuth Security Testing Suite
# This script runs basic security tests against the GAuth educational implementation

set -e

echo "🔒 Starting GAuth Security Testing Suite"
echo "========================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
PASSED=0
FAILED=0
WARNINGS=0

log_test() {
    echo -e "${GREEN}[TEST]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    ((WARNINGS++))
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((FAILED++))
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((PASSED++))
}

# 1. Run Go fuzz tests
echo -e "\n🎯 Running Fuzz Tests"
echo "====================="

log_test "Token issuance fuzz testing..."
if go test -fuzz=FuzzTokenIssuance -fuzztime=30s ./test/fuzz/ 2>/dev/null; then
    log_pass "Token issuance fuzz test completed"
else
    log_error "Token issuance fuzz test failed"
fi

log_test "Token validation fuzz testing..."
if go test -fuzz=FuzzTokenValidation -fuzztime=30s ./test/fuzz/ 2>/dev/null; then
    log_pass "Token validation fuzz test completed"
else
    log_error "Token validation fuzz test failed"
fi

log_test "Delegation flow fuzz testing..."
if go test -fuzz=FuzzDelegationFlow -fuzztime=30s ./test/fuzz/ 2>/dev/null; then
    log_pass "Delegation flow fuzz test completed"
else
    log_error "Delegation flow fuzz test failed"
fi

# 2. Run penetration tests
echo -e "\n🛡️ Running Penetration Tests"
echo "============================"

log_test "Privilege escalation tests..."
if go test -v ./test/pentest/ -run TestPrivilegeEscalation 2>/dev/null; then
    log_pass "Privilege escalation tests passed"
else
    log_error "Privilege escalation tests failed"
fi

log_test "Token replay attack tests..."
if go test -v ./test/pentest/ -run TestTokenReplayAttack 2>/dev/null; then
    log_pass "Token replay attack tests passed"
else
    log_error "Token replay attack tests failed"
fi

log_test "HTTP injection attack tests..."
if go test -v ./test/pentest/ -run TestHTTPInjectionAttacks 2>/dev/null; then
    log_pass "HTTP injection attack tests passed"
else
    log_error "HTTP injection attack tests failed"
fi

log_test "Denial of service tests..."
if go test -v ./test/pentest/ -run TestDenialOfService 2>/dev/null; then
    log_pass "DoS tests passed"
else
    log_error "DoS tests failed"
fi

log_test "Concurrent attack tests..."
if go test -v ./test/pentest/ -run TestConcurrentAttacks 2>/dev/null; then
    log_pass "Concurrent attack tests passed"
else
    log_error "Concurrent attack tests failed"
fi

# 3. Audit security tests
echo -e "\n📋 Running Audit Security Tests"
echo "==============================="

log_test "Audit log tampering tests..."
if go test -v ./test/pentest/ -run TestAuditLogTampering 2>/dev/null; then
    log_pass "Audit log tampering tests passed"
else
    log_error "Audit log tampering tests failed"
fi

log_test "Audit log integrity tests..."
if go test -v ./test/pentest/ -run TestAuditLogIntegrity 2>/dev/null; then
    log_pass "Audit log integrity tests passed"
else
    log_error "Audit log integrity tests failed"
fi

# 4. Static security analysis
echo -e "\n🔍 Running Static Security Analysis"
echo "==================================="

log_test "Checking for hardcoded secrets..."
if grep -r "password\|secret\|key" --include="*.go" pkg/ internal/ | grep -v "// " | grep -v test; then
    log_warning "Potential hardcoded credentials found"
else
    log_pass "No hardcoded credentials detected"
fi

log_test "Checking for SQL injection patterns..."
if grep -r "fmt.Sprintf.*SELECT\|fmt.Sprintf.*INSERT\|fmt.Sprintf.*UPDATE\|fmt.Sprintf.*DELETE" --include="*.go" pkg/ internal/; then
    log_error "Potential SQL injection vulnerability found"
else
    log_pass "No SQL injection patterns detected"
fi

log_test "Checking for path traversal patterns..."
if grep -r "\.\./\.\./\|filepath.Join.*\.\." --include="*.go" pkg/ internal/; then
    log_warning "Potential path traversal patterns found"
else
    log_pass "No path traversal patterns detected"
fi

# 5. Dependency security check
echo -e "\n📦 Running Dependency Security Check"
echo "===================================="

log_test "Checking for known vulnerable dependencies..."
if command -v nancy &> /dev/null; then
    if go list -json -m all | nancy sleuth; then
        log_pass "No known vulnerable dependencies"
    else
        log_error "Vulnerable dependencies detected"
    fi
else
    log_warning "Nancy security scanner not installed"
fi

# 6. TLS/Crypto security check
echo -e "\n🔐 Running Crypto Security Check"
echo "==============================="

log_test "Checking for weak cryptographic practices..."
if grep -r "MD5\|SHA1\|DES\|RC4" --include="*.go" pkg/ internal/; then
    log_error "Weak cryptographic algorithms detected"
else
    log_pass "No weak cryptographic algorithms detected"
fi

log_test "Checking for proper random number generation..."
if grep -r "math/rand" --include="*.go" pkg/ internal/ | grep -v crypto; then
    log_warning "Non-cryptographic random number generation detected"
else
    log_pass "Proper cryptographic random number generation"
fi

# 7. Performance security tests
echo -e "\n⚡ Running Performance Security Tests"
echo "===================================="

log_test "Testing for algorithmic complexity attacks..."
# This would run performance tests to detect potential DoS via complexity
echo "Running complexity analysis..."
go test -bench=. -benchtime=5s ./pkg/... > /dev/null 2>&1
if [ $? -eq 0 ]; then
    log_pass "Performance tests completed"
else
    log_warning "Some performance tests failed"
fi

# Summary
echo -e "\n📊 Security Test Summary"
echo "======================="
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo -e "Warnings: ${YELLOW}$WARNINGS${NC}"

if [ $FAILED -gt 0 ]; then
    echo -e "\n${RED}❌ Security tests failed. Review and fix issues before deployment.${NC}"
    exit 1
elif [ $WARNINGS -gt 0 ]; then
    echo -e "\n${YELLOW}⚠️  Security tests passed with warnings. Review recommendations.${NC}"
    exit 0
else
    echo -e "\n${GREEN}✅ All security tests passed!${NC}"
    exit 0
fi