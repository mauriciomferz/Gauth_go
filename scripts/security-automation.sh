#!/bin/bash
# Security Automation Script for AgentAuth
# Handles certificate rotation, vulnerability scanning, and security monitoring

set -euo pipefail

# Configuration
NAMESPACE="${NAMESPACE:-agentauth}"
VAULT_ADDR="${VAULT_ADDR:-https://vault.vault.svc.cluster.local:8200}"
SLACK_WEBHOOK="${SLACK_WEBHOOK:-}"
LOG_FILE="${LOG_FILE:-/var/log/agentauth/security-automation.log}"
SCAN_RESULTS_DIR="/var/log/agentauth/security-scans"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR] $*${NC}" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS] $*${NC}" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING] $*${NC}" | tee -a "$LOG_FILE"
}

# Slack notification
notify_slack() {
    local message="$1"
    local color="${2:-good}"  # good, warning, danger
    
    if [ -n "$SLACK_WEBHOOK" ]; then
        curl -X POST "$SLACK_WEBHOOK" \
            -H 'Content-Type: application/json' \
            -d "{
                \"attachments\": [{
                    \"color\": \"$color\",
                    \"title\": \"AgentAuth Security Automation\",
                    \"text\": \"$message\",
                    \"footer\": \"Security Team\",
                    \"ts\": $(date +%s)
                }]
            }" || log_error "Failed to send Slack notification"
    fi
}

