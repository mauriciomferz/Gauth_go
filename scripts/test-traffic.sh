#!/bin/bash
# Generate test traffic for GAuth+ monitoring dashboard

BASE_URL="http://localhost:8080"

echo "=== Generating GAuth+ Test Traffic ==="
echo ""

# 1. Create capability assessments (varying levels)
echo "📊 Creating capability assessments..."
for i in {1..10}; do
  level=$((i % 4 + 1))
  curl -s -X POST "$BASE_URL/api/v1/gauthplus/capabilities" \
    -H "Content-Type: application/json" \
    -d "{
      \"agent_id\": \"agent-$(printf '%03d' $i)\",
      \"overall_level\": \"L$level\",
      \"assessment_timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
      \"assessor_id\": \"assessor-001\"
    }" > /dev/null
  echo "  ✓ Agent-$(printf '%03d' $i): Level L$level"
done

echo ""

# 2. Create delegation chains
echo "🔗 Creating delegation chains..."
for i in {1..5}; do
  next=$((i + 1))
  curl -s -X POST "$BASE_URL/api/v1/gauthplus/delegations" \
    -H "Content-Type: application/json" \
    -d "{
      \"delegator_agent_id\": \"agent-$(printf '%03d' $i)\",
      \"delegate_agent_id\": \"agent-$(printf '%03d' $next)\",
      \"delegated_capability\": \"read\",
      \"delegation_timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
      \"expiry_timestamp\": \"$(date -u -v+7d +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '+7 days' +%Y-%m-%dT%H:%M:%SZ)\"
    }" > /dev/null
  echo "  ✓ Delegation: agent-$(printf '%03d' $i) → agent-$(printf '%03d' $next)"
done

echo ""

# 3. Create dual control approvals
echo "👥 Creating dual control approvals..."
for i in {1..3}; do
  curl -s -X POST "$BASE_URL/api/v1/gauthplus/dual-control" \
    -H "Content-Type: application/json" \
    -d "{
      \"action_type\": \"high_value_transfer\",
      \"primary_agent_id\": \"agent-$(printf '%03d' $i)\",
      \"approver_agent_id\": \"agent-$(printf '%03d' $((i+5)))\",
      \"poa_id\": \"550e8400-e29b-41d4-a716-44665544000$i\",
      \"request_timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"
    }" > /dev/null
  echo "  ✓ Dual control: agent-$(printf '%03d' $i) + agent-$(printf '%03d' $((i+5)))"
done

echo ""

# 4. Record fiduciary violations
echo "⚠️  Recording fiduciary violations..."
severities=("low" "medium" "high")
duties=("transparency" "loyalty" "prudence" "accountability")

for i in {1..5}; do
  sev=${severities[$((i % 3))]}
  duty=${duties[$((i % 4))]}
  curl -s -X POST "$BASE_URL/api/v1/gauthplus/fiduciary" \
    -H "Content-Type: application/json" \
    -d "{
      \"poa_id\": \"550e8400-e29b-41d4-a716-44665544000$i\",
      \"agent_id\": \"agent-$(printf '%03d' $i)\",
      \"duty_type\": \"$duty\",
      \"violation_timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
      \"severity\": \"$sev\"
    }" > /dev/null
  echo "  ✓ Violation: $duty ($sev) - agent-$(printf '%03d' $i)"
done

echo ""

# 5. Activate successors
echo "🔄 Activating successors..."
for i in {1..2}; do
  curl -s -X POST "$BASE_URL/api/v1/gauthplus/successors/activate" \
    -H "Content-Type: application/json" \
    -d "{
      \"poa_id\": \"550e8400-e29b-41d4-a716-44665544000$i\",
      \"successor_agent_id\": \"successor-$(printf '%03d' $i)\",
      \"trigger_event\": \"primary_unavailable\",
      \"activation_timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"
    }" > /dev/null
  echo "  ✓ Successor activated: successor-$(printf '%03d' $i)"
done

echo ""

# 6. Test capability API multiple times to trigger cache
echo "💾 Testing cache performance..."
for i in {1..20}; do
  curl -s "$BASE_URL/api/v1/gauthplus/capabilities/agent-001/latest" > /dev/null
  [ $((i % 5)) -eq 0 ] && echo "  ✓ $i requests completed"
done

echo ""
echo "=== Test Traffic Generation Complete ==="
echo ""
echo "📊 View metrics at:"
echo "   Grafana:    http://localhost:3000/d/gauthplus-monitoring/gauthplus-monitoring"
echo "   Prometheus: http://localhost:9090/graph"
echo "   Metrics:    http://localhost:8080/metrics"
echo ""
