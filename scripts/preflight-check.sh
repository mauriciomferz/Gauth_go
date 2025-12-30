#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  AgentAuth CI/CD Pre-Flight Checklist${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Track overall status
CHECKS_PASSED=0
CHECKS_FAILED=0
CHECKS_WARNING=0

# Function to print status
check_status() {
    local status=$1
    local message=$2
    local details=$3
    
    if [ "$status" == "pass" ]; then
        echo -e "${GREEN}✅ PASS${NC}: $message"
        [ -n "$details" ] && echo -e "   ${details}"
        ((CHECKS_PASSED++))
    elif [ "$status" == "fail" ]; then
        echo -e "${RED}❌ FAIL${NC}: $message"
        [ -n "$details" ] && echo -e "   ${details}"
        ((CHECKS_FAILED++))
    elif [ "$status" == "warn" ]; then
        echo -e "${YELLOW}⚠️  WARN${NC}: $message"
        [ -n "$details" ] && echo -e "   ${details}"
        ((CHECKS_WARNING++))
    else
        echo -e "${BLUE}ℹ️  INFO${NC}: $message"
        [ -n "$details" ] && echo -e "   ${details}"
    fi
}

echo -e "${BLUE}[1/7] Checking Local Tools...${NC}"
echo ""

# Check Docker
if command -v docker &> /dev/null; then
    DOCKER_VERSION=$(docker --version | cut -d ' ' -f3 | cut -d ',' -f1)
    check_status "pass" "Docker installed" "Version: $DOCKER_VERSION"
    
    # Check if Docker daemon is running
    if docker info &> /dev/null; then
        check_status "pass" "Docker daemon running"
    else
        check_status "fail" "Docker daemon not running" "Start Docker Desktop or run: sudo systemctl start docker"
    fi
else
    check_status "fail" "Docker not installed" "Install: https://docs.docker.com/get-docker/"
fi

# Check kubectl
if command -v kubectl &> /dev/null; then
    KUBECTL_VERSION=$(kubectl version --client -o json 2>/dev/null | grep -o '"gitVersion":"[^"]*"' | head -1 | cut -d '"' -f4)
    check_status "pass" "kubectl installed" "Version: $KUBECTL_VERSION"
else
    check_status "fail" "kubectl not installed" "Install: brew install kubectl"
fi

# Check git
if command -v git &> /dev/null; then
    GIT_VERSION=$(git --version | cut -d ' ' -f3)
    check_status "pass" "git installed" "Version: $GIT_VERSION"
else
    check_status "fail" "git not installed" "Install: brew install git"
fi

# Check gh CLI (optional)
if command -v gh &> /dev/null; then
    GH_VERSION=$(gh --version | head -1 | cut -d ' ' -f3)
    check_status "pass" "GitHub CLI installed" "Version: $GH_VERSION"
else
    check_status "warn" "GitHub CLI not installed (optional)" "Install: brew install gh"
fi

echo ""
echo -e "${BLUE}[2/7] Checking GitHub Repository...${NC}"
echo ""

# Check if in git repository
if git rev-parse --is-inside-work-tree &> /dev/null; then
    check_status "pass" "Inside git repository"
    
    # Check remote
    REMOTE_URL=$(git config --get remote.origin.url 2>/dev/null || echo "")
    if [ -n "$REMOTE_URL" ]; then
        check_status "pass" "Git remote configured" "URL: $REMOTE_URL"
    else
        check_status "fail" "No git remote configured" "Add remote: git remote add origin <url>"
    fi
    
    # Check branch
    CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
    check_status "info" "Current branch" "$CURRENT_BRANCH"
    
    # Check uncommitted changes
    if git diff-index --quiet HEAD --; then
        check_status "pass" "No uncommitted changes"
    else
        check_status "warn" "Uncommitted changes detected" "Run: git status"
    fi
else
    check_status "fail" "Not inside a git repository"
fi

echo ""
echo -e "${BLUE}[3/7] Checking Workflow Files...${NC}"
echo ""

