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
gauth-prod-2025-11.priv.pem
gauth-prod-2025-11.pub.pem
gauth-prod-2025-12.priv.pem
gauth-prod-2025-12.pub.pem
```

---

## Environment Configuration

### Required Environment Variables

```bash
# Enable/disable JWE encryption
export GAUTH_JWE_ENABLED=true

# Key encryption algorithm (RSA-OAEP-256 recommended)
export GAUTH_JWE_ALGORITHM=RSA-OAEP-256

# Content encryption algorithm (A256GCM recommended)
export GAUTH_JWE_ENCRYPTION=A256GCM

# Path to RSA public key
export GAUTH_JWE_PUBLIC_KEY=/etc/gauth/keys/public.pem

# Path to RSA private key
export GAUTH_JWE_PRIVATE_KEY=/etc/gauth/keys/private.pem

# Key identifier (for rotation)
export GAUTH_JWE_KEY_ID=gauth-prod-2025-11

# Key rotation interval (days)
export GAUTH_JWE_ROTATION_DAYS=365
```

### Multi-Key Registry (Optional)

For zero-downtime key rotation, use key directory:

```bash
# Directory containing multiple key pairs
export GAUTH_JWE_KEY_DIR=/etc/gauth/keys

# Key files in directory:
# - gauth-prod-2025-11.priv.pem
# - gauth-prod-2025-11.pub.pem
# - gauth-prod-2025-12.priv.pem (new key)
# - gauth-prod-2025-12.pub.pem (new key)
```

### Configuration Validation

```bash
# Validate environment configuration
go run ./cmd/web-server --validate-env

# Expected output:
# ✓ GAUTH_JWE_ENABLED: true
# ✓ GAUTH_JWE_ALGORITHM: RSA-OAEP-256
# ✓ GAUTH_JWE_ENCRYPTION: A256GCM
# ✓ GAUTH_JWE_PUBLIC_KEY: /etc/gauth/keys/public.pem (file exists)
# ✓ GAUTH_JWE_PRIVATE_KEY: /etc/gauth/keys/private.pem (file exists)
# ✓ GAUTH_JWE_KEY_ID: gauth-prod-2025-11
# Configuration valid ✓
```

---

## Docker Deployment

### Build Image

```bash
# Build production image with JWE support
docker build \
  -f deployments/docker/Dockerfile.jwe \
  -t gauth:jwe-latest \
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
  --name gauth-server \
  -p 8080:8080 \
  -v $(pwd)/keys:/etc/gauth/keys:ro \
  -e GAUTH_JWE_ENABLED=true \
  -e GAUTH_JWE_PUBLIC_KEY=/etc/gauth/keys/public.pem \
  -e GAUTH_JWE_PRIVATE_KEY=/etc/gauth/keys/private.pem \
  -e GAUTH_JWE_KEY_ID=gauth-prod-2025-11 \
  gauth:jwe-latest
```

### Docker Compose Deployment

```bash
# Navigate to deployment directory
cd deployments/docker

# Copy keys to deployment directory
cp ../../public.pem ./keys/
cp ../../private.pem ./keys/

# Start all services
docker-compose -f docker-compose.jwe.yml up -d

# Check logs
docker-compose -f docker-compose.jwe.yml logs -f gauth-server

# Stop services
docker-compose -f docker-compose.jwe.yml down
```

### Docker Secrets (Production)

```bash
# Create Docker secrets
echo "$(cat private.pem)" | docker secret create gauth_private_key -
echo "$(cat public.pem)" | docker secret create gauth_public_key -

# Deploy with secrets
docker service create \
  --name gauth-server \
  --secret source=gauth_private_key,target=/run/secrets/private.pem,mode=0400 \
  --secret source=gauth_public_key,target=/run/secrets/public.pem,mode=0444 \
  -e GAUTH_JWE_PRIVATE_KEY=/run/secrets/private.pem \
  -e GAUTH_JWE_PUBLIC_KEY=/run/secrets/public.pem \
  gauth:jwe-latest
```

---

## Kubernetes Deployment

### Create Namespace

```bash
kubectl create namespace gauth
```

### Generate Secrets

```bash
# Create secret from key files
kubectl create secret generic gauth-jwe-keys \
  --from-file=public.pem=./public.pem \
  --from-file=private.pem=./private.pem \
  --namespace=gauth

# Verify secret
kubectl describe secret gauth-jwe-keys -n gauth
```

### Alternative: Base64 Encoding

```bash
# Base64 encode keys for manifest
cat public.pem | base64 -w 0 > public.pem.b64
cat private.pem | base64 -w 0 > private.pem.b64

# Edit gauth-jwe-deployment.yaml and paste base64 values
```

### Deploy Application

```bash
# Apply deployment manifest
kubectl apply -f deployments/kubernetes/gauth-jwe-deployment.yaml

# Check deployment status
kubectl get deployments -n gauth
kubectl get pods -n gauth

# Check logs
kubectl logs -f deployment/gauth-server -n gauth

# Check service
kubectl get svc -n gauth
```

### Verify Deployment

```bash
# Port forward to test locally
kubectl port-forward svc/gauth-service 8080:8080 -n gauth

# Test health endpoint
curl http://localhost:8080/health
```

### Update Configuration

```bash
# Edit ConfigMap
kubectl edit configmap gauth-jwe-config -n gauth

# Restart pods to apply changes
kubectl rollout restart deployment/gauth-server -n gauth

