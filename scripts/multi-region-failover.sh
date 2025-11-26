#!/bin/bash
# Multi-Region Failover Automation Script
# Handles automatic failover between regions

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/var/log/gauth/failover.log"
SLACK_WEBHOOK="${SLACK_WEBHOOK_URL}"
PAGERDUTY_KEY="${PAGERDUTY_INTEGRATION_KEY}"

# Regions
PRIMARY_REGION="${PRIMARY_REGION:-us-east-1}"
FAILOVER_REGIONS=("eu-west-1" "ap-south-1" "us-west-2")

# Health check endpoints
declare -A REGION_ENDPOINTS=(
  ["us-east-1"]="https://gauth.us-east-1.example.com"
  ["eu-west-1"]="https://gauth.eu-west-1.example.com"
  ["ap-south-1"]="https://gauth.ap-south-1.example.com"
  ["us-west-2"]="https://gauth.us-west-2.example.com"
)

# Logging function
log() {
  local level=$1
  shift
  echo "[$(date +'%Y-%m-%d %H:%M:%S')] [$level] $*" | tee -a "$LOG_FILE"
}

# Send Slack notification
notify_slack() {
  local message=$1
  local color=${2:-warning}
  
  if [ -n "$SLACK_WEBHOOK" ]; then
    curl -X POST "$SLACK_WEBHOOK" \
      -H 'Content-Type: application/json' \
      -d "{\"text\": \"$message\", \"color\": \"$color\"}" \
      2>/dev/null || log "ERROR" "Failed to send Slack notification"
  fi
}

# Trigger PagerDuty alert
trigger_pagerduty() {
  local description=$1
  local severity=${2:-critical}
  
  if [ -n "$PAGERDUTY_KEY" ]; then
    curl -X POST 'https://events.pagerduty.com/v2/enqueue' \
      -H 'Content-Type: application/json' \
      -d "{
        \"routing_key\": \"$PAGERDUTY_KEY\",
        \"event_action\": \"trigger\",
        \"payload\": {
          \"summary\": \"$description\",
          \"severity\": \"$severity\",
          \"source\": \"gauth-failover-automation\",
          \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"
        }
      }" 2>/dev/null || log "ERROR" "Failed to trigger PagerDuty alert"
  fi
}

# Check region health
check_region_health() {
  local region=$1
  local endpoint="${REGION_ENDPOINTS[$region]}/api/v1/health"
  local max_retries=3
  local retry_count=0
  
  log "INFO" "Checking health of region: $region"
  
  while [ $retry_count -lt $max_retries ]; do
    if response=$(curl -s -f -m 10 "$endpoint" 2>&1); then
      if echo "$response" | grep -q '"status":"healthy"'; then
        log "INFO" "Region $region is healthy"
        return 0
      fi
    fi
    
    retry_count=$((retry_count + 1))
    log "WARN" "Health check failed for $region (attempt $retry_count/$max_retries)"
    sleep 5
  done
  
  log "ERROR" "Region $region is unhealthy after $max_retries attempts"
  return 1
}

# Check database health
check_database_health() {
  local region=$1
  local endpoint="${REGION_ENDPOINTS[$region]}/api/v1/health/database"
  
  log "INFO" "Checking database health in region: $region"
  
  if response=$(curl -s -f -m 10 "$endpoint" 2>&1); then
    if echo "$response" | grep -q '"database":"connected"'; then
      log "INFO" "Database in $region is healthy"
      return 0
    fi
  fi
  
  log "ERROR" "Database in $region is unhealthy"
  return 1
}

# Promote database to primary
promote_database() {
  local region=$1
  
  log "INFO" "Promoting database in $region to primary"
  
  # Trigger Patroni failover via API
  kubectl exec -n gauth postgresql-0 -- \
    curl -X POST http://localhost:8008/failover \
    -H "Content-Type: application/json" \
    -d "{\"leader\": \"postgresql-0\", \"candidate\": \"postgresql-1\"}"
  
  if [ $? -eq 0 ]; then
    log "INFO" "Database promoted successfully in $region"
    return 0
  else
    log "ERROR" "Failed to promote database in $region"
    return 1
  fi
}