# Check certificate expiry
check_certificate_expiry() {
    log "Checking certificate expiry dates..."
    
    local expiring_soon=()
    
    # Get all certificates in namespace
    kubectl get certificates -n "$NAMESPACE" -o json | jq -r '.items[] | "\(.metadata.name)|\(.status.notAfter)"' | while IFS='|' read -r cert_name not_after; do
        if [ -z "$not_after" ] || [ "$not_after" == "null" ]; then
            log_warning "Certificate $cert_name has no expiry date"
            continue
        fi
        
        # Calculate days until expiry
        expiry_epoch=$(date -d "$not_after" +%s 2>/dev/null || echo 0)
        current_epoch=$(date +%s)
        days_remaining=$(( ($expiry_epoch - $current_epoch) / 86400 ))
        
        log "Certificate $cert_name expires in $days_remaining days"
        
        if [ "$days_remaining" -lt 7 ]; then
            expiring_soon+=("$cert_name ($days_remaining days)")
        fi
        
        if [ "$days_remaining" -lt 2 ]; then
            log_error "Certificate $cert_name expires in less than 2 days!"
            notify_slack "🚨 Certificate $cert_name expires in $days_remaining days" "danger"
        elif [ "$days_remaining" -lt 7 ]; then
            log_warning "Certificate $cert_name expires soon"
            notify_slack "⚠️ Certificate $cert_name expires in $days_remaining days" "warning"
        fi
    done
    
    if [ ${#expiring_soon[@]} -eq 0 ]; then
        log_success "All certificates have sufficient validity"
    fi
}

# Rotate certificates
rotate_certificates() {
    local cert_name="$1"
    
    log "Starting certificate rotation for $cert_name"
    
    # Annotate certificate to trigger renewal
    kubectl annotate certificate "$cert_name" -n "$NAMESPACE" \
        cert-manager.io/issue-temporary-certificate="true" \
        --overwrite || {
            log_error "Failed to annotate certificate $cert_name"
            return 1
        }
    
    # Wait for renewal
    log "Waiting for certificate renewal..."
    for i in {1..60}; do
        if kubectl get certificate "$cert_name" -n "$NAMESPACE" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' | grep -q "True"; then
            log_success "Certificate $cert_name renewed successfully"
            notify_slack "✅ Certificate $cert_name rotated successfully" "good"
            return 0
        fi
        sleep 5
    done
    
    log_error "Certificate $cert_name renewal timed out"
    notify_slack "🚨 Certificate $cert_name rotation failed" "danger"
    return 1
}

# Vulnerability scanning with Trivy
vulnerability_scan() {
    log "Starting vulnerability scanning..."
    
    mkdir -p "$SCAN_RESULTS_DIR"
    local scan_date=$(date +%Y%m%d-%H%M%S)
    local scan_file="$SCAN_RESULTS_DIR/scan-$scan_date.json"
    
    # Scan all container images in namespace
    kubectl get pods -n "$NAMESPACE" -o jsonpath='{range .items[*]}{.spec.containers[*].image}{"\n"}{end}' | sort -u | while read -r image; do
        log "Scanning image: $image"
        
        trivy image --format json --output "${scan_file%.json}-$(echo "$image" | tr '/:' '-').json" \
            --severity CRITICAL,HIGH \
            "$image" || log_error "Failed to scan $image"
    done
    
    # Aggregate results
    find "$SCAN_RESULTS_DIR" -name "scan-$scan_date-*.json" -exec jq -s 'map(.Results[]? | select(.Vulnerabilities != null) | .Vulnerabilities[])' {} + > "$scan_file"
    
    # Count vulnerabilities by severity
    local critical=$(jq '[.[] | select(.Severity == "CRITICAL")] | length' "$scan_file")
    local high=$(jq '[.[] | select(.Severity == "HIGH")] | length' "$scan_file")
    
    log "Vulnerability scan completed: $critical CRITICAL, $high HIGH"
    
    if [ "$critical" -gt 0 ]; then
        log_error "Found $critical CRITICAL vulnerabilities"
        notify_slack "🚨 Security scan found $critical CRITICAL and $high HIGH vulnerabilities" "danger"
    elif [ "$high" -gt 0 ]; then
        log_warning "Found $high HIGH vulnerabilities"
        notify_slack "⚠️ Security scan found $high HIGH vulnerabilities" "warning"
    else
        log_success "No CRITICAL or HIGH vulnerabilities found"
        notify_slack "✅ Security scan passed with no critical vulnerabilities" "good"
    fi
    
    # Cleanup old scan results (keep last 30 days)
    find "$SCAN_RESULTS_DIR" -name "scan-*.json" -mtime +30 -delete
}

# Check Vault seal status
check_vault_status() {
    log "Checking Vault seal status..."
    
    local vault_status=$(kubectl exec -n vault vault-0 -- vault status -format=json 2>/dev/null || echo '{}')
    local sealed=$(echo "$vault_status" | jq -r '.sealed // true')
    local initialized=$(echo "$vault_status" | jq -r '.initialized // false')
    
    if [ "$sealed" == "true" ]; then
        log_error "Vault is SEALED!"
        notify_slack "🚨 Vault is SEALED - immediate attention required" "danger"
        return 1
    fi
    
    if [ "$initialized" == "false" ]; then
        log_error "Vault is NOT initialized"
        notify_slack "🚨 Vault is not initialized" "danger"
        return 1
    fi
    
    log_success "Vault is unsealed and operational"
    return 0
}

# Rotate Vault tokens
rotate_vault_tokens() {
    log "Rotating Vault tokens..."
    
    # Get all Vault tokens from Kubernetes secrets
    kubectl get secrets -n "$NAMESPACE" -l vault-token=true -o json | jq -r '.items[].metadata.name' | while read -r secret_name; do
        log "Rotating token in secret: $secret_name"
        
        # Delete secret to trigger token regeneration by Vault Agent
        kubectl delete secret "$secret_name" -n "$NAMESPACE" || log_error "Failed to delete secret $secret_name"
        
        # Wait for secret to be recreated
        sleep 10
        
        if kubectl get secret "$secret_name" -n "$NAMESPACE" &>/dev/null; then
            log_success "Token in $secret_name rotated successfully"
        else
            log_error "Failed to rotate token in $secret_name"
        fi
    done
}

# Audit log analysis
analyze_audit_logs() {
    log "Analyzing security audit logs..."
    
    # Failed authentication attempts
    local failed_auth=$(kubectl logs -n "$NAMESPACE" -l app=agentauth-api --tail=10000 | grep -c "authentication failed" || echo 0)
    
    if [ "$failed_auth" -gt 100 ]; then
        log_error "High number of failed authentication attempts: $failed_auth"
        notify_slack "🚨 Detected $failed_auth failed authentication attempts" "danger"
    fi
    
    # Failed mTLS verifications
    local failed_mtls=$(kubectl logs -n "$NAMESPACE" -l app=nginx-mtls --tail=10000 | grep -c "ssl_client_verify\":\"FAILED" || echo 0)
    
    if [ "$failed_mtls" -gt 50 ]; then
        log_error "High number of failed mTLS verifications: $failed_mtls"
        notify_slack "🚨 Detected $failed_mtls failed mTLS verifications" "danger"
    fi
    
    # Suspicious API activity (high 4xx rates)
    local high_4xx=$(kubectl logs -n "$NAMESPACE" -l app=agentauth-api --tail=10000 | grep -c "\"status\":4" || echo 0)
    
    if [ "$high_4xx" -gt 500 ]; then
        log_warning "High rate of 4xx responses: $high_4xx"
        notify_slack "⚠️ Detected $high_4xx 4xx responses - possible attack" "warning"
    fi
    
    log_success "Audit log analysis completed"
}

# Network policy validation
validate_network_policies() {
    log "Validating network policies..."
    
    # Check if network policies exist
    local policy_count=$(kubectl get networkpolicies -n "$NAMESPACE" --no-headers | wc -l)
    
    if [ "$policy_count" -eq 0 ]; then
        log_error "No network policies found in namespace $NAMESPACE"
        notify_slack "🚨 No network policies configured" "danger"
        return 1
    fi
    
    log "Found $policy_count network policies"
    
    # Validate critical policies
    for policy in "agentauth-api-network-policy" "postgresql-network-policy" "redis-network-policy"; do
        if kubectl get networkpolicy "$policy" -n "$NAMESPACE" &>/dev/null; then
            log_success "Network policy $policy is configured"
        else
            log_error "Network policy $policy is missing"
            notify_slack "🚨 Critical network policy $policy is missing" "danger"
        fi
    done
}

# HSM connectivity check
check_hsm_connectivity() {
    log "Checking HSM connectivity..."
    
    # Test CloudHSM connectivity from Vault
    if kubectl exec -n vault vault-0 -- vault status &>/dev/null; then
        log_success "HSM connectivity is healthy"
        return 0
    else
        log_error "HSM connectivity check failed"
        notify_slack "🚨 HSM connectivity issue detected" "danger"
        return 1
    fi
}

# Security compliance check
compliance_check() {
    log "Running security compliance checks..."
    
    local issues=0
    
    # Check 1: TLS version enforcement
    if kubectl get configmap nginx-mtls-config -n "$NAMESPACE" -o yaml | grep -q "ssl_protocols TLSv1.3"; then
        log_success "TLS 1.3 enforcement verified"
    else
        log_error "TLS 1.3 not enforced"
        ((issues++))
    fi
    
    # Check 2: mTLS requirement
    if kubectl get peerauthentication default-mtls-strict -n "$NAMESPACE" -o yaml | grep -q "mode: STRICT"; then
        log_success "mTLS STRICT mode verified"
    else
        log_error "mTLS STRICT mode not configured"
        ((issues++))
    fi
    
    # Check 3: Vault seal type
    if kubectl get configmap vault-config -n vault -o yaml | grep -q "seal \"awskms\""; then
        log_success "Vault auto-unseal with HSM verified"
    else
        log_warning "Vault not using HSM auto-unseal"
    fi
    
    # Check 4: Pod security policies
    if kubectl get psp &>/dev/null; then
        log_success "Pod Security Policies enabled"
    else
        log_warning "Pod Security Policies not found (may be using PSA)"
    fi
    
    if [ "$issues" -eq 0 ]; then
        log_success "All compliance checks passed"
    else
        log_error "$issues compliance issues found"
        notify_slack "⚠️ $issues security compliance issues detected" "warning"
    fi
}

# Generate security report
generate_security_report() {
    log "Generating security report..."
    
    local report_file="/var/log/agentauth/security-report-$(date +%Y%m%d).md"
    
    cat > "$report_file" <<EOF
# AgentAuth Security Report
**Date**: $(date +"%Y-%m-%d %H:%M:%S")
**Namespace**: $NAMESPACE

## Certificate Status
\`\`\`
$(kubectl get certificates -n "$NAMESPACE" -o wide)
\`\`\`

## Vault Status
\`\`\`
$(kubectl exec -n vault vault-0 -- vault status 2>/dev/null || echo "Vault unavailable")
\`\`\`

## Network Policies
\`\`\`
$(kubectl get networkpolicies -n "$NAMESPACE" -o wide)
\`\`\`

## Recent Security Events
\`\`\`
$(tail -n 50 "$LOG_FILE")
\`\`\`

## Latest Vulnerability Scan
\`\`\`
$(find "$SCAN_RESULTS_DIR" -name "scan-*.json" -type f -printf '%T@ %p\n' | sort -nr | head -1 | cut -d' ' -f2- | xargs cat | jq -r '[.[] | select(.Severity == "CRITICAL" or .Severity == "HIGH")] | unique_by(.VulnerabilityID) | .[] | "\(.Severity): \(.VulnerabilityID) - \(.Title)"' | head -20)
\`\`\`

---
*Generated by AgentAuth Security Automation*
EOF
    
    log_success "Security report generated: $report_file"
}

# Main execution
main() {
    local action="${1:-all}"
    
    log "=== AgentAuth Security Automation Started ==="
    log "Action: $action"
    
    case "$action" in
        certs)
            check_certificate_expiry
            ;;
        rotate)
            local cert_name="${2:-}"
            if [ -z "$cert_name" ]; then
                log_error "Certificate name required for rotation"
                exit 1
            fi
            rotate_certificates "$cert_name"
            ;;
        scan)
            vulnerability_scan
            ;;
        vault)
            check_vault_status
            ;;
        audit)
            analyze_audit_logs
            ;;
        network)
            validate_network_policies
            ;;
        hsm)
            check_hsm_connectivity
            ;;
        compliance)
            compliance_check
            ;;
        report)
            generate_security_report
            ;;
        all)
            check_certificate_expiry
            check_vault_status
            validate_network_policies
            check_hsm_connectivity
            analyze_audit_logs
            compliance_check
            vulnerability_scan
            generate_security_report
            ;;
        *)
            echo "Usage: $0 {certs|rotate <cert-name>|scan|vault|audit|network|hsm|compliance|report|all}"
            exit 1
            ;;
    esac
    
    log "=== AgentAuth Security Automation Completed ==="
}

# Ensure log directory exists
mkdir -p "$(dirname "$LOG_FILE")"
mkdir -p "$SCAN_RESULTS_DIR"

# Run main function
main "$@"