# Monitor rollout
kubectl rollout status deployment/gauth-server -n gauth
```

---

## Key Rotation Strategy

### Zero-Downtime Key Rotation

**Step 1: Generate New Key Pair**

```bash
# Generate new key pair with new key ID
openssl genpkey -algorithm RSA \
  -out gauth-prod-2025-12.priv.pem \
  -pkeyopt rsa_keygen_bits:2048

openssl rsa -pubout \
  -in gauth-prod-2025-12.priv.pem \
  -out gauth-prod-2025-12.pub.pem
```

**Step 2: Add New Key to Registry**

For Kubernetes:

```bash
# Create new secret with both old and new keys
kubectl create secret generic gauth-jwe-keys-v2 \
  --from-file=gauth-prod-2025-11.priv.pem=./old-private.pem \
  --from-file=gauth-prod-2025-11.pub.pem=./old-public.pem \
  --from-file=gauth-prod-2025-12.priv.pem=./new-private.pem \
  --from-file=gauth-prod-2025-12.pub.pem=./new-public.pem \
  --namespace=gauth

# Update deployment to use key directory
# Edit deployment YAML:
#   GAUTH_JWE_KEY_DIR=/etc/gauth/keys
# (instead of individual key paths)

# Apply updated deployment
kubectl apply -f deployments/kubernetes/gauth-jwe-deployment.yaml
```

**Step 3: Update ConfigMap to Use New Key for Encryption**

```bash
# Update key ID in ConfigMap
kubectl patch configmap gauth-jwe-config \
  -n gauth \
  --type merge \
  -p '{"data":{"GAUTH_JWE_KEY_ID":"gauth-prod-2025-12"}}'

# Restart pods
kubectl rollout restart deployment/gauth-server -n gauth
```

**Step 4: Wait for Token Expiry**

```bash
# Wait for all old tokens to expire (typically 1 hour)
# Monitor token age metrics in Grafana
```

**Step 5: Remove Old Key**

```bash
# Create new secret without old key
kubectl create secret generic gauth-jwe-keys-v3 \
  --from-file=gauth-prod-2025-12.priv.pem=./new-private.pem \
  --from-file=gauth-prod-2025-12.pub.pem=./new-public.pem \
  --namespace=gauth

# Update deployment
kubectl apply -f deployments/kubernetes/gauth-jwe-deployment.yaml
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
// pkg/gauth/jwe_hsm_aws.go
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
kubectl exec -it deployment/gauth-server -n gauth -- \
  ls -la /etc/gauth/keys/

# Verify GAUTH_JWE_KEY_ID matches file naming
kubectl get configmap gauth-jwe-config -n gauth -o yaml
```

### Issue 2: "Invalid JWE format"

**Cause**: Token is not JWE-encrypted (plain JWT)

**Solution:**

```bash
# Check token format
# JWE: 5 parts (header.key.iv.ciphertext.tag)
# JWT: 3 parts (header.payload.signature)

echo "$TOKEN" | awk -F. '{print NF}'

# If 3 parts, check GAUTH_JWE_ENABLED
kubectl get configmap gauth-jwe-config -n gauth -o yaml | grep ENABLED
```

### Issue 3: "Performance degradation"

**Cause**: Large token size, network latency, or CPU exhaustion

**Solution:**

```bash
# Check token size
echo -n "$JWE_TOKEN" | wc -c

# Check JWE compression enabled
kubectl get configmap gauth-jwe-config -n gauth -o yaml | grep COMPRESSION

# Check CPU usage
kubectl top pods -n gauth

# Scale horizontally if needed
kubectl scale deployment/gauth-server --replicas=5 -n gauth
```

### Issue 4: "Old tokens fail after key rotation"

**Cause**: Old key removed too soon

**Solution:**

```bash
# Re-add old key to registry temporarily
kubectl create secret generic gauth-jwe-keys \
  --from-file=gauth-prod-2025-11.priv.pem=./old-private.pem \
  --from-file=gauth-prod-2025-12.priv.pem=./new-private.pem \
  --namespace=gauth \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart pods
kubectl rollout restart deployment/gauth-server -n gauth
```

### Debug Mode

Enable debug logging:

```bash
# Set log level
kubectl patch configmap gauth-jwe-config \
  -n gauth \
  --type merge \
  -p '{"data":{"GAUTH_LOG_LEVEL":"debug"}}'

# Restart pods
kubectl rollout restart deployment/gauth-server -n gauth

# View detailed logs
kubectl logs -f deployment/gauth-server -n gauth | grep JWE
```

---

## Appendix

### A. Environment Variable Reference

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `GAUTH_JWE_ENABLED` | bool | `true` | Enable/disable JWE encryption |
| `GAUTH_JWE_ALGORITHM` | string | `RSA-OAEP-256` | Key encryption algorithm |
| `GAUTH_JWE_ENCRYPTION` | string | `A256GCM` | Content encryption algorithm |
| `GAUTH_JWE_PUBLIC_KEY` | path | - | RSA public key file path |
| `GAUTH_JWE_PRIVATE_KEY` | path | - | RSA private key file path |
| `GAUTH_JWE_KEY_ID` | string | - | Key identifier |
| `GAUTH_JWE_KEY_DIR` | path | - | Multi-key directory |
| `GAUTH_JWE_ROTATION_DAYS` | int | `365` | Key rotation interval |

### B. File Structure

```
/etc/gauth/
├── keys/
│   ├── gauth-prod-2025-11.pub.pem    (644)
│   ├── gauth-prod-2025-11.priv.pem   (400)
│   ├── gauth-prod-2025-12.pub.pem    (644)
│   └── gauth-prod-2025-12.priv.pem   (400)
├── config/
│   └── gauth.yaml
└── logs/
    └── gauth.log
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