# Update DNS records
update_dns() {
  local failover_region=$1
  local dns_zone_id="${ROUTE53_ZONE_ID}"
  
  log "INFO" "Updating DNS to point to $failover_region"
  
  # Create Route53 change batch
  cat > /tmp/dns-change.json <<EOF
{
  "Comment": "Failover to $failover_region",
  "Changes": [
    {
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "gauth.example.com",
        "Type": "A",
        "TTL": 60,
        "ResourceRecords": [
          {
            "Value": "$(kubectl get svc gauth -n gauth -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"
          }
        ]
      }
    }
  ]
}
EOF
  
  aws route53 change-resource-record-sets \
    --hosted-zone-id "$dns_zone_id" \
    --change-batch file:///tmp/dns-change.json
  
  if [ $? -eq 0 ]; then
    log "INFO" "DNS updated successfully"
    rm /tmp/dns-change.json
    return 0
  else
    log "ERROR" "Failed to update DNS"
    return 1
  fi
}

# Scale up application in failover region
scale_application() {
  local region=$1
  local replicas=${2:-10}
  
  log "INFO" "Scaling application in $region to $replicas replicas"
  
  kubectl scale deployment gauth -n gauth --replicas="$replicas"
  
  # Wait for pods to be ready
  kubectl wait --for=condition=ready pod \
    -l app=gauth -n gauth \
    --timeout=300s
  
  if [ $? -eq 0 ]; then
    log "INFO" "Application scaled successfully in $region"
    return 0
  else
    log "ERROR" "Failed to scale application in $region"
    return 1
  fi
}

# Perform automatic failover
perform_failover() {
  local failed_region=$1
  local target_region=""
  
  log "INFO" "Starting failover from $failed_region"
  notify_slack "🚨 FAILOVER: Initiating failover from $failed_region" "danger"
  trigger_pagerduty "Region $failed_region has failed - failover in progress" "critical"
  
  # Find healthy failover region
  for region in "${FAILOVER_REGIONS[@]}"; do
    if check_region_health "$region"; then
      target_region=$region
      break
    fi
  done
  
  if [ -z "$target_region" ]; then
    log "ERROR" "No healthy failover region found!"
    notify_slack "❌ CRITICAL: All failover regions are unhealthy!" "danger"
    trigger_pagerduty "ALL REGIONS DOWN - Manual intervention required" "critical"
    return 1
  fi
  
  log "INFO" "Selected failover region: $target_region"
  
  # Step 1: Promote database in target region
  if ! promote_database "$target_region"; then
    log "ERROR" "Failed to promote database in $target_region"
    return 1
  fi
  
  # Step 2: Scale up application
  if ! scale_application "$target_region" 10; then
    log "ERROR" "Failed to scale application in $target_region"
    return 1
  fi
  
  # Step 3: Update DNS
  if ! update_dns "$target_region"; then
    log "ERROR" "Failed to update DNS"
    return 1
  fi
  
  # Step 4: Verify failover
  sleep 30  # Wait for DNS propagation
  
  if check_region_health "$target_region" && check_database_health "$target_region"; then
    log "INFO" "Failover completed successfully to $target_region"
    notify_slack "✅ SUCCESS: Failover to $target_region completed successfully" "good"
    return 0
  else
    log "ERROR" "Failover verification failed"
    notify_slack "⚠️ WARNING: Failover completed but verification failed" "warning"
    return 1
  fi
}