# Check workflow file exists
if [ -f ".github/workflows/deploy-staging.yml" ]; then
    check_status "pass" "Workflow file exists" ".github/workflows/deploy-staging.yml"
    
    # Validate YAML syntax
    if python3 -c "import yaml; yaml.safe_load(open('.github/workflows/deploy-staging.yml'))" 2>/dev/null; then
        check_status "pass" "Workflow YAML syntax valid"
    else
        check_status "fail" "Workflow YAML syntax invalid" "Check syntax errors"
    fi
else
    check_status "fail" "Workflow file not found" ".github/workflows/deploy-staging.yml"
fi

# Check Kubernetes manifests
MANIFESTS=(
    "deployments/k8s/staging/namespace.yaml"
    "deployments/k8s/staging/configmap.yaml"
    "deployments/k8s/staging/secrets.yaml"
    "deployments/k8s/staging/deployment.yaml"
    "deployments/k8s/staging/service.yaml"
    "deployments/k8s/staging/ingress.yaml"
    "deployments/k8s/staging/hpa-pdb-rbac-netpol.yaml"
    "deployments/k8s/staging/postgres-statefulset.yaml"
    "deployments/k8s/staging/redis-statefulset.yaml"
)

MISSING_MANIFESTS=0
for manifest in "${MANIFESTS[@]}"; do
    if [ -f "$manifest" ]; then
        : # Manifest exists (no output to reduce noise)
    else
        check_status "fail" "Manifest not found" "$manifest"
        ((MISSING_MANIFESTS++))
    fi
done

if [ $MISSING_MANIFESTS -eq 0 ]; then
    check_status "pass" "All Kubernetes manifests present" "${#MANIFESTS[@]} files"
fi

echo ""
echo -e "${BLUE}[4/7] Checking Docker Registry Access...${NC}"
echo ""

# Prompt for Docker registry
read -p "Enter Docker registry (e.g., ghcr.io, or press Enter to skip): " DOCKER_REGISTRY
if [ -n "$DOCKER_REGISTRY" ]; then
    read -p "Enter Docker username: " DOCKER_USERNAME
    read -sp "Enter Docker password/token: " DOCKER_PASSWORD
    echo ""
    
    # Test Docker login
    if echo "$DOCKER_PASSWORD" | docker login "$DOCKER_REGISTRY" -u "$DOCKER_USERNAME" --password-stdin &> /dev/null; then
        check_status "pass" "Docker registry login successful" "$DOCKER_REGISTRY"
    else
        check_status "fail" "Docker registry login failed" "Check credentials"
    fi
else
    check_status "warn" "Docker registry check skipped"
fi

echo ""
echo -e "${BLUE}[5/7] Checking Kubernetes Cluster Access...${NC}"
echo ""

# Test kubectl connectivity
if kubectl cluster-info &> /dev/null; then
    CLUSTER_SERVER=$(kubectl config view -o jsonpath='{.clusters[0].cluster.server}')
    check_status "pass" "Kubernetes cluster accessible" "Server: $CLUSTER_SERVER"
    
    # Check namespace
    if kubectl get namespace gauth-staging &> /dev/null; then
        check_status "pass" "Namespace gauth-staging exists"
    else
        check_status "warn" "Namespace gauth-staging not found" "Create: kubectl apply -f deployments/k8s/staging/namespace.yaml"
    fi
    
    # Check NGINX Ingress
    if kubectl get namespace ingress-nginx &> /dev/null; then
        check_status "pass" "NGINX Ingress Controller namespace exists"
    else
        check_status "warn" "NGINX Ingress Controller not found" "Install: helm install ingress-nginx ingress-nginx/ingress-nginx --create-namespace -n ingress-nginx"
    fi
    
    # Check cert-manager
    if kubectl get namespace cert-manager &> /dev/null; then
        check_status "pass" "cert-manager namespace exists"
    else
        check_status "warn" "cert-manager not found" "Install: helm install cert-manager jetstack/cert-manager --create-namespace -n cert-manager --set installCRDs=true"
    fi
    
    # Check metrics-server
    if kubectl top nodes &> /dev/null; then
        check_status "pass" "metrics-server working"
    else
        check_status "warn" "metrics-server not working" "Install: helm install metrics-server metrics-server/metrics-server -n kube-system"
    fi
else
    check_status "fail" "Cannot connect to Kubernetes cluster" "Check kubeconfig: kubectl cluster-info"
fi

echo ""
echo -e "${BLUE}[6/7] Checking GitHub Secrets Configuration...${NC}"
echo ""

check_status "info" "GitHub Secrets cannot be verified locally"
echo -e "   ${YELLOW}Please verify the following secrets are configured:${NC}"
echo -e "   1. DOCKER_REGISTRY (e.g., ghcr.io)"
echo -e "   2. DOCKER_USERNAME (registry username)"
echo -e "   3. DOCKER_PASSWORD (PAT or access token)"
echo -e "   4. KUBE_CONFIG_STAGING (base64-encoded kubeconfig)"
echo -e "   5. SLACK_WEBHOOK_URL (Slack webhook URL)"
echo -e "   6. CODECOV_TOKEN (optional)"
echo ""
echo -e "   ${BLUE}Verify at:${NC} https://github.com/$(git config --get remote.origin.url | sed 's/.*github.com[:/]\(.*\)\.git/\1/')/settings/secrets/actions"
echo ""

read -p "Are all GitHub secrets configured? (y/N): " SECRETS_CONFIGURED
if [[ "$SECRETS_CONFIGURED" =~ ^[Yy]$ ]]; then
    check_status "pass" "GitHub secrets configured (user confirmed)"
else
    check_status "fail" "GitHub secrets not configured" "Configure secrets before proceeding"
fi

echo ""
echo -e "${BLUE}[7/7] Checking Slack Webhook...${NC}"
echo ""

read -p "Enter Slack webhook URL (or press Enter to skip): " SLACK_WEBHOOK_URL
if [ -n "$SLACK_WEBHOOK_URL" ]; then
    # Test Slack webhook
    RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" -X POST -H 'Content-type: application/json' \
        --data '{"text":"✅ AgentAuth CI/CD pre-flight check successful"}' \
        "$SLACK_WEBHOOK_URL")
    
    if [ "$RESPONSE" == "200" ]; then
        check_status "pass" "Slack webhook working" "Test message sent"
    else
        check_status "fail" "Slack webhook failed" "HTTP $RESPONSE"
    fi
else
    check_status "warn" "Slack webhook check skipped"
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Pre-Flight Check Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${GREEN}✅ Passed:${NC} $CHECKS_PASSED"
echo -e "${YELLOW}⚠️  Warnings:${NC} $CHECKS_WARNING"
echo -e "${RED}❌ Failed:${NC} $CHECKS_FAILED"
echo ""

if [ $CHECKS_FAILED -gt 0 ]; then
    echo -e "${RED}❌ Pre-flight check FAILED${NC}"
    echo -e "   Fix the failed checks above before proceeding."
    exit 1
elif [ $CHECKS_WARNING -gt 0 ]; then
    echo -e "${YELLOW}⚠️  Pre-flight check completed with WARNINGS${NC}"
    echo -e "   You may proceed, but some optional features may not work."
    echo ""
    read -p "Continue anyway? (y/N): " CONTINUE
    if [[ "$CONTINUE" =~ ^[Yy]$ ]]; then
        echo -e "${GREEN}✅ Proceeding...${NC}"
        exit 0
    else
        echo -e "${YELLOW}⚠️  Aborted by user${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}✅ All checks PASSED${NC}"
    echo -e "   You can safely push to GitHub to trigger the CI/CD pipeline."
    echo ""
    echo -e "${BLUE}Next steps:${NC}"
    echo -e "   1. git push origin main"
    echo -e "   2. Monitor workflow: https://github.com/$(git config --get remote.origin.url | sed 's/.*github.com[:/]\(.*\)\.git/\1/')/actions"
    exit 0
fi
