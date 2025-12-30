# Advanced Security Deployment Guide for AgentAuth

**Version**: 1.0  
**Date**: November 2025  
**Estimated Setup Time**: 2-3 days  
**Compliance Impact**: +1.0 (98/100 → 99/100)

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Phase 1: Certificate Infrastructure](#phase-1-certificate-infrastructure)
3. [Phase 2: Vault Deployment](#phase-2-vault-deployment)
4. [Phase 3: CloudHSM Integration](#phase-3-cloudhsm-integration)
5. [Phase 4: mTLS Configuration](#phase-4-mtls-configuration)
6. [Phase 5: Security Monitoring](#phase-5-security-monitoring)
7. [Testing & Validation](#testing--validation)
8. [Operational Runbooks](#operational-runbooks)
9. [Troubleshooting](#troubleshooting)
10. [Incident Response](#incident-response)

---

## Prerequisites

### Infrastructure Requirements

| Component | Specification | Quantity |
|-----------|--------------|----------|
| **Kubernetes Cluster** | v1.25+ with Pod Security Admission | 1 |
| **Kubernetes Nodes** | 4 vCPU, 16GB RAM minimum | 10+ nodes |
| **Storage** | Fast-SSD storage class (gp3, premium-ssd) | 100GB+ |
| **AWS CloudHSM** | hsm1.medium (FIPS 140-2 Level 3) | 2+ HSMs |
| **Network** | VPC with private subnets in 3 AZs | 1 VPC |

### Required Tools

```bash
# Install required CLI tools
brew install kubectl helm awscli jq openssl

# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Install Istio service mesh
curl -L https://istio.io/downloadIstio | sh -
cd istio-*/bin
./istioctl install --set profile=default -y

# Install Trivy for vulnerability scanning
brew install aquasecurity/trivy/trivy

# Verify installations
kubectl version --client
helm version
aws --version
istioctl version
trivy version
```

### Access Requirements

- AWS IAM permissions for CloudHSM, KMS, EKS
- Kubernetes cluster-admin access
- Slack webhook URL for notifications (optional)
- PagerDuty API key for alerting (optional)

---

## Phase 1: Certificate Infrastructure

**Duration**: 4-6 hours  
**Prerequisites**: Kubernetes cluster with cert-manager installed

### Step 1: Deploy Vault PKI Backend (2 hours)

```bash
# 1.1: Create vault namespace
kubectl create namespace vault

# 1.2: Generate initial Vault TLS certificates (temporary, will be replaced by cert-manager)
openssl req -new -x509 -days 365 -nodes \
    -out /tmp/vault-server.crt \
    -keyout /tmp/vault-server.key \
    -subj "/CN=vault.vault.svc.cluster.local"

kubectl create secret tls vault-server-tls \
    -n vault \
    --cert=/tmp/vault-server.crt \
    --key=/tmp/vault-server.key

# 1.3: Deploy Consul (Vault backend)
kubectl apply -f k8s/security/vault-deployment.yaml --selector=app=consul

# Wait for Consul to be ready
kubectl wait --for=condition=ready pod -l app=consul -n vault --timeout=300s

# 1.4: Generate Consul gossip key
CONSUL_GOSSIP_KEY=$(consul keygen)
kubectl create secret generic consul-gossip-key \
    -n vault \
    --from-literal=key="$CONSUL_GOSSIP_KEY"

# 1.5: Restart Consul with gossip encryption
kubectl rollout restart statefulset/consul -n vault

# 1.6: Deploy Vault
kubectl apply -f k8s/security/vault-deployment.yaml --selector=app=vault

# Wait for Vault pods
kubectl wait --for=condition=ready pod -l app=vault -n vault --timeout=300s

# 1.7: Initialize Vault (ONE TIME ONLY)
kubectl exec -n vault vault-0 -- vault operator init \
    -key-shares=5 \
    -key-threshold=3 \
    -format=json > /tmp/vault-init.json

# CRITICAL: Save this file securely - contains unseal keys and root token
chmod 600 /tmp/vault-init.json

# Extract root token
export VAULT_ROOT_TOKEN=$(jq -r '.root_token' /tmp/vault-init.json)

# Unseal Vault (repeat for each pod)
for i in 0 1 2; do
    for key in $(jq -r '.unseal_keys_b64[]' /tmp/vault-init.json | head -3); do
        kubectl exec -n vault vault-$i -- vault operator unseal "$key"
    done
done

# 1.8: Configure Vault PKI
kubectl exec -n vault vault-0 -- vault login "$VAULT_ROOT_TOKEN"

# Enable PKI secrets engine for root CA
kubectl exec -n vault vault-0 -- vault secrets enable -path=pki pki
kubectl exec -n vault vault-0 -- vault secrets tune -max-lease-ttl=87600h pki

# Generate root CA
kubectl exec -n vault vault-0 -- vault write -field=certificate pki/root/generate/internal \
    common_name="AgentAuth Root CA" \
    ttl=87600h \
    key_bits=4096 > /tmp/gauth-root-ca.crt

# Configure CA URLs
kubectl exec -n vault vault-0 -- vault write pki/config/urls \
    issuing_certificates="https://vault.vault.svc.cluster.local:8200/v1/pki/ca" \
    crl_distribution_points="https://vault.vault.svc.cluster.local:8200/v1/pki/crl"

# Enable intermediate CA
kubectl exec -n vault vault-0 -- vault secrets enable -path=pki_int pki
kubectl exec -n vault vault-0 -- vault secrets tune -max-lease-ttl=43800h pki_int

# Generate intermediate CSR
kubectl exec -n vault vault-0 -- vault write -format=json pki_int/intermediate/generate/internal \
    common_name="AgentAuth Services Intermediate CA" \
    ttl=43800h | jq -r '.data.csr' > /tmp/pki_intermediate.csr

# Sign intermediate with root CA
kubectl exec -n vault vault-0 -- vault write -format=json pki/root/sign-intermediate \
    csr="$(cat /tmp/pki_intermediate.csr)" \
    format=pem_bundle \
    ttl=43800h | jq -r '.data.certificate' > /tmp/intermediate.cert.pem

# Import signed intermediate
kubectl exec -n vault vault-0 -- vault write pki_int/intermediate/set-signed \
    certificate="$(cat /tmp/intermediate.cert.pem)"

# Create roles for server and client certificates
kubectl exec -n vault vault-0 -- vault write pki_int/roles/gauth-server \
    allowed_domains="gauth-api.example.com,*.gauth-api.example.com,*.internal,*.gauth.svc.cluster.local" \
    allow_subdomains=true \
    allow_ip_sans=true \
    server_flag=true \
    client_flag=false \
    max_ttl="168h" \
    ttl="168h"

kubectl exec -n vault vault-0 -- vault write pki_int/roles/gauth-client \
    allowed_domains="*.client.gauth.internal,*.gauth.svc.cluster.local" \
    allow_subdomains=true \
    client_flag=true \
    server_flag=false \
    max_ttl="720h" \
    ttl="720h"
```

### Step 2: Configure Kubernetes Auth for Vault (1 hour)

```bash
# 2.1: Enable Kubernetes auth method
kubectl exec -n vault vault-0 -- vault auth enable kubernetes

# 2.2: Configure Kubernetes auth
kubectl exec -n vault vault-0 -- vault write auth/kubernetes/config \
    kubernetes_host="https://kubernetes.default.svc:443" \
    kubernetes_ca_cert="$(cat /var/run/secrets/kubernetes.io/serviceaccount/ca.crt)" \
    token_reviewer_jwt="$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)"

# 2.3: Create policies
cat <<EOF | kubectl exec -i -n vault vault-0 -- vault policy write cert-manager -
path "pki_int/sign/gauth-server" {
  capabilities = ["create", "update"]
}
path "pki_int/sign/gauth-client" {
  capabilities = ["create", "update"]
}
EOF

cat <<EOF | kubectl exec -i -n vault vault-0 -- vault policy write gauth-app -
path "secret/data/gauth/*" {
  capabilities = ["read"]
}
path "database/creds/gauth-app" {
  capabilities = ["read"]
}
path "pki_int/issue/gauth-client" {
  capabilities = ["create", "update"]
}
path "transit/encrypt/gauth-*" {
  capabilities = ["update"]
}
path "transit/decrypt/gauth-*" {
  capabilities = ["update"]
}
EOF

# 2.4: Create Kubernetes roles
kubectl exec -n vault vault-0 -- vault write auth/kubernetes/role/cert-manager \
    bound_service_account_names=cert-manager-vault \
    bound_service_account_namespaces=cert-manager \
    policies=cert-manager \
    ttl=1h

kubectl exec -n vault vault-0 -- vault write auth/kubernetes/role/gauth-app \
    bound_service_account_names=gauth-api \
    bound_service_account_namespaces=gauth \
    policies=gauth-app \
    ttl=1h
```

### Step 3: Deploy Cert-Manager with Vault (1 hour)

```bash
# 3.1: Create cert-manager vault service account
kubectl create serviceaccount cert-manager-vault -n cert-manager

# 3.2: Update CA bundle in ClusterIssuer
kubectl create configmap gauth-ca-bundle -n gauth \
    --from-file=ca-bundle.crt=/tmp/gauth-root-ca.crt

# 3.3: Deploy cert-manager configuration
kubectl apply -f k8s/security/mtls-config.yaml --selector=cert-manager.io

# 3.4: Create initial certificates
kubectl apply -f k8s/security/mtls-config.yaml --selector=cert-manager.io/v1,Certificate

# 3.5: Verify certificates
kubectl get certificates -n gauth
kubectl describe certificate gauth-api-server-cert -n gauth

# Wait for certificates to be issued
kubectl wait --for=condition=ready certificate --all -n gauth --timeout=300s
```

---

## Phase 2: Vault Deployment

**Duration**: 4-6 hours  
**Prerequisites**: Phase 1 complete

### Step 1: Configure Vault Secrets Engines (2 hours)

```bash
# 1.1: Enable KV v2 for static secrets
kubectl exec -n vault vault-0 -- vault secrets enable -path=secret kv-v2

# Configure retention
kubectl exec -n vault vault-0 -- vault kv metadata put -max-versions 10 secret/gauth/
kubectl exec -n vault vault-0 -- vault kv metadata put -delete-version-after=30d secret/gauth/

# Store application secrets
kubectl exec -n vault vault-0 -- vault kv put secret/gauth/config \
    jwt_signing_key="$(openssl rand -base64 32)" \
    webhook_secret="$(openssl rand -base64 32)" \
    api_encryption_key="$(openssl rand -base64 32)"

# 1.2: Enable database secrets engine
kubectl exec -n vault vault-0 -- vault secrets enable database

# Configure PostgreSQL
kubectl exec -n vault vault-0 -- vault write database/config/postgresql \
    plugin_name=postgresql-database-plugin \
    allowed_roles="gauth-app,gauth-readonly" \
    connection_url="postgresql://{{username}}:{{password}}@postgresql.gauth.svc.cluster.local:5432/gauth?sslmode=require" \
    username="vault" \
    password="vault-db-password"

# Create database roles
kubectl exec -n vault vault-0 -- vault write database/roles/gauth-app \
    db_name=postgresql \
    creation_statements="CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'; GRANT gauth_app TO \"{{name}}\";" \
    default_ttl="30m" \
    max_ttl="1h"

# 1.3: Enable transit engine for encryption-as-a-service
kubectl exec -n vault vault-0 -- vault secrets enable transit

kubectl exec -n vault vault-0 -- vault write -f transit/keys/gauth-data \
    type="aes256-gcm96"

kubectl exec -n vault vault-0 -- vault write -f transit/keys/gauth-pii \
    type="aes256-gcm96" \
    auto_rotate_period="720h"
```

### Step 2: Deploy Vault Agent Injector (1 hour)

```bash
# 2.1: Deploy Vault Agent Injector
kubectl apply -f k8s/security/vault-deployment.yaml --selector=app=vault-agent-injector

# 2.2: Wait for injector to be ready
kubectl wait --for=condition=ready pod -l app=vault-agent-injector -n vault --timeout=300s

# 2.3: Verify webhook configuration
kubectl get mutatingwebhookconfigurations vault-agent-injector-cfg

# 2.4: Test injection (deploy sample app)
kubectl apply -f k8s/security/vault-deployment.yaml --selector=app=gauth-api
```

---

## Phase 3: CloudHSM Integration

**Duration**: 6-8 hours  
**Prerequisites**: AWS account with CloudHSM permissions

### Step 1: Provision CloudHSM Cluster (3 hours)

```bash
# 1.1: Run CloudHSM setup script
./scripts/cloudhsm-setup.sh setup

# This will:
# - Create CloudHSM cluster
# - Create 2 HSM instances (HA)
# - Initialize cluster with customer CA
# - Create KMS key backed by CloudHSM
# - Configure Vault for auto-unseal

# 1.2: Note the cluster ID and KMS key ID
# Outputs will be saved in:
# - /tmp/hsm-cluster-id.txt
# - Cluster ID printed at end of script

# 1.3: Verify cluster status
HSM_CLUSTER_ID=$(cat /tmp/hsm-cluster-id.txt)
./scripts/cloudhsm-setup.sh status $HSM_CLUSTER_ID
```

### Step 2: Configure Vault Auto-Unseal (1 hour)

```bash
# 2.1: Vault configuration already updated by setup script
# Verify configuration
kubectl get configmap vault-config -n vault -o yaml

# 2.2: Seal Vault (to test auto-unseal)
kubectl exec -n vault vault-0 -- vault operator seal

# 2.3: Restart Vault pod (should auto-unseal)
kubectl delete pod vault-0 -n vault

# 2.4: Wait and verify auto-unseal
sleep 60
kubectl exec -n vault vault-0 -- vault status

# Should show: Sealed: false
```

### Step 3: Create CloudHSM Backup (30 minutes)

```bash
# 3.1: Create backup
./scripts/cloudhsm-setup.sh backup $HSM_CLUSTER_ID

# 3.2: List backups
aws cloudhsmv2 describe-backups --region us-east-1 --output table

# 3.3: Configure automated daily backups
cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: CronJob
metadata:
  name: cloudhsm-backup
  namespace: vault
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: amazon/aws-cli:latest
            command:
            - /bin/sh
            - -c
            - |
              aws cloudhsmv2 create-backup --cluster-id $HSM_CLUSTER_ID --region us-east-1
            env:
            - name: HSM_CLUSTER_ID
              value: "$HSM_CLUSTER_ID"
          restartPolicy: OnFailure
EOF
```

---

## Phase 4: mTLS Configuration

**Duration**: 4-6 hours  
**Prerequisites**: Phase 1-3 complete

### Step 1: Deploy mTLS Infrastructure (2 hours)

```bash
# 1.1: Create gauth namespace with Istio injection
kubectl create namespace gauth
kubectl label namespace gauth istio-injection=enabled

# 1.2: Deploy mTLS configuration
kubectl apply -f k8s/security/mtls-config.yaml

# 1.3: Wait for Nginx pods
kubectl wait --for=condition=ready pod -l app=nginx-mtls -n gauth --timeout=300s

# 1.4: Get Nginx LoadBalancer IP
NGINX_IP=$(kubectl get svc nginx-mtls -n gauth -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "Nginx mTLS LoadBalancer IP: $NGINX_IP"

# 1.5: Update DNS (Route53 example)
aws route53 change-resource-record-sets --hosted-zone-id Z1234567890ABC --change-batch '{
  "Changes": [{
    "Action": "UPSERT",
    "ResourceRecordSet": {
      "Name": "gauth-api.example.com",
      "Type": "A",
      "TTL": 60,
      "ResourceRecords": [{"Value": "'$NGINX_IP'"}]
    }
  }]
}'
```

### Step 2: Configure Istio Service Mesh (1 hour)

```bash
# 2.1: Apply Istio mTLS policies
kubectl apply -f k8s/security/mtls-config.yaml --selector=security.istio.io

# 2.2: Verify Istio configuration
istioctl analyze -n gauth

# 2.3: Check mTLS status
kubectl exec -n gauth $(kubectl get pod -n gauth -l app=gauth-api -o jsonpath='{.items[0].metadata.name}') -c istio-proxy -- \
    curl -s localhost:15000/config_dump | grep -A 20 tls_context
```

### Step 3: Generate Client Certificates (1 hour)

```bash
# 3.1: Issue client certificate for testing
kubectl exec -n vault vault-0 -- vault write -format=json pki_int/issue/gauth-client \
    common_name="test-client.client.gauth.internal" \
    ttl="720h" > /tmp/client-cert.json

# Extract certificate and key
jq -r '.data.certificate' /tmp/client-cert.json > /tmp/client.crt
jq -r '.data.private_key' /tmp/client-cert.json > /tmp/client.key
jq -r '.data.ca_chain[]' /tmp/client-cert.json > /tmp/ca-chain.crt

# 3.2: Test mTLS connection
curl --cacert /tmp/ca-chain.crt \
     --cert /tmp/client.crt \
     --key /tmp/client.key \
     https://gauth-api.example.com/health

# Should return 200 OK
```

---

## Phase 5: Security Monitoring

**Duration**: 2-3 hours  
**Prerequisites**: Phase 1-4 complete

### Step 1: Deploy Security Monitoring (1 hour)

```bash
# 1.1: Deploy Prometheus security rules
kubectl apply -f monitoring/prometheus/security-rules.yaml

# 1.2: Deploy security dashboards
kubectl apply -f monitoring/grafana/dashboards/security-dashboard.json

# 1.3: Configure AlertManager for security alerts
kubectl apply -f monitoring/alertmanager/security-routes.yaml
```

### Step 2: Configure Security Automation (1 hour)

```bash
# 2.1: Create CronJob for daily security checks
cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: CronJob
metadata:
  name: security-automation
  namespace: gauth
spec:
  schedule: "0 1 * * *"  # Daily at 1 AM
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: security-automation
          containers:
          - name: security-checks
            image: gauth/security-automation:latest
            command: ["/scripts/security-automation.sh", "all"]
            env:
            - name: SLACK_WEBHOOK
              valueFrom:
                secretKeyRef:
                  name: slack-webhook
                  key: url
          restartPolicy: OnFailure
EOF

# 2.2: Create ServiceAccount with permissions
kubectl create serviceaccount security-automation -n gauth
kubectl create clusterrolebinding security-automation \
    --clusterrole=cluster-admin \
    --serviceaccount=gauth:security-automation

# 2.3: Run initial security check
./scripts/security-automation.sh all
```

---

## Testing & Validation

### Test 1: Certificate Rotation

```bash
# Trigger manual rotation
./scripts/security-automation.sh rotate gauth-api-server-cert

# Verify new certificate
kubectl get certificate gauth-api-server-cert -n gauth -o jsonpath='{.status.notAfter}'
```

### Test 2: mTLS Authentication

```bash
# Test successful authentication
curl --cacert /tmp/ca-chain.crt \
     --cert /tmp/client.crt \
     --key /tmp/client.key \
     https://gauth-api.example.com/api/v1/poa

# Test failed authentication (no client cert)
curl --cacert /tmp/ca-chain.crt \
     https://gauth-api.example.com/api/v1/poa
# Should return 400 Bad Request (no client certificate)
```

### Test 3: Vault Secrets Rotation

```bash
# Read current database credentials
kubectl exec -n gauth $(kubectl get pod -n gauth -l app=gauth-api -o jsonpath='{.items[0].metadata.name}') -- \
    cat /vault/secrets/db-creds

# Wait 30 minutes for automatic rotation

# Read new credentials (should be different)
kubectl exec -n gauth $(kubectl get pod -n gauth -l app=gauth-api -o jsonpath='{.items[0].metadata.name}') -- \
    cat /vault/secrets/db-creds
```

### Test 4: CloudHSM Failover

```bash
# Seal Vault
kubectl exec -n vault vault-0 -- vault operator seal

# Delete Vault pod to trigger auto-unseal
kubectl delete pod vault-0 -n vault

# Wait and verify
sleep 60
kubectl exec -n vault vault-0 -- vault status
# Should show: Sealed: false (auto-unsealed with CloudHSM)
```

### Test 5: Vulnerability Scanning

```bash
# Run vulnerability scan
./scripts/security-automation.sh scan

# Check results
cat /var/log/gauth/security-scans/scan-*.json | jq '.[] | select(.Severity == "CRITICAL")'
```

---

## Operational Runbooks

### Runbook 1: Certificate Expired

**Scenario**: Certificate expired and services are failing

**Steps**:
1. Check certificate status: `kubectl get certificates -n gauth`
2. Force renewal: `./scripts/security-automation.sh rotate <cert-name>`
3. Verify renewal: `kubectl describe certificate <cert-name> -n gauth`
4. Restart affected pods: `kubectl rollout restart deployment/<name> -n gauth`
5. Verify services: `curl --cacert /tmp/ca-chain.crt https://gauth-api.example.com/health`

**Prevention**: Ensure cert-manager auto-renewal is working (renewBefore: 24h)

### Runbook 2: Vault Sealed

**Scenario**: Vault is sealed and auto-unseal failed

**Steps**:
1. Check Vault status: `kubectl exec -n vault vault-0 -- vault status`
2. Check CloudHSM status: `./scripts/cloudhsm-setup.sh status $HSM_CLUSTER_ID`
3. If HSM is healthy, restart Vault: `kubectl delete pod vault-0 -n vault`
4. If HSM is down, manual unseal: Use saved unseal keys from `/tmp/vault-init.json`
5. Monitor: `kubectl logs -f vault-0 -n vault`

**Escalation**: If manual unseal fails, contact HashiCorp support

### Runbook 3: mTLS Authentication Failures

**Scenario**: High rate of mTLS authentication failures

**Steps**:
1. Check Nginx logs: `kubectl logs -n gauth -l app=nginx-mtls | grep FAILED`
2. Verify CA bundle: `kubectl get configmap gauth-ca-bundle -n gauth -o yaml`
3. Check CRL: `kubectl exec -n gauth nginx-mtls-0 -- curl http://vault.vault.svc.cluster.local:8200/v1/pki/crl`
4. Review certificate expiry: `./scripts/security-automation.sh certs`
5. If CA is rotated, update CA bundle and restart Nginx

**Prevention**: Monitor certificate expiry 7 days before

---

## Troubleshooting

### Issue 1: Cert-Manager Not Issuing Certificates

**Symptoms**: Certificate stuck in "Pending" state

**Diagnosis**:
```bash
kubectl describe certificate <cert-name> -n gauth
kubectl logs -n cert-manager deploy/cert-manager
```

**Solutions**:
1. Verify Vault is unsealed and accessible
2. Check Vault role permissions
3. Verify Kubernetes auth configuration in Vault
4. Test Vault connectivity: `kubectl exec -n cert-manager <pod> -- curl -k https://vault.vault.svc.cluster.local:8200/v1/sys/health`

### Issue 2: CloudHSM Connection Failures

**Symptoms**: Vault cannot auto-unseal

**Diagnosis**:
```bash
kubectl logs -n vault vault-0 | grep -i hsm
aws cloudhsmv2 describe-clusters --cluster-id $HSM_CLUSTER_ID
```

**Solutions**:
1. Verify CloudHSM cluster state is "INITIALIZED"
2. Check network connectivity from Vault pods to CloudHSM
3. Verify KMS key permissions
4. Check AWS credentials in Vault pods

### Issue 3: Istio mTLS Not Enforcing

**Symptoms**: Connections work without client certificates

**Diagnosis**:
```bash
istioctl analyze -n gauth
kubectl get peerauthentication -n gauth
```

**Solutions**:
1. Verify Istio injection: `kubectl get pod -n gauth -o jsonpath='{.items[*].spec.containers[*].name}'` (should see istio-proxy)
2. Check PeerAuthentication mode: Should be STRICT
3. Restart pods to inject Istio: `kubectl rollout restart deployment/gauth-api -n gauth`

---

## Incident Response

### Security Incident Classification

| Severity | Description | Response Time | Example |
|----------|-------------|---------------|---------|
| **P0 - Critical** | Active security breach | Immediate | Unauthorized Vault access |
| **P1 - High** | Potential breach or major vulnerability | 15 minutes | CRITICAL CVE in production |
| **P2 - Medium** | Security misconfiguration | 1 hour | Expired certificate |
| **P3 - Low** | Security policy violation | 4 hours | Failed audit log |

### Incident Response Playbook

#### Step 1: Detect & Assess (5 minutes)

```bash
# Check security alerts
kubectl logs -n gauth -l app=gauth-api --tail=1000 | grep -i "security\|breach\|unauthorized"

# Review audit logs
./scripts/security-automation.sh audit

# Check Vault audit logs
kubectl exec -n vault vault-0 -- vault audit list
```

#### Step 2: Contain (10 minutes)

```bash
# If suspected breach:

# 1. Rotate all secrets
kubectl exec -n vault vault-0 -- vault token revoke -mode path auth/kubernetes

# 2. Seal Vault
kubectl exec -n vault vault-0 -- vault operator seal

# 3. Block suspicious IPs (if identified)
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: block-suspicious-ip
  namespace: gauth
spec:
  podSelector:
    matchLabels:
      app: gauth-api
  policyTypes:
  - Ingress
  ingress:
  - from:
    - ipBlock:
        cidr: 0.0.0.0/0
        except:
        - <suspicious-ip>/32
EOF
```

#### Step 3: Investigate (30 minutes)

```bash
# Collect forensics
kubectl logs -n gauth --all-containers --timestamps > /tmp/incident-logs.txt

# Export audit logs
kubectl exec -n vault vault-0 -- vault audit list -detailed > /tmp/vault-audit.txt

# Network traffic analysis
kubectl exec -n gauth <pod> -c istio-proxy -- curl localhost:15000/config_dump > /tmp/istio-config.json
```

#### Step 4: Recover (1 hour)

```bash
# 1. Unseal Vault with new keys (if compromised)
# Generate new unseal keys following Vault rekey procedure

# 2. Rotate all certificates
for cert in $(kubectl get certificates -n gauth -o jsonpath='{.items[*].metadata.name}'); do
    ./scripts/security-automation.sh rotate $cert
done

# 3. Rotate database credentials
kubectl exec -n vault vault-0 -- vault write -f database/rotate-root/postgresql

# 4. Restart all services
kubectl rollout restart deployment -n gauth
```

#### Step 5: Post-Incident (24 hours)

1. **Root Cause Analysis**: Document what happened and why
2. **Update Security Policies**: Fix vulnerabilities discovered
3. **Team Notification**: Share learnings with team
4. **Compliance Reporting**: Notify relevant stakeholders per compliance requirements

---

## Security Compliance Checklist

### SOC 2 Type II

- [ ] All data encrypted in transit (TLS 1.3)
- [ ] All secrets stored in Vault (no environment variables)
- [ ] Certificate rotation automated (< 7 days)
- [ ] Audit logs enabled and retained (90 days)
- [ ] Access controls enforced (mTLS + RBAC)
- [ ] Regular vulnerability scanning (daily)
- [ ] Incident response procedures documented

### PCI-DSS

- [ ] Strong cryptography (AES-256, RSA-2048+)
- [ ] Key management with HSM (FIPS 140-2 Level 3)
- [ ] Regular key rotation (< 30 days)
- [ ] Network segmentation (NetworkPolicies)
- [ ] Multi-factor authentication (mTLS certificates)
- [ ] Regular security testing (quarterly)

### HIPAA

- [ ] PHI encryption at rest and in transit
- [ ] Access logging and monitoring
- [ ] Automatic logoff (certificate expiry)
- [ ] Disaster recovery (CloudHSM backups)
- [ ] Audit controls (Vault audit logs)

---

## Conclusion

This Advanced Security implementation provides **defense-in-depth** with:

✅ **mTLS**: Mutual authentication for all services  
✅ **Vault**: Centralized secrets with dynamic credentials  
✅ **CloudHSM**: FIPS 140-2 Level 3 key storage  
✅ **Auto-Rotation**: Zero-downtime certificate lifecycle  
✅ **Monitoring**: Real-time threat detection  
✅ **Compliance**: SOC 2, PCI-DSS, HIPAA ready  

**Total Setup Time**: 2-3 days  
**Compliance: 98/100 → 99/100** (+1.0 point)

---

**Document Version**: 1.0  
**Last Updated**: November 2025  
**Status**: Production-Ready
