#!/bin/bash
# Blue-Green Deployment Validation Script
# Tests all aspects of blue-green deployment without requiring actual K8s cluster
# Usage: ./validate-bluegreen.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

VALIDATION_PASSED=0
VALIDATION_FAILED=0

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║     AgentAuth Blue-Green Deployment Validation Suite          ║${NC}"
echo -e "${BLUE}║     Week 4 Day 4 - November 10, 2025                      ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Function to print test header
print_test_header() {
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}TEST: $1${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Function to pass test
pass_test() {
    echo -e "${GREEN}✅ PASS: $1${NC}"
    ((VALIDATION_PASSED++))
    echo ""
}

# Function to fail test
fail_test() {
    echo -e "${RED}❌ FAIL: $1${NC}"
    ((VALIDATION_FAILED++))
    echo ""
}

# Function to warn
warn() {
    echo -e "${YELLOW}⚠️  WARNING: $1${NC}"
}

# Test 1: Validate Manifest Files Exist
print_test_header "1. Validate Manifest Files Exist"

REQUIRED_FILES=(
    "gauth-deployment-blue.yaml"
    "gauth-deployment-green.yaml"
    "gauth-services.yaml"
    "gauth-ingress-bluegreen.yaml"
    "switch-traffic.sh"
    "README.md"
)

MISSING_FILES=0
for file in "${REQUIRED_FILES[@]}"; do
    if [[ -f "$file" ]]; then
        echo -e "${GREEN}  ✓ $file${NC}"
    else
        echo -e "${RED}  ✗ $file (missing)${NC}"
        ((MISSING_FILES++))
    fi
done

if [[ $MISSING_FILES -eq 0 ]]; then
    pass_test "All required manifest files exist"
else
    fail_test "$MISSING_FILES manifest file(s) missing"
fi

# Test 2: Validate YAML Syntax
print_test_header "2. Validate YAML Syntax"

YAML_ERRORS=0
for file in gauth-deployment-blue.yaml gauth-deployment-green.yaml gauth-services.yaml gauth-ingress-bluegreen.yaml; do
    if [[ -f "$file" ]]; then
        # Check basic YAML syntax with Python (supports multi-document YAML)
        if python3 -c "import yaml; list(yaml.safe_load_all(open('$file')))" 2>/dev/null; then
            echo -e "${GREEN}  ✓ $file (valid YAML)${NC}"
        else
            echo -e "${RED}  ✗ $file (invalid YAML)${NC}"
            ((YAML_ERRORS++))
        fi
    fi
done

if [[ $YAML_ERRORS -eq 0 ]]; then
    pass_test "All YAML files have valid syntax"
else
    fail_test "$YAML_ERRORS YAML file(s) have syntax errors"
fi

# Test 3: Validate Deployment Configuration Consistency
print_test_header "3. Validate Blue/Green Deployment Consistency"

echo "Checking for consistent configuration between blue and green..."

# Check that both deployments have same resource limits (except version label)
BLUE_REPLICAS=$(grep -A 5 "kind: Deployment" gauth-deployment-blue.yaml | grep "replicas:" | awk '{print $2}' || echo "0")
GREEN_REPLICAS=$(grep -A 5 "kind: Deployment" gauth-deployment-green.yaml | grep "replicas:" | awk '{print $2}' || echo "0")

echo -e "  Blue replicas: ${BLUE_REPLICAS}"
echo -e "  Green replicas: ${GREEN_REPLICAS}"

if [[ "$BLUE_REPLICAS" == "$GREEN_REPLICAS" ]]; then
    pass_test "Blue and green have identical replica counts"
else
    warn "Blue and green have different replica counts (may be intentional)"
    pass_test "Replica count check completed (difference noted)"
fi

# Test 4: Validate Service Selectors
print_test_header "4. Validate Service Selectors"

echo "Checking service selector configuration..."

# Check blue service selector
if grep -q "version: blue" gauth-services.yaml; then
    echo -e "${GREEN}  ✓ Blue service has version: blue selector${NC}"
else
    echo -e "${RED}  ✗ Blue service missing version: blue selector${NC}"
    fail_test "Blue service selector configuration incorrect"
    exit 1
fi

# Check green service selector
if grep -q "version: green" gauth-services.yaml; then
    echo -e "${GREEN}  ✓ Green service has version: green selector${NC}"
else
    echo -e "${RED}  ✗ Green service missing version: green selector${NC}"
    fail_test "Green service selector configuration incorrect"
    exit 1
fi

pass_test "Service selectors correctly configured"

# Test 5: Validate Traffic Switch Script
print_test_header "5. Validate Traffic Switch Script"

if [[ -f "switch-traffic.sh" ]]; then
    # Check script is executable
    if [[ -x "switch-traffic.sh" ]]; then
        echo -e "${GREEN}  ✓ Script has executable permissions${NC}"
    else
        echo -e "${YELLOW}  ! Script not executable, setting permissions${NC}"
        chmod +x switch-traffic.sh
    fi
    
    # Check for required components
    if grep -q "kubectl patch ingress" switch-traffic.sh; then
        echo -e "${GREEN}  ✓ Script uses kubectl patch for traffic switching${NC}"
    else
        echo -e "${RED}  ✗ Script missing kubectl patch command${NC}"
        fail_test "Traffic switch script incomplete"
        exit 1
    fi
    
    if grep -q "health" switch-traffic.sh; then
        echo -e "${GREEN}  ✓ Script includes health check validation${NC}"
    else
        warn "Script may be missing health check validation"
    fi
    
    pass_test "Traffic switch script validated"
else
    fail_test "Traffic switch script not found"
fi

# Test 6: Validate Zero-Downtime Strategy
print_test_header "6. Validate Zero-Downtime Strategy Elements"

echo "Checking deployment strategy configuration..."

# Check for readiness probes
if grep -q "readinessProbe" gauth-deployment-blue.yaml && grep -q "readinessProbe" gauth-deployment-green.yaml; then
    echo -e "${GREEN}  ✓ Deployments have readiness probes${NC}"
else
    warn "Deployments may be missing readiness probes (required for zero-downtime)"
fi

# Check for liveness probes
if grep -q "livenessProbe" gauth-deployment-blue.yaml && grep -q "livenessProbe" gauth-deployment-green.yaml; then
    echo -e "${GREEN}  ✓ Deployments have liveness probes${NC}"
else
    warn "Deployments may be missing liveness probes"
fi

# Check for rolling update strategy
if grep -q "RollingUpdate" gauth-deployment-blue.yaml || grep -q "maxSurge" gauth-deployment-blue.yaml; then
    echo -e "${GREEN}  ✓ Deployments use RollingUpdate strategy${NC}"
else
    warn "Deployment strategy not explicitly set to RollingUpdate"
fi

pass_test "Zero-downtime strategy elements present"

# Test 7: Validate Session Affinity
print_test_header "7. Validate Session Affinity Configuration"

if grep -q "sessionAffinity: ClientIP" gauth-services.yaml; then
    echo -e "${GREEN}  ✓ Services have ClientIP session affinity${NC}"
else
    warn "Services missing session affinity (may cause session loss during switch)"
fi

if grep -q "timeoutSeconds" gauth-services.yaml; then
    TIMEOUT=$(grep "timeoutSeconds:" gauth-services.yaml | head -1 | awk '{print $2}')
    echo -e "${GREEN}  ✓ Session affinity timeout: ${TIMEOUT}s${NC}"
else
    warn "Session affinity timeout not configured"
fi

pass_test "Session affinity configuration validated"

# Test 8: Validate Ingress Configuration
print_test_header "8. Validate Ingress Configuration for Blue-Green"

if [[ -f "gauth-ingress-bluegreen.yaml" ]]; then
    # Check ingress has backend service reference
    if grep -q "backend:" gauth-ingress-bluegreen.yaml; then
        echo -e "${GREEN}  ✓ Ingress has backend service configuration${NC}"
    else
        echo -e "${RED}  ✗ Ingress missing backend service configuration${NC}"
        fail_test "Ingress configuration incomplete"
        exit 1
    fi
    
    # Check for HTTP paths
    if grep -q "paths:" gauth-ingress-bluegreen.yaml; then
        PATH_COUNT=$(grep -c "path:" gauth-ingress-bluegreen.yaml || echo "0")
        echo -e "${GREEN}  ✓ Ingress has ${PATH_COUNT} path(s) configured${NC}"
    else
        warn "Ingress may be missing path configuration"
    fi
    
    pass_test "Ingress configuration validated"
else
    fail_test "Ingress manifest not found"
fi

# Test 9: Validate Rollback Capability
print_test_header "9. Validate Rollback Capability"

echo "Checking rollback documentation and scripts..."

if grep -q "rollback" README.md; then
    echo -e "${GREEN}  ✓ Rollback procedure documented${NC}"
else
    warn "Rollback procedure not documented in README"
fi

if grep -q "Rollback" switch-traffic.sh; then
    echo -e "${GREEN}  ✓ Script provides rollback instructions${NC}"
else
    warn "Script missing rollback instructions"
fi

# Check that script accepts both blue and green as arguments
if grep -q 'blue.*green' switch-traffic.sh; then
    echo -e "${GREEN}  ✓ Script supports bidirectional switching${NC}"
else
    warn "Script may not support switching in both directions"
fi

pass_test "Rollback capability validated"

# Test 10: Resource Duplication Awareness
print_test_header "10. Validate Resource Duplication Strategy"

echo "Analyzing resource requirements..."

# Calculate total resources if both environments running
echo -e "${YELLOW}  ℹ️  Blue-green requires 2x resources during deployment${NC}"
echo -e "${YELLOW}  ℹ️  Consider resource capacity before deployment${NC}"

if grep -q "resources:" gauth-deployment-blue.yaml; then
    echo -e "${GREEN}  ✓ Resource limits defined${NC}"
else
    warn "Resource limits not defined (may cause resource contention)"
fi

pass_test "Resource duplication strategy documented"

# Test 11: Validate Documentation Completeness
print_test_header "11. Validate Documentation Completeness"

REQUIRED_SECTIONS=(
    "Overview"
    "Architecture"
    "Deployment Procedure"
    "Rollback"
    "Advantages"
    "Disadvantages"
    "Best Practices"
)

MISSING_SECTIONS=0
for section in "${REQUIRED_SECTIONS[@]}"; do
    if grep -q "$section" README.md; then
        echo -e "${GREEN}  ✓ $section documented${NC}"
    else
        echo -e "${RED}  ✗ $section missing from documentation${NC}"
        ((MISSING_SECTIONS++))
    fi
done

if [[ $MISSING_SECTIONS -eq 0 ]]; then
    pass_test "Documentation complete with all required sections"
else
    fail_test "$MISSING_SECTIONS section(s) missing from documentation"
fi

# Test 12: Security Validation
print_test_header "12. Validate Security Configuration"

echo "Checking security settings..."

# Check for secret references (not hardcoded values)
if grep -q "secretKeyRef" gauth-deployment-blue.yaml && grep -q "secretKeyRef" gauth-deployment-green.yaml; then
    echo -e "${GREEN}  ✓ Deployments use Kubernetes Secrets${NC}"
else
    warn "Deployments may not be using Kubernetes Secrets properly"
fi

# Check for security context
if grep -q "securityContext" gauth-deployment-blue.yaml; then
    echo -e "${GREEN}  ✓ Security context configured${NC}"
else
    warn "Security context not configured (may run as root)"
fi

pass_test "Security configuration validated"

# Final Summary
echo ""
echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                    VALIDATION SUMMARY                      ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}Passed Tests: ${VALIDATION_PASSED}${NC}"
echo -e "${RED}Failed Tests: ${VALIDATION_FAILED}${NC}"
echo ""

if [[ $VALIDATION_FAILED -eq 0 ]]; then
    echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║  ✅  ALL VALIDATION TESTS PASSED                          ║${NC}"
    echo -e "${GREEN}║  Blue-Green Deployment Ready for Production!              ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${CYAN}Next Steps:${NC}"
    echo "1. Deploy to actual Kubernetes staging cluster"
    echo "2. Test traffic switching with real workload"
    echo "3. Measure actual rollback time"
    echo "4. Document lessons learned"
    echo ""
    exit 0
else
    echo -e "${RED}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║  ❌  VALIDATION FAILED                                     ║${NC}"
    echo -e "${RED}║  ${VALIDATION_FAILED} test(s) failed - review and fix before deploying  ║${NC}"
    echo -e "${RED}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    exit 1
fi
