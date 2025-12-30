#!/bin/bash
# CloudHSM Setup and Integration Script for AgentAuth
# Provisions AWS CloudHSM cluster and integrates with Vault

set -euo pipefail

# Configuration
AWS_REGION="${AWS_REGION:-us-east-1}"
CLUSTER_NAME="${CLUSTER_NAME:-gauth-hsm-cluster}"
HSM_TYPE="${HSM_TYPE:-hsm1.medium}"
BACKUP_ID="${BACKUP_ID:-}"  # For DR restoration
SLACK_WEBHOOK="${SLACK_WEBHOOK:-}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*"
}

log_error() {
    echo -e "${RED}[ERROR] $*${NC}"
}

log_success() {
    echo -e "${GREEN}[SUCCESS] $*${NC}"
}

notify_slack() {
    local message="$1"
    local color="${2:-good}"
    
    if [ -n "$SLACK_WEBHOOK" ]; then
        curl -X POST "$SLACK_WEBHOOK" \
            -H 'Content-Type: application/json' \
            -d "{
                \"attachments\": [{
                    \"color\": \"$color\",
                    \"title\": \"CloudHSM Setup\",
                    \"text\": \"$message\",
                    \"footer\": \"AgentAuth Security\",
                    \"ts\": $(date +%s)
                }]
            }"
    fi
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check AWS CLI
    if ! command -v aws &> /dev/null; then
        log_error "AWS CLI not found. Install with: pip install awscli"
        exit 1
    fi
    
    # Check AWS credentials
    if ! aws sts get-caller-identity &> /dev/null; then
        log_error "AWS credentials not configured"
        exit 1
    fi
    
    # Check jq
    if ! command -v jq &> /dev/null; then
        log_error "jq not found. Install with: apt-get install jq"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Get VPC subnets
get_vpc_subnets() {
    log "Getting VPC subnets..."
    
    # Get subnets from EKS cluster VPC
    local vpc_id=$(aws eks describe-cluster --name gauth-cluster --region "$AWS_REGION" --query 'cluster.resourcesVpcConfig.vpcId' --output text 2>/dev/null || echo "")
    
    if [ -z "$vpc_id" ]; then
        log_error "Could not find EKS cluster VPC. Please specify manually."
        exit 1
    fi
    
    log "Found VPC: $vpc_id"
    
    # Get private subnets in different AZs
    local subnets=$(aws ec2 describe-subnets \
        --region "$AWS_REGION" \
        --filters "Name=vpc-id,Values=$vpc_id" "Name=tag:kubernetes.io/role/internal-elb,Values=1" \
        --query 'Subnets[0:3].SubnetId' \
        --output json)
    
    echo "$subnets" | jq -r '.[]'
}

# Create CloudHSM cluster
create_hsm_cluster() {
    log "Creating CloudHSM cluster..."
    
    # Get subnets
    local subnet_ids=($(get_vpc_subnets))
    
    if [ ${#subnet_ids[@]} -lt 2 ]; then
        log_error "At least 2 subnets required for HA. Found: ${#subnet_ids[@]}"
        exit 1
    fi
    
    log "Using subnets: ${subnet_ids[*]}"
    
    # Create cluster
    local create_args=(
        --region "$AWS_REGION"
        --hsm-type "$HSM_TYPE"
        --subnet-ids "${subnet_ids[@]}"
    )
    
    if [ -n "$BACKUP_ID" ]; then
        create_args+=(--source-backup-id "$BACKUP_ID")
        log "Restoring from backup: $BACKUP_ID"
    fi
    
    local cluster_id=$(aws cloudhsmv2 create-cluster "${create_args[@]}" --query 'Cluster.ClusterId' --output text)
    
    log_success "CloudHSM cluster created: $cluster_id"
    echo "$cluster_id" > /tmp/hsm-cluster-id.txt
    
    notify_slack "CloudHSM cluster created: $cluster_id" "good"
    
    echo "$cluster_id"
}

# Create HSM instances
create_hsm_instances() {
    local cluster_id="$1"
    local count="${2:-2}"  # Default: 2 HSMs for HA
    
    log "Creating $count HSM instances in cluster $cluster_id..."
    
    # Get availability zones
    local azs=$(aws cloudhsmv2 describe-clusters \
        --region "$AWS_REGION" \
        --filters clusterIds="$cluster_id" \
        --query 'Clusters[0].SubnetMapping' \
        --output json | jq -r 'keys[]' | head -n "$count")
    
    local hsm_ids=()
    
    for az in $azs; do
        log "Creating HSM in AZ: $az"
        
        local hsm_id=$(aws cloudhsmv2 create-hsm \
            --region "$AWS_REGION" \
            --cluster-id "$cluster_id" \
            --availability-zone "$az" \
            --query 'Hsm.HsmId' \
            --output text)
        
        log_success "HSM created: $hsm_id in $az"
        hsm_ids+=("$hsm_id")
        
        # Wait between creations
        sleep 5
    done
    
    log_success "Created ${#hsm_ids[@]} HSM instances: ${hsm_ids[*]}"
    notify_slack "Created ${#hsm_ids[@]} HSM instances" "good"
}

# Wait for cluster initialization
wait_for_cluster() {
    local cluster_id="$1"
    local max_wait=1800  # 30 minutes
    local elapsed=0
    
    log "Waiting for cluster initialization..."
    
    while [ $elapsed -lt $max_wait ]; do
        local state=$(aws cloudhsmv2 describe-clusters \
            --region "$AWS_REGION" \
            --filters clusterIds="$cluster_id" \
            --query 'Clusters[0].State' \
            --output text)
        
        log "Cluster state: $state"
        
        if [ "$state" == "UNINITIALIZED" ]; then
            log_success "Cluster ready for initialization"
            return 0
        elif [ "$state" == "INITIALIZED" ]; then
            log_success "Cluster already initialized"
            return 0
        elif [ "$state" == "CREATE_IN_PROGRESS" ]; then
            log "Cluster creation in progress... ($elapsed/$max_wait seconds)"
        else
            log_error "Unexpected cluster state: $state"
            return 1
        fi
        
        sleep 30
        elapsed=$((elapsed + 30))
    done
    
    log_error "Cluster initialization timed out"
    return 1
}

# Initialize cluster
initialize_cluster() {
    local cluster_id="$1"
    
    log "Initializing CloudHSM cluster..."
    
    # Get cluster certificate
    local cluster_csr=$(aws cloudhsmv2 describe-clusters \
        --region "$AWS_REGION" \
        --filters clusterIds="$cluster_id" \
        --query 'Clusters[0].Certificates.ClusterCsr' \
        --output text)
    
    if [ -z "$cluster_csr" ] || [ "$cluster_csr" == "None" ]; then
        log_error "Cluster CSR not available"
        return 1
    fi
    
    echo "$cluster_csr" > /tmp/cluster-csr.pem
    log "Cluster CSR saved to /tmp/cluster-csr.pem"
    
    # Generate self-signed CA (for demo - use proper CA in production)
    log "Generating customer CA certificate..."
    
    openssl genrsa -out /tmp/customer-ca.key 2048
    openssl req -new -x509 -days 3650 -key /tmp/customer-ca.key \
        -out /tmp/customer-ca.crt \
        -subj "/C=US/ST=State/L=City/O=AgentAuth/CN=AgentAuth CloudHSM CA"
    
    # Sign cluster CSR
    log "Signing cluster certificate..."
    openssl x509 -req -days 3650 \
        -in /tmp/cluster-csr.pem \
        -CA /tmp/customer-ca.crt \
        -CAkey /tmp/customer-ca.key \
        -CAcreateserial \
        -out /tmp/cluster-cert.pem
    
    # Initialize cluster
    log "Uploading certificates to cluster..."
    aws cloudhsmv2 initialize-cluster \
        --region "$AWS_REGION" \
        --cluster-id "$cluster_id" \
        --signed-cert file:///tmp/cluster-cert.pem \
        --trust-anchor file:///tmp/customer-ca.crt
    
    log_success "Cluster initialization started"
    notify_slack "CloudHSM cluster $cluster_id initialized" "good"
    
    # Wait for initialization to complete
    local max_wait=600
    local elapsed=0
    
    while [ $elapsed -lt $max_wait ]; do
        local state=$(aws cloudhsmv2 describe-clusters \
            --region "$AWS_REGION" \
            --filters clusterIds="$cluster_id" \
            --query 'Clusters[0].State' \
            --output text)
        
        if [ "$state" == "INITIALIZED" ]; then
            log_success "Cluster initialization complete"
            return 0
        fi
        
        log "Waiting for initialization... ($elapsed/$max_wait seconds)"
        sleep 30
        elapsed=$((elapsed + 30))
    done
    
    log_error "Initialization timed out"
    return 1
}

# Create KMS key with CloudHSM backing
create_kms_key() {
    local cluster_id="$1"
    
    log "Creating KMS key backed by CloudHSM..."
    
    # Get cluster certificate
    local cluster_cert=$(aws cloudhsmv2 describe-clusters \
        --region "$AWS_REGION" \
        --filters clusterIds="$cluster_id" \
        --query 'Clusters[0].Certificates.ClusterCertificate' \
        --output text)
    
    # Create KMS custom key store
    local key_store_name="gauth-cloudhsm-keystore"
    
    local key_store_id=$(aws kms create-custom-key-store \
        --region "$AWS_REGION" \
        --custom-key-store-name "$key_store_name" \
        --cloud-hsm-cluster-id "$cluster_id" \
        --key-store-password "ChangeMe123!" \
        --trust-anchor-certificate "$cluster_cert" \
        --query 'CustomKeyStoreId' \
        --output text 2>/dev/null || echo "")
    
    if [ -z "$key_store_id" ]; then
        log "Custom key store may already exist, retrieving..."
        key_store_id=$(aws kms describe-custom-key-stores \
            --region "$AWS_REGION" \
            --custom-key-store-name "$key_store_name" \
            --query 'CustomKeyStores[0].CustomKeyStoreId' \
            --output text)
    fi
    
    log "Custom key store ID: $key_store_id"
    
    # Connect key store
    aws kms connect-custom-key-store \
        --region "$AWS_REGION" \
        --custom-key-store-id "$key_store_id" || true
    
    # Create KMS key
    local key_id=$(aws kms create-key \
        --region "$AWS_REGION" \
        --origin AWS_CLOUDHSM \
        --custom-key-store-id "$key_store_id" \
        --description "AgentAuth Vault auto-unseal key (CloudHSM-backed)" \
        --query 'KeyMetadata.KeyId' \
        --output text)
    
    log_success "KMS key created: $key_id"
    
    # Create alias
    aws kms create-alias \
        --region "$AWS_REGION" \
        --alias-name alias/gauth-vault-unseal \
        --target-key-id "$key_id" || true
    
    log_success "KMS key alias created: alias/gauth-vault-unseal"
    notify_slack "KMS key created with CloudHSM backing: $key_id" "good"
    
    echo "$key_id"
}

# Configure Vault with HSM
configure_vault_hsm() {
    local kms_key_id="$1"
    
    log "Configuring Vault to use CloudHSM..."
    
    # Update Vault ConfigMap
    kubectl patch configmap vault-config -n vault --type merge -p "{\"data\":{\"vault.hcl\":\"$(cat <<EOF | sed 's/"/\\"/g'
ui = true

listener "tcp" {
  address       = "[::]:8200"
  tls_cert_file = "/vault/tls/tls.crt"
  tls_key_file  = "/vault/tls/tls.key"
  tls_min_version = "tls13"
}

storage "consul" {
  address = "consul.vault.svc.cluster.local:8500"
  path    = "vault/"
}

seal "awskms" {
  region     = "$AWS_REGION"
  kms_key_id = "$kms_key_id"
}

api_addr = "https://vault.vault.svc.cluster.local:8200"
cluster_addr = "https://vault-active.vault.svc.cluster.local:8201"

telemetry {
  prometheus_retention_time = "30s"
  disable_hostname          = true
}
EOF
)\"}}"
    
    log_success "Vault ConfigMap updated with CloudHSM configuration"
    
    # Restart Vault pods to apply configuration
    kubectl rollout restart statefulset/vault -n vault
    kubectl rollout status statefulset/vault -n vault --timeout=5m
    
    log_success "Vault pods restarted with HSM configuration"
    notify_slack "Vault configured to use CloudHSM for auto-unseal" "good"
}

# Create backup
create_backup() {
    local cluster_id="$1"
    
    log "Creating CloudHSM cluster backup..."
    
    local backup_id=$(aws cloudhsmv2 create-backup \
        --region "$AWS_REGION" \
        --cluster-id "$cluster_id" \
        --query 'Backup.BackupId' \
        --output text)
    
    log_success "Backup created: $backup_id"
    notify_slack "CloudHSM backup created: $backup_id" "good"
    
    echo "$backup_id"
}

# Main setup flow
main() {
    local action="${1:-setup}"
    
    log "=== CloudHSM Setup for AgentAuth ==="
    
    check_prerequisites
    
    case "$action" in
        setup)
            # Full setup flow
            local cluster_id=$(create_hsm_cluster)
            create_hsm_instances "$cluster_id" 2
            wait_for_cluster "$cluster_id"
            initialize_cluster "$cluster_id"
            local kms_key_id=$(create_kms_key "$cluster_id")
            configure_vault_hsm "$kms_key_id"
            create_backup "$cluster_id"
            
            log_success "CloudHSM setup complete!"
            log "Cluster ID: $cluster_id"
            log "KMS Key ID: $kms_key_id"
            ;;
        
        backup)
            local cluster_id="${2:-}"
            if [ -z "$cluster_id" ]; then
                log_error "Cluster ID required for backup"
                exit 1
            fi
            create_backup "$cluster_id"
            ;;
        
        status)
            local cluster_id="${2:-}"
            if [ -z "$cluster_id" ]; then
                log_error "Cluster ID required"
                exit 1
            fi
            
            aws cloudhsmv2 describe-clusters \
                --region "$AWS_REGION" \
                --filters clusterIds="$cluster_id" \
                --output table
            ;;
        
        *)
            echo "Usage: $0 {setup|backup <cluster-id>|status <cluster-id>}"
            exit 1
            ;;
    esac
}

main "$@"