# Continuous health monitoring
monitor_health() {
  local check_interval=${1:-30}  # seconds
  
  log "INFO" "Starting continuous health monitoring (interval: ${check_interval}s)"
  
  while true; do
    if ! check_region_health "$PRIMARY_REGION"; then
      log "ERROR" "Primary region $PRIMARY_REGION is unhealthy"
      
      # Perform automatic failover
      if perform_failover "$PRIMARY_REGION"; then
        log "INFO" "Failover completed successfully"
        # Update primary region
        PRIMARY_REGION=$(get_active_region)
      else
        log "ERROR" "Failover failed - manual intervention required"
        notify_slack "🚨 CRITICAL: Automatic failover failed - manual intervention required" "danger"
        trigger_pagerduty "Automatic failover failed" "critical"
      fi
    else
      log "DEBUG" "Primary region $PRIMARY_REGION is healthy"
    fi
    
    sleep "$check_interval"
  done
}

# Get currently active region
get_active_region() {
  # Query DNS to determine active region
  local active_ip=$(dig +short gauth.example.com @8.8.8.8 | head -1)
  
  for region in "${!REGION_ENDPOINTS[@]}"; do
    local region_ip=$(kubectl get svc gauth -n gauth \
      --context "$region" \
      -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
    
    if [ "$active_ip" = "$region_ip" ]; then
      echo "$region"
      return 0
    fi
  done
  
  echo "$PRIMARY_REGION"
}

# Test failover (dry run)
test_failover() {
  local target_region=${1:-eu-west-1}
  
  log "INFO" "Testing failover to $target_region (dry run)"
  
  echo "1. Checking health of target region..."
  if check_region_health "$target_region"; then
    echo "   ✓ Target region is healthy"
  else
    echo "   ✗ Target region is unhealthy"
    return 1
  fi
  
  echo "2. Checking database health..."
  if check_database_health "$target_region"; then
    echo "   ✓ Database is healthy"
  else
    echo "   ✗ Database is unhealthy"
    return 1
  fi
  
  echo "3. Simulating database promotion (dry run)..."
  echo "   ✓ Would promote database in $target_region"
  
  echo "4. Simulating application scaling (dry run)..."
  echo "   ✓ Would scale to 10 replicas in $target_region"
  
  echo "5. Simulating DNS update (dry run)..."
  echo "   ✓ Would update DNS to point to $target_region"
  
  echo ""
  echo "Failover test completed successfully!"
  return 0
}

# Rollback failover
rollback_failover() {
  local original_region=$1
  
  log "INFO" "Rolling back to original region: $original_region"
  notify_slack "🔄 Rolling back to $original_region" "warning"
  
  # Check if original region is healthy
  if ! check_region_health "$original_region"; then
    log "ERROR" "Original region $original_region is still unhealthy"
    return 1
  fi
  
  # Perform failover back to original region
  if perform_failover "$(get_active_region)"; then
    log "INFO" "Rollback completed successfully"
    notify_slack "✅ Rollback to $original_region completed" "good"
    return 0
  else
    log "ERROR" "Rollback failed"
    return 1
  fi
}

# Main command dispatcher
main() {
  case "${1:-}" in
    monitor)
      monitor_health "${2:-30}"
      ;;
    failover)
      perform_failover "${2:-$PRIMARY_REGION}"
      ;;
    test)
      test_failover "${2:-eu-west-1}"
      ;;
    rollback)
      rollback_failover "${2:-$PRIMARY_REGION}"
      ;;
    check)
      check_region_health "${2:-$PRIMARY_REGION}"
      ;;
    *)
      echo "Usage: $0 {monitor|failover|test|rollback|check} [region]"
      echo ""
      echo "Commands:"
      echo "  monitor [interval]   - Continuous health monitoring (default: 30s)"
      echo "  failover [region]    - Perform failover from specified region"
      echo "  test [target]        - Test failover to target region (dry run)"
      echo "  rollback [region]    - Rollback to original region"
      echo "  check [region]       - Check health of specific region"
      echo ""
      echo "Examples:"
      echo "  $0 monitor 60        - Monitor health every 60 seconds"
      echo "  $0 failover us-east-1 - Failover from us-east-1"
      echo "  $0 test eu-west-1    - Test failover to eu-west-1"
      echo "  $0 check ap-south-1  - Check health of ap-south-1"
      exit 1
      ;;
  esac
}

# Run main function
main "$@"
