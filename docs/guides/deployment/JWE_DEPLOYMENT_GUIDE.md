# AgentAuth JWE Encryption - Production Deployment Guide

**Version**: 1.0  
**Date**: November 12, 2025  
**Status**: Phase 3 Complete

---

## Table of Contents

1. [Overview](#overview)
2. [Key Generation](#key-generation)
3. [Environment Configuration](#environment-configuration)
4. [Docker Deployment](#docker-deployment)
5. [Kubernetes Deployment](#kubernetes-deployment)
6. [Key Rotation Strategy](#key-rotation-strategy)
7. [Monitoring and Metrics](#monitoring-and-metrics)
8. [Security Best Practices](#security-best-practices)
9. [Troubleshooting](#troubleshooting)

---

## Overview

AgentAuth implements JWE (JSON Web Encryption) for Extended Token protection, providing:

- **Encryption**: RSA-OAEP-256 + AES-256-GCM (default)
- **Compression**: DEFLATE (reduces token size)
- **Key Rotation**: Multi-key registry support (zero-downtime rotation)
- **Performance**: 126μs encryption, 833μs decryption, 1.02ms full cycle

### Architecture

```
┌─────────────┐                  ┌──────────────┐
│   Client    │                  │  Resource    │
│ Application │                  │   Server     │
└──────┬──────┘                  └───────┬──────┘
       │                                 │
       │ 1. Token Request                │
       ├────────────────────────────────►│
       │                                 │
       │                     ┌───────────▼───────────┐
       │                     │  Authorization Server │
       │                     │  ┌─────────────────┐  │
       │                     │  │ JWE Service     │  │
       │                     │  │ - Encrypt JWT   │  │
       │                     │  │ - Key Registry  │  │
       │                     │  └─────────────────┘  │
       │                     └───────────┬───────────┘
       │                                 │
       │ 2. JWE Token (encrypted)        │
       │◄────────────────────────────────┤
       │                                 │
       │ 3. API Request + JWE Token      │
       ├────────────────────────────────►│
       │                                 │
       │                     ┌───────────▼───────────┐
       │                     │  Resource Server      │
       │                     │  ┌─────────────────┐  │
       │                     │  │ JWE Service     │  │
       │                     │  │ - Decrypt JWE   │  │
       │                     │  │ - Parse JWT     │  │
       │                     │  │ - Validate      │  │
       │                     │  └─────────────────┘  │
       │                     └───────────┬───────────┘
       │                                 │
       │ 4. Protected Resource           │
       │◄────────────────────────────────┤
       │                                 │
```

---

## Key Generation

### Generate RSA Key Pair (2048-bit)

```bash
# Generate private key
openssl genpkey -algorithm RSA \
  -out private.pem \
  -pkeyopt rsa_keygen_bits:2048

# Extract public key
openssl rsa -pubout \
  -in private.pem \
  -out public.pem

# Verify key size
openssl rsa -in private.pem -text -noout | grep "Private-Key"
# Expected: Private-Key: (2048 bit)
```

### Generate RSA Key Pair (4096-bit) - Higher Security

```bash
# Generate private key (4096-bit)
openssl genpkey -algorithm RSA \
  -out private-4096.pem \
  -pkeyopt rsa_keygen_bits:4096

# Extract public key
openssl rsa -pubout \
  -in private-4096.pem \
  -out public-4096.pem
```

### File Permissions (Critical)

```bash
# Private key: owner read-only
chmod 400 private.pem

# Public key: world-readable
chmod 644 public.pem
```

### Key Naming Convention

For key rotation support, use sortable naming:

```
agentauth-prod-2025-11.priv.pem
agentauth-prod-2025-11.pub.pem
agentauth-prod-2025-12.priv.pem
agentauth-prod-2025-12.pub.pem
```

---

## Environment Configuration

### Required Environment Variables

```bash
# Enable/disable JWE encryption
export AGENTAUTH_JWE_ENABLED=true

# Key encryption algorithm (RSA-OAEP-256 recommended)
export AGENTAUTH_JWE_ALGORITHM=RSA-OAEP-256

# Content encryption algorithm (A256GCM recommended)
export AGENTAUTH_JWE_ENCRYPTION=A256GCM

# Path to RSA public key
export AGENTAUTH_JWE_PUBLIC_KEY=/etc/agentauth/keys/public.pem

# Path to RSA private key
export AGENTAUTH_JWE_PRIVATE_KEY=/etc/agentauth/keys/private.pem

# Key identifier (for rotation)
export AGENTAUTH_JWE_KEY_ID=agentauth-prod-2025-11

# Key rotation interval (days)
export AGENTAUTH_JWE_ROTATION_DAYS=365
```

### Multi-Key Registry (Optional)

For zero-downtime key rotation, use key directory:

```bash
# Directory containing multiple key pairs
export AGENTAUTH_JWE_KEY_DIR=/etc/agentauth/keys

# Key files in directory:
# - agentauth-prod-2025-11.priv.pem
# - agentauth-prod-2025-11.pub.pem
# - agentauth-prod-2025-12.priv.pem (new key)
# - agentauth-prod-2025-12.pub.pem (new key)
```

### Configuration Validation

```bash
# Validate environment configuration
go run ./cmd/web-server --validate-env

# Expected output:
# ✓ AGENTAUTH_JWE_ENABLED: true
# ✓ AGENTAUTH_JWE_ALGORITHM: RSA-OAEP-256
# ✓ AGENTAUTH_JWE_ENCRYPTION: A256GCM
# ✓ AGENTAUTH_JWE_PUBLIC_KEY: /etc/agentauth/keys/public.pem (file exists)
# ✓ AGENTAUTH_JWE_PRIVATE_KEY: /etc/agentauth/keys/private.pem (file exists)
# ✓ AGENTAUTH_JWE_KEY_ID: agentauth-prod-2025-11
# Configuration valid ✓
```

---

## Docker Deployment

### Build Image

```bash
# Build production image with JWE support
docker build \
  -f deployments/docker/Dockerfile.jwe \
  -t agentauth:jwe-latest \
  .
```

### Run Standalone Container

```bash
# Create keys directory
mkdir -p ./keys
cp public.pem ./keys/
cp private.pem ./keys/

# Run container with volume mount
docker run -d \
  --name agentauth-server \
  -p 8080:8080 \
  -v $(pwd)/keys:/etc/agentauth/keys:ro \
  -e AGENTAUTH_JWE_ENABLED=true \
  -e AGENTAUTH_JWE_PUBLIC_KEY=/etc/agentauth/keys/public.pem \
  -e AGENTAUTH_JWE_PRIVATE_KEY=/etc/agentauth/keys/private.pem \
  -e AGENTAUTH_JWE_KEY_ID=agentauth-prod-2025-11 \
  agentauth:jwe-latest
```

### Docker Compose Deployment

```bash
# Navigate to deployment directory
cd deployments/docker

# Copy keys to deployment directory
cp ../../public.pem ./keys/
cp ../../private.pem ./keys/

# Start all services
docker compose -f docker-compose.jwe.yml up -d

# Check logs
docker compose -f docker-compose.jwe.yml logs -f agentauth-server

# Stop services
docker compose -f docker-compose.jwe.yml down
```

### Docker Secrets (Production)

```bash
# Create Docker secrets
echo "$(cat private.pem)" | docker secret create agentauth_private_key -
echo "$(cat public.pem)" | docker secret create agentauth_public_key -

# Deploy with secrets
docker service create \
  --name agentauth-server \
  --secret source=agentauth_private_key,target=/run/secrets/private.pem,mode=0400 \
  --secret source=agentauth_public_key,target=/run/secrets/public.pem,mode=0444 \
  -e AGENTAUTH_JWE_PRIVATE_KEY=/run/secrets/private.pem \
  -e AGENTAUTH_JWE_PUBLIC_KEY=/run/secrets/public.pem \
  agentauth:jwe-latest
```

---

## Kubernetes Deployment

### Create Namespace

```bash
kubectl create namespace agentauth
```

### Generate Secrets

```bash
# Create secret from key files
kubectl create secret generic agentauth-jwe-keys \
  --from-file=public.pem=./public.pem \
  --from-file=private.pem=./private.pem \
  --namespace=agentauth

# Verify secret
kubectl describe secret agentauth-jwe-keys -n agentauth
```

### Alternative: Base64 Encoding

```bash
# Base64 encode keys for manifest
cat public.pem | base64 -w 0 > public.pem.b64
cat private.pem | base64 -w 0 > private.pem.b64

# Edit agentauth-jwe-deployment.yaml and paste base64 values
```

### Deploy Application

```bash
# Apply deployment manifest
kubectl apply -f deployments/kubernetes/agentauth-jwe-deployment.yaml

# Check deployment status
kubectl get deployments -n agentauth
kubectl get pods -n agentauth

# Check logs
kubectl logs -f deployment/agentauth-server -n agentauth

# Check service
kubectl get svc -n agentauth
```

### Verify Deployment

```bash
# Port forward to test locally
kubectl port-forward svc/agentauth-service 8080:8080 -n agentauth

# Test health endpoint
curl http://localhost:8080/health
```

### Update Configuration

```bash
# Edit ConfigMap
kubectl edit configmap agentauth-jwe-config -n agentauth

# Restart pods to apply changes
kubectl rollout restart deployment/agentauth-server -n agentauth

# Monitor rollout
kubectl rollout status deployment/agentauth-server -n agentauth
```

---

## Key Rotation Strategy

### Zero-Downtime Key Rotation

**Step 1: Generate New Key Pair**

```bash
# Generate new key pair with new key ID
openssl genpkey -algorithm RSA \
  -out agentauth-prod-2025-12.priv.pem \
  -pkeyopt rsa_keygen_bits:2048

openssl rsa -pubout \
  -in agentauth-prod-2025-12.priv.pem \
  -out agentauth-prod-2025-12.pub.pem
```

**Step 2: Add New Key to Registry**

For Kubernetes:

```bash
# Create new secret with both old and new keys
kubectl create secret generic agentauth-jwe-keys-v2 \
  --from-file=agentauth-prod-2025-11.priv.pem=./old-private.pem \
  --from-file=agentauth-prod-2025-11.pub.pem=./old-public.pem \
  --from-file=agentauth-prod-2025-12.priv.pem=./new-private.pem \
  --from-file=agentauth-prod-2025-12.pub.pem=./new-public.pem \
  --namespace=agentauth

# Update deployment to use key directory
# Edit deployment YAML:
#   AGENTAUTH_JWE_KEY_DIR=/etc/agentauth/keys
# (instead of individual key paths)

# Apply updated deployment
kubectl apply -f deployments/kubernetes/agentauth-jwe-deployment.yaml
```

**Step 3: Update ConfigMap to Use New Key for Encryption**

```bash
# Update key ID in ConfigMap
kubectl patch configmap agentauth-jwe-config \
  -n agentauth \
  --type merge \
  -p '{"data":{"AGENTAUTH_JWE_KEY_ID":"agentauth-prod-2025-12"}}'

# Restart pods
kubectl rollout restart deployment/agentauth-server -n agentauth
```

**Step 4: Wait for Token Expiry**

```bash
# Wait for all old tokens to expire (typically 1 hour)
# Monitor token age metrics in Grafana
```

**Step 5: Remove Old Key**

```bash
# Create new secret without old key
kubectl create secret generic agentauth-jwe-keys-v3 \
  --from-file=agentauth-prod-2025-12.priv.pem=./new-private.pem \
  --from-file=agentauth-prod-2025-12.pub.pem=./new-public.pem \
  --namespace=agentauth

# Update deployment
kubectl apply -f deployments/kubernetes/agentauth-jwe-deployment.yaml
```

### Rotation Frequency Recommendations

- **Production**: Annual rotation (365 days)
- **Staging**: Quarterly rotation (90 days)
- **Development**: Monthly rotation (30 days)
- **Post-Incident**: Immediate rotation

---

## Monitoring and Metrics

### Key Metrics to Monitor

See `examples/jwe/MONITORING.md` for complete metrics guide.

**Performance Metrics:**
- `jwe_encryption_duration_microseconds` (histogram)
- `jwe_decryption_duration_microseconds` (histogram)

**Operational Metrics:**
- `jwe_encryption_total` (counter)
- `jwe_decryption_total` (counter)
- `jwe_encryption_failures_total` (counter)
- `jwe_decryption_failures_total` (counter)

**Key Management Metrics:**
- `jwe_active_keys` (gauge)
- `jwe_key_rotation_total` (counter)
- `jwe_key_age_days` (gauge)

### Grafana Dashboard

Import dashboard from `examples/jwe/grafana-dashboard.json`:

```bash
# Import via Grafana UI or API
curl -X POST \
  http://grafana:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @examples/jwe/grafana-dashboard.json
```

### Alerts

Critical alerts (see `examples/jwe/MONITORING.md`):

1. **High Encryption Failure Rate** (> 5% over 5 min)
2. **High Decryption Failure Rate** (> 10% over 5 min)
3. **Encryption Performance Degradation** (P99 > 500μs)
4. **Key Rotation Overdue** (> 365 days)
5. **Increasing Decryption Errors** (> 10/hour) - potential attack

---

## Security Best Practices

### Key Storage

**DO:**
- ✅ Store private keys in Kubernetes Secrets (encrypted at rest)
- ✅ Use Hardware Security Modules (HSM) for production keys
- ✅ Use AWS KMS, Azure Key Vault, GCP Cloud KMS for cloud deployments
- ✅ Set file permissions to 400 (owner read-only)
- ✅ Rotate keys annually (minimum)
- ✅ Use 2048-bit RSA keys (minimum), 4096-bit preferred

**DON'T:**
- ❌ Store private keys in Git repositories
- ❌ Store private keys in ConfigMaps (unencrypted)
- ❌ Share private keys between environments
- ❌ Use the same key for multiple services
- ❌ Transmit private keys over insecure channels

### HSM Integration

For high-security deployments, integrate with HSM:

**AWS CloudHSM Example:**

```go
// pkg/agentauth/jwe_hsm_aws.go
import (
    "github.com/aws/aws-sdk-go/service/cloudhsmv2"
)

func NewJWEServiceWithHSM(hsmConfig *CloudHSMConfig) (*JWEService, error) {
    // Use HSM for private key operations
    // Public key can be exported for encryption
}
```

**PKCS#11 Example:**

```go
import (
    "github.com/miekg/pkcs11"
)

func NewJWEServiceWithPKCS11(p11Config *PKCS11Config) (*JWEService, error) {
    // Connect to PKCS#11 device (YubiHSM, etc.)
}
```

### Network Security

- **TLS Required**: Always use HTTPS/TLS 1.3 for token transmission
- **Certificate Pinning**: Pin resource server certificates
- **mTLS**: Use mutual TLS between authorization server and resource server

### Token Lifetime

- **Access Token**: 1 hour (default)
- **Refresh Token**: 30 days (default)
- **Balance**: Shorter = more secure, Longer = better UX

### Compliance

- **FIPS 140-2**: Use FIPS-validated crypto libraries for government deployments
- **PCI DSS**: JWE encryption satisfies tokenization requirements
- **GDPR**: JWE encryption protects PII in tokens

---

## Troubleshooting

### Issue 1: "JWE decryption failed: private key not found"

**Cause**: Key ID mismatch or key not loaded

**Solution:**

```bash
# Check key ID in JWE token
echo "$JWE_TOKEN" | cut -d. -f1 | base64 -d | jq .kid

# Check loaded keys
kubectl exec -it deployment/agentauth-server -n agentauth -- \
  ls -la /etc/agentauth/keys/

# Verify AGENTAUTH_JWE_KEY_ID matches file naming
kubectl get configmap agentauth-jwe-config -n agentauth -o yaml
```

### Issue 2: "Invalid JWE format"

**Cause**: Token is not JWE-encrypted (plain JWT)

**Solution:**

```bash
# Check token format
# JWE: 5 parts (header.key.iv.ciphertext.tag)
# JWT: 3 parts (header.payload.signature)

echo "$TOKEN" | awk -F. '{print NF}'

# If 3 parts, check AGENTAUTH_JWE_ENABLED
kubectl get configmap agentauth-jwe-config -n agentauth -o yaml | grep ENABLED
```

### Issue 3: "Performance degradation"

**Cause**: Large token size, network latency, or CPU exhaustion

**Solution:**

```bash
# Check token size
echo -n "$JWE_TOKEN" | wc -c

# Check JWE compression enabled
kubectl get configmap agentauth-jwe-config -n agentauth -o yaml | grep COMPRESSION

# Check CPU usage
kubectl top pods -n agentauth

# Scale horizontally if needed
kubectl scale deployment/agentauth-server --replicas=5 -n agentauth
```

### Issue 4: "Old tokens fail after key rotation"

**Cause**: Old key removed too soon

**Solution:**

```bash
# Re-add old key to registry temporarily
kubectl create secret generic agentauth-jwe-keys \
  --from-file=agentauth-prod-2025-11.priv.pem=./old-private.pem \
  --from-file=agentauth-prod-2025-12.priv.pem=./new-private.pem \
  --namespace=agentauth \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart pods
kubectl rollout restart deployment/agentauth-server -n agentauth
```

### Debug Mode

Enable debug logging:

```bash
# Set log level
kubectl patch configmap agentauth-jwe-config \
  -n agentauth \
  --type merge \
  -p '{"data":{"AGENTAUTH_LOG_LEVEL":"debug"}}'

# Restart pods
kubectl rollout restart deployment/agentauth-server -n agentauth

# View detailed logs
kubectl logs -f deployment/agentauth-server -n agentauth | grep JWE
```

---

## Appendix

### A. Environment Variable Reference

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `AGENTAUTH_JWE_ENABLED` | bool | `true` | Enable/disable JWE encryption |
| `AGENTAUTH_JWE_ALGORITHM` | string | `RSA-OAEP-256` | Key encryption algorithm |
| `AGENTAUTH_JWE_ENCRYPTION` | string | `A256GCM` | Content encryption algorithm |
| `AGENTAUTH_JWE_PUBLIC_KEY` | path | - | RSA public key file path |
| `AGENTAUTH_JWE_PRIVATE_KEY` | path | - | RSA private key file path |
| `AGENTAUTH_JWE_KEY_ID` | string | - | Key identifier |
| `AGENTAUTH_JWE_KEY_DIR` | path | - | Multi-key directory |
| `AGENTAUTH_JWE_ROTATION_DAYS` | int | `365` | Key rotation interval |

### B. File Structure

```
/etc/agentauth/
├── keys/
│   ├── agentauth-prod-2025-11.pub.pem    (644)
│   ├── agentauth-prod-2025-11.priv.pem   (400)
│   ├── agentauth-prod-2025-12.pub.pem    (644)
│   └── agentauth-prod-2025-12.priv.pem   (400)
├── config/
│   └── agentauth.yaml
└── logs/
    └── agentauth.log
```

### C. Quick Start Checklist

- [ ] Generate RSA key pair (2048-bit minimum)
- [ ] Set file permissions (private: 400, public: 644)
- [ ] Set environment variables
- [ ] Validate configuration (`--validate-env`)
- [ ] Deploy application (Docker or Kubernetes)
- [ ] Verify health endpoint (`/health`)
- [ ] Test token encryption/decryption
- [ ] Configure monitoring (Prometheus + Grafana)
- [ ] Set up alerts (critical: encryption/decryption failures)
- [ ] Document key rotation schedule
- [ ] Test key rotation procedure
- [ ] Backup private keys (encrypted backup)

---

**Document Version**: 1.0  
**Last Updated**: November 12, 2025  
**Maintainer**: AgentAuth Development Team  
**License**: See LICENSE file
