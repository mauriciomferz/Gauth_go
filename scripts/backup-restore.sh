#!/bin/bash

# GAuth Backup and Disaster Recovery Script
# This script handles automated backups and disaster recovery procedures

set -euo pipefail

# Configuration
NAMESPACE="${NAMESPACE:-gauth-production}"
BACKUP_BUCKET="${BACKUP_BUCKET:-gauth-backups}"
BACKUP_RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
POSTGRES_SERVICE="${POSTGRES_SERVICE:-postgres-service}"
REDIS_SERVICE="${REDIS_SERVICE:-redis-service}"
VAULT_SERVICE="${VAULT_SERVICE:-vault-service}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR:${NC} $1" >&2
}

warning() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING:${NC} $1"
}

# Function to check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    # Check if namespace exists
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        error "Namespace $NAMESPACE does not exist"
        exit 1
    fi
    
    # Check if required tools are available
    for tool in pg_dump redis-cli aws; do
        if ! command -v "$tool" &> /dev/null; then
            warning "$tool is not installed - some backup features may not work"
        fi
    done
    
    log "Prerequisites check completed"
}

# Function to backup PostgreSQL database
backup_postgres() {
    log "Starting PostgreSQL backup..."
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="postgres_backup_${timestamp}.sql"
    local pod_name=$(kubectl get pods -n "$NAMESPACE" -l app=postgres -o jsonpath='{.items[0].metadata.name}')
    
    if [[ -z "$pod_name" ]]; then
        error "No PostgreSQL pod found in namespace $NAMESPACE"
        return 1
    fi
    
    # Create backup directory if it doesn't exist
    mkdir -p backups/postgres
    
    # Perform database backup
    kubectl exec -n "$NAMESPACE" "$pod_name" -- pg_dump -U postgres -d gauth --clean --if-exists > "backups/postgres/$backup_file"
    
    # Compress backup
    gzip "backups/postgres/$backup_file"
    
    # Upload to cloud storage if configured
    if command -v aws &> /dev/null && [[ -n "$BACKUP_BUCKET" ]]; then
        aws s3 cp "backups/postgres/${backup_file}.gz" "s3://$BACKUP_BUCKET/postgres/${backup_file}.gz"
        log "PostgreSQL backup uploaded to S3: ${backup_file}.gz"
    fi
    
    log "PostgreSQL backup completed: ${backup_file}.gz"
}

# Function to backup Redis data
backup_redis() {
    log "Starting Redis backup..."
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="redis_backup_${timestamp}.rdb"
    local pod_name=$(kubectl get pods -n "$NAMESPACE" -l app=redis -o jsonpath='{.items[0].metadata.name}')
    
    if [[ -z "$pod_name" ]]; then
        error "No Redis pod found in namespace $NAMESPACE"
        return 1
    fi
    
    # Create backup directory if it doesn't exist
    mkdir -p backups/redis
    
    # Trigger Redis save and copy the RDB file
    kubectl exec -n "$NAMESPACE" "$pod_name" -- redis-cli -a "$REDIS_PASSWORD" BGSAVE
    sleep 5  # Wait for background save to complete
    kubectl cp "$NAMESPACE/$pod_name:/data/dump.rdb" "backups/redis/$backup_file"
    
    # Compress backup
    gzip "backups/redis/$backup_file"
    
    # Upload to cloud storage if configured
    if command -v aws &> /dev/null && [[ -n "$BACKUP_BUCKET" ]]; then
        aws s3 cp "backups/redis/${backup_file}.gz" "s3://$BACKUP_BUCKET/redis/${backup_file}.gz"
        log "Redis backup uploaded to S3: ${backup_file}.gz"
    fi
    
    log "Redis backup completed: ${backup_file}.gz"
}

# Function to backup Vault data
backup_vault() {
    log "Starting Vault backup..."
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="vault_backup_${timestamp}.json"
    local pod_name=$(kubectl get pods -n "$NAMESPACE" -l app=vault -o jsonpath='{.items[0].metadata.name}')
    
    if [[ -z "$pod_name" ]]; then
        warning "No Vault pod found in namespace $NAMESPACE - skipping Vault backup"
        return 0
    fi
    
    # Create backup directory if it doesn't exist
    mkdir -p backups/vault
    
    # Export Vault secrets (this is a simplified approach)
    # In production, use proper Vault backup procedures
    kubectl exec -n "$NAMESPACE" "$pod_name" -- vault kv export -format=json secret/ > "backups/vault/$backup_file" || true
    
    # Compress backup
    gzip "backups/vault/$backup_file"
    
    # Upload to cloud storage if configured
    if command -v aws &> /dev/null && [[ -n "$BACKUP_BUCKET" ]]; then
        aws s3 cp "backups/vault/${backup_file}.gz" "s3://$BACKUP_BUCKET/vault/${backup_file}.gz"
        log "Vault backup uploaded to S3: ${backup_file}.gz"
    fi
    
    log "Vault backup completed: ${backup_file}.gz"
}

