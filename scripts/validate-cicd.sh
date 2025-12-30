#!/bin/bash

# AgentAuth CI/CD Quick Validation Script
# This script validates the CI/CD setup without requiring user interaction

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  AgentAuth CI/CD Quick Validation${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 1. Check workflow YAML syntax
echo -e "${BLUE}[1/4] Validating Workflow YAML Syntax...${NC}"
if python3 -c "import yaml; yaml.safe_load(open('.github/workflows/deploy-staging.yml'))" 2>/dev/null; then
    echo -e "${GREEN}✅ Workflow YAML syntax valid${NC}"
else
    echo -e "${RED}❌ Workflow YAML syntax invalid${NC}"
    exit 1
fi

# 2. Check required files exist
echo -e "${BLUE}[2/4] Checking Required Files...${NC}"
REQUIRED_FILES=(
    ".github/workflows/deploy-staging.yml"
    "deployments/k8s/staging/namespace.yaml"
    "deployments/k8s/staging/configmap.yaml"
    "deployments/k8s/staging/secrets.yaml"
    "deployments/k8s/staging/deployment.yaml"
    "deployments/k8s/staging/service.yaml"
    "deployments/k8s/staging/ingress.yaml"
    "deployments/k8s/staging/postgres-statefulset.yaml"
    "deployments/k8s/staging/redis-statefulset.yaml"
    "deployments/k8s/staging/bluegreen/gauth-deployment-blue.yaml"
    "deployments/k8s/staging/bluegreen/gauth-deployment-green.yaml"
    "deployments/k8s/staging/bluegreen/gauth-services.yaml"
    "deployments/k8s/staging/bluegreen/gauth-ingress-bluegreen.yaml"
    "deployments/k8s/staging/bluegreen/switch-traffic.sh"
    "Dockerfile"
)

ALL_PRESENT=true
for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        : # File exists (silent)
    else
        echo -e "${RED}❌ Missing: $file${NC}"
        ALL_PRESENT=false
    fi
done

if [ "$ALL_PRESENT" = true ]; then
    echo -e "${GREEN}✅ All required files present (${#REQUIRED_FILES[@]} files)${NC}"
else
    exit 1
fi

# 3. Check git status
echo -e "${BLUE}[3/4] Checking Git Status...${NC}"
if git rev-parse --is-inside-work-tree &> /dev/null; then
    REMOTE_URL=$(git config --get remote.origin.url 2>/dev/null || echo "")
    CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
    
    echo -e "${GREEN}✅ Git repository OK${NC}"
    echo -e "   Remote: $REMOTE_URL"
    echo -e "   Branch: $CURRENT_BRANCH"
    
    # Check for uncommitted changes
    if git diff-index --quiet HEAD --; then
        echo -e "${GREEN}✅ No uncommitted changes${NC}"
    else
        echo -e "${YELLOW}⚠️  Uncommitted changes detected${NC}"
        git status --short | head -10
    fi
else
    echo -e "${RED}❌ Not a git repository${NC}"
    exit 1
fi

# 4. Generate setup instructions
echo -e "${BLUE}[4/4] Setup Instructions...${NC}"
echo ""
echo -e "${YELLOW}Before pushing to GitHub, configure these secrets:${NC}"
echo -e "   GitHub Repository Settings → Secrets and variables → Actions"
echo ""
echo -e "   ${BLUE}1. DOCKER_REGISTRY${NC}"
echo -e "      Example: ghcr.io"
echo ""
echo -e "   ${BLUE}2. DOCKER_USERNAME${NC}"
echo -e "      Example: mauriciomferz"
echo ""
echo -e "   ${BLUE}3. DOCKER_PASSWORD${NC}"
echo -e "      GitHub PAT with 'write:packages' scope"
echo -e "      Generate: https://github.com/settings/tokens/new"
echo ""
echo -e "   ${BLUE}4. KUBE_CONFIG_STAGING${NC}"
echo -e "      Base64-encoded kubeconfig"
echo -e "      Generate: cat ~/.kube/config | base64 | pbcopy"
echo ""
echo -e "   ${BLUE}5. SLACK_WEBHOOK_URL${NC}"
echo -e "      Slack incoming webhook URL"
echo -e "      Create: https://api.slack.com/apps"
echo ""
echo -e "   ${BLUE}6. CODECOV_TOKEN${NC} (optional)"
echo -e "      Codecov upload token"
echo -e "      Get: https://codecov.io/"
echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Validation Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${BLUE}Next Steps:${NC}"
echo ""
echo -e "   ${YELLOW}1. Configure GitHub Secrets${NC}"
echo -e "      https://github.com/mauriciomferz/Gauth_go/settings/secrets/actions"
echo ""
echo -e "   ${YELLOW}2. Review setup guide${NC}"
echo -e "      cat deployments/GITHUB_ACTIONS_SETUP.md"
echo ""
echo -e "   ${YELLOW}3. Push to GitHub${NC}"
echo -e "      git push origin main"
echo ""
echo -e "   ${YELLOW}4. Monitor workflow${NC}"
echo -e "      https://github.com/mauriciomferz/Gauth_go/actions"
echo ""
