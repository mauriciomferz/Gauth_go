#!/bin/bash
# Traffic switching script for blue-green deployment
# Usage: ./switch-traffic.sh [blue|green]

set -e

NAMESPACE="agentauth-staging"
TARGET_VERSION="${1:-blue}"
INGRESS_NAME="agentauth-ingress"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== AgentAuth Blue-Green Traffic Switcher ===${NC}"
echo ""

# Validate input
if [[ "$TARGET_VERSION" != "blue" && "$TARGET_VERSION" != "green" ]]; then
    echo -e "${RED}❌ Error: Target version must be 'blue' or 'green'${NC}"
    echo "Usage: $0 [blue|green]"
    exit 1
fi

# Check kubectl connectivity
if ! kubectl cluster-info &> /dev/null; then
    echo -e "${RED}❌ Error: Cannot connect to Kubernetes cluster${NC}"
    echo "Please configure kubectl and try again"
    exit 1
fi

# Check if target deployment exists and is ready
echo -e "${YELLOW}📋 Checking target deployment status...${NC}"
if ! kubectl get deployment "agentauth-deployment-${TARGET_VERSION}" -n "$NAMESPACE" &> /dev/null; then
    echo -e "${RED}❌ Error: Deployment 'agentauth-deployment-${TARGET_VERSION}' not found${NC}"
    exit 1
fi

# Check if target deployment is ready
READY_REPLICAS=$(kubectl get deployment "agentauth-deployment-${TARGET_VERSION}" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}')
DESIRED_REPLICAS=$(kubectl get deployment "agentauth-deployment-${TARGET_VERSION}" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')

if [[ "$READY_REPLICAS" != "$DESIRED_REPLICAS" ]]; then
    echo -e "${RED}❌ Error: Target deployment not ready (${READY_REPLICAS}/${DESIRED_REPLICAS} replicas ready)${NC}"
    echo "Wait for rollout to complete before switching traffic"
    exit 1
fi

echo -e "${GREEN}✓ Target deployment is ready (${READY_REPLICAS}/${DESIRED_REPLICAS} replicas)${NC}"

# Get current active version
CURRENT_SERVICE=$(kubectl get ingress "$INGRESS_NAME" -n "$NAMESPACE" -o jsonpath='{.spec.rules[0].http.paths[0].backend.service.name}')
if [[ "$CURRENT_SERVICE" == *"blue"* ]]; then
    CURRENT_VERSION="blue"
elif [[ "$CURRENT_SERVICE" == *"green"* ]]; then
    CURRENT_VERSION="green"
else
    CURRENT_VERSION="unknown"
fi

echo -e "${BLUE}Current active version: ${CURRENT_VERSION}${NC}"
echo -e "${BLUE}Target version: ${TARGET_VERSION}${NC}"
echo ""

# Confirm switch
if [[ "$CURRENT_VERSION" == "$TARGET_VERSION" ]]; then
    echo -e "${YELLOW}⚠️  Target version is already active. No switch needed.${NC}"
    exit 0
fi

read -p "$(echo -e ${YELLOW}Switch traffic from ${CURRENT_VERSION} to ${TARGET_VERSION}? [y/N]:${NC} )" -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${RED}❌ Traffic switch cancelled${NC}"
    exit 1
fi

# Perform traffic switch by updating ingress
echo -e "${YELLOW}🔄 Switching traffic to ${TARGET_VERSION}...${NC}"

# Method 1: Update ingress backend service
kubectl patch ingress "$INGRESS_NAME" -n "$NAMESPACE" --type=json -p="[
  {
    \"op\": \"replace\",
    \"path\": \"/spec/rules/0/http/paths/0/backend/service/name\",
    \"value\": \"agentauth-service-${TARGET_VERSION}\"
  },
  {
    \"op\": \"replace\",
    \"path\": \"/spec/rules/0/http/paths/1/backend/service/name\",
    \"value\": \"agentauth-service-${TARGET_VERSION}\"
  },
  {
    \"op\": \"replace\",
    \"path\": \"/spec/rules/0/http/paths/2/backend/service/name\",
    \"value\": \"agentauth-service-${TARGET_VERSION}\"
  },
  {
    \"op\": \"replace\",
    \"path\": \"/spec/rules/0/http/paths/3/backend/service/name\",
    \"value\": \"agentauth-service-${TARGET_VERSION}\"
  }
]"

# Wait for ingress to update
echo -e "${YELLOW}⏳ Waiting for ingress controller to update (10s)...${NC}"
sleep 10

# Verify switch
NEW_SERVICE=$(kubectl get ingress "$INGRESS_NAME" -n "$NAMESPACE" -o jsonpath='{.spec.rules[0].http.paths[0].backend.service.name}')
if [[ "$NEW_SERVICE" == "agentauth-service-${TARGET_VERSION}" ]]; then
    echo -e "${GREEN}✅ Traffic successfully switched to ${TARGET_VERSION}${NC}"
else
    echo -e "${RED}❌ Error: Traffic switch failed${NC}"
    exit 1
fi

# Health check
echo -e "${YELLOW}🏥 Running health checks on ${TARGET_VERSION}...${NC}"
INGRESS_IP=$(kubectl get ingress "$INGRESS_NAME" -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

if [[ -z "$INGRESS_IP" ]]; then
    INGRESS_IP=$(kubectl get ingress "$INGRESS_NAME" -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
fi

if [[ -n "$INGRESS_IP" ]]; then
    if curl -sf -k "https://${INGRESS_IP}/healthz" > /dev/null; then
        echo -e "${GREEN}✅ Health check passed${NC}"
    else
        echo -e "${RED}⚠️  Health check failed - manual verification recommended${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Could not determine ingress IP - manual verification recommended${NC}"
fi

# Show pod status
echo ""
echo -e "${BLUE}=== Pod Status ===${NC}"
kubectl get pods -n "$NAMESPACE" -l "app=agentauth,version=${TARGET_VERSION}"

echo ""
echo -e "${GREEN}✅ Traffic switch complete!${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Monitor logs: kubectl logs -n $NAMESPACE -l app=agentauth,version=${TARGET_VERSION} --tail=100 -f"
echo "2. Check metrics: curl -k https://${INGRESS_IP}/metrics | grep agentauth_http_requests_total"
echo "3. If issues arise, rollback: $0 ${CURRENT_VERSION}"
echo ""
echo -e "${BLUE}=== Rollback Command ===${NC}"
echo -e "${YELLOW}$0 ${CURRENT_VERSION}${NC}"