# Function to backup Kubernetes manifests
backup_k8s_manifests() {
    log "Starting Kubernetes manifests backup..."
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_dir="backups/k8s_manifests_${timestamp}"
    
    mkdir -p "$backup_dir"
    
    # Export all resources in the namespace
    for resource in deployment service configmap secret pvc statefulset ingress hpa pdb; do
        kubectl get "$resource" -n "$NAMESPACE" -o yaml > "$backup_dir/${resource}.yaml" 2>/dev/null || true
    done
    
    # Create tarball
    tar -czf "${backup_dir}.tar.gz" -C backups "$(basename "$backup_dir")"
    rm -rf "$backup_dir"
    
    # Upload to cloud storage if configured
    if command -v aws &> /dev/null && [[ -n "$BACKUP_BUCKET" ]]; then
        aws s3 cp "${backup_dir}.tar.gz" "s3://$BACKUP_BUCKET/manifests/$(basename "${backup_dir}.tar.gz")"
        log "Kubernetes manifests backup uploaded to S3: $(basename "${backup_dir}.tar.gz")"
    fi
    
    log "Kubernetes manifests backup completed: $(basename "${backup_dir}.tar.gz")"
}

# Function to clean old backups
cleanup_old_backups() {
    log "Cleaning up old backups (older than $BACKUP_RETENTION_DAYS days)..."
    
    # Clean local backups
    find backups/ -name "*.gz" -mtime +$BACKUP_RETENTION_DAYS -delete 2>/dev/null || true
    find backups/ -name "*.tar.gz" -mtime +$BACKUP_RETENTION_DAYS -delete 2>/dev/null || true
    
    # Clean S3 backups if configured
    if command -v aws &> /dev/null && [[ -n "$BACKUP_BUCKET" ]]; then
        local cutoff_date=$(date -d "$BACKUP_RETENTION_DAYS days ago" +%Y-%m-%d)
        aws s3 ls "s3://$BACKUP_BUCKET/" --recursive | while read -r line; do
            local file_date=$(echo "$line" | awk '{print $1}')
            local file_path=$(echo "$line" | awk '{print $4}')
            if [[ "$file_date" < "$cutoff_date" ]]; then
                aws s3 rm "s3://$BACKUP_BUCKET/$file_path"
                log "Removed old backup: $file_path"
            fi
        done
    fi
    
    log "Cleanup completed"
}

# Function to restore from backup
restore_postgres() {
    local backup_file="$1"
    
    if [[ -z "$backup_file" ]]; then
        error "Please specify a backup file to restore"
        return 1
    fi
    
    log "Starting PostgreSQL restore from $backup_file..."
    
    local pod_name=$(kubectl get pods -n "$NAMESPACE" -l app=postgres -o jsonpath='{.items[0].metadata.name}')
    
    if [[ -z "$pod_name" ]]; then
        error "No PostgreSQL pod found in namespace $NAMESPACE"
        return 1
    fi
    
    # Copy backup file to pod and restore
    kubectl cp "$backup_file" "$NAMESPACE/$pod_name:/tmp/restore.sql"
    kubectl exec -n "$NAMESPACE" "$pod_name" -- psql -U postgres -d gauth -f /tmp/restore.sql
    kubectl exec -n "$NAMESPACE" "$pod_name" -- rm /tmp/restore.sql
    
    log "PostgreSQL restore completed"
}

# Function to test disaster recovery
test_disaster_recovery() {
    log "Starting disaster recovery test..."
    
    # Check if all critical services are running
    local services=("postgres" "redis" "gauth")
    for service in "${services[@]}"; do
        local pod_count=$(kubectl get pods -n "$NAMESPACE" -l app="$service" --field-selector=status.phase=Running -o name | wc -l)
        if [[ $pod_count -eq 0 ]]; then
            error "Service $service is not running"
        else
            log "Service $service is healthy ($pod_count pods running)"
        fi
    done
    
    # Test database connectivity
    local postgres_pod=$(kubectl get pods -n "$NAMESPACE" -l app=postgres -o jsonpath='{.items[0].metadata.name}')
    if kubectl exec -n "$NAMESPACE" "$postgres_pod" -- pg_isready -U postgres >/dev/null 2>&1; then
        log "PostgreSQL connectivity test passed"
    else
        error "PostgreSQL connectivity test failed"
    fi
    
    # Test Redis connectivity
    local redis_pod=$(kubectl get pods -n "$NAMESPACE" -l app=redis -o jsonpath='{.items[0].metadata.name}')
    if kubectl exec -n "$NAMESPACE" "$redis_pod" -- redis-cli ping >/dev/null 2>&1; then
        log "Redis connectivity test passed"
    else
        error "Redis connectivity test failed"
    fi
    
    log "Disaster recovery test completed"
}

# Main function
main() {
    case "${1:-}" in
        "backup")
            check_prerequisites
            backup_postgres
            backup_redis
            backup_vault
            backup_k8s_manifests
            cleanup_old_backups
            ;;
        "restore-postgres")
            restore_postgres "$2"
            ;;
        "test")
            test_disaster_recovery
            ;;
        "cleanup")
            cleanup_old_backups
            ;;
        *)
            echo "Usage: $0 {backup|restore-postgres <file>|test|cleanup}"
            echo ""
            echo "Commands:"
            echo "  backup               - Perform full backup of all services"
            echo "  restore-postgres     - Restore PostgreSQL from backup file"
            echo "  test                 - Test disaster recovery procedures"
            echo "  cleanup              - Clean up old backup files"
            echo ""
            echo "Environment Variables:"
            echo "  NAMESPACE            - Kubernetes namespace (default: gauth-production)"
            echo "  BACKUP_BUCKET        - S3 bucket for backups"
            echo "  BACKUP_RETENTION_DAYS - Days to retain backups (default: 30)"
            echo "  POSTGRES_SERVICE     - PostgreSQL service name"
            echo "  REDIS_SERVICE        - Redis service name"
            echo "  VAULT_SERVICE        - Vault service name"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"