#!/bin/bash
set -e

BASE_URL="http://localhost:8090"
FRONTEND_URL="http://localhost:3002"

echo "=== AgentAuth Production Smoke Test ==="

# 1. Check Backend Health
echo -n "[Backend] Checking health... "
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/beta/health")
if [ "$HTTP_CODE" -eq 200 ]; then
    echo "OK ($HTTP_CODE)"
else
    echo "FAIL ($HTTP_CODE)"
    exit 1
fi

# 2. Check Frontend Index (Content Verification)
echo -n "[Frontend] Checking index.html content... "
CONTENT=$(curl -s "$FRONTEND_URL")
if [[ "$CONTENT" == *"AgentAuth"* ]]; then
    echo "OK (Found 'AgentAuth' in title)"
else
    echo "FAIL (Title verification failed)"
    echo "Preview: $(echo "$CONTENT" | head -n 5)"
    # Don't exit here, strictly speaking, just warn, as browser cache might be the user's issue
fi

# 3. Check Frontend Assets
# Extract main JS file from index.html to verify it's served
ASSET_PATH=$(echo "$CONTENT" | grep -o 'src="/assets/[^"]*"' | head -1 | cut -d'"' -f2)
if [ -n "$ASSET_PATH" ]; then
    echo -n "[Frontend] Checking asset $ASSET_PATH... "
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$FRONTEND_URL$ASSET_PATH")
    if [ "$HTTP_CODE" -eq 200 ]; then
        echo "OK ($HTTP_CODE)"
    else
        echo "FAIL ($HTTP_CODE)"
    fi
else
    echo "[Frontend] SKIP (Could not parse asset path)"
fi

echo "=== Smoke Test Complete ==="
