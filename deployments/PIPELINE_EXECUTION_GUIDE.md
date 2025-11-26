---
title: CI/CD Pipeline Execution Guide (Week 4 Day 4)
category: deployment-guide
status: active
lastUpdated: 2025-11-12
owners: platform-team
source: internal-preproduction-runbook
refreshCadence: weekly
---
# Week 4 Day 4: CI/CD Pipeline Execution Guide

**Date**: November 9, 2025  
**Phase**: Pre-Production Week 4 Day 4  
**Focus**: GitHub Actions Pipeline Execution and Monitoring

---

## Executive Summary

This guide documents the process for pushing code to GitHub, triggering the CI/CD pipeline, and monitoring the deployment to the staging environment. The pipeline consists of 5 jobs that will validate, build, and deploy the GAuth application automatically.

**Status**: Ready to push 23 commits to trigger pipeline  
**Prerequisites**: GitHub secrets must be configured (see `GITHUB_ACTIONS_SETUP.md`)  
**Expected Duration**: ~15-20 minutes for full pipeline execution  
**Pipeline Jobs**: 5 (test, security, build, deploy, rollback-on-failure)

---

## 1. Pre-Push Checklist

### ✅ Local Validation Complete
- [x] All 23 commits ready to push
- [x] Working tree clean (no uncommitted changes)
- [x] Branch: main
- [x] Remote: https://github.com/mauriciomferz/Gauth_go
- [x] Workflow YAML syntax validated
- [x] All 15 required Kubernetes manifests present
- [x] Scripts executable (preflight-check.sh, validate-cicd.sh, switch-traffic.sh)

### ⚠️ Required GitHub Secrets

Before pushing, verify these 6 secrets are configured at:  
**https://github.com/mauriciomferz/Gauth_go/settings/secrets/actions**

| Secret Name | Required | Description | Example Value |
|-------------|----------|-------------|---------------|
| `DOCKER_REGISTRY` | ✅ Yes | Docker registry URL | `ghcr.io` |
| `DOCKER_USERNAME` | ✅ Yes | Registry username | `mauriciomferz` |
| `DOCKER_PASSWORD` | ✅ Yes | Registry password/PAT | `ghp_xxxxxxxxxxxx` (GitHub PAT) |
| `KUBE_CONFIG_STAGING` | ✅ Yes | Base64 kubeconfig | `apiVersion: v1\nclusters:...` (base64) |
| `SLACK_WEBHOOK_URL` | ✅ Yes | Slack webhook URL | `https://hooks.slack.com/services/xxx` |
| `CODECOV_TOKEN` | ⚠️ Optional | Codecov token | `xxxx-xxxx-xxxx` |

**How to configure**: See `deployments/GITHUB_ACTIONS_SETUP.md` Section 2 for detailed instructions.

### 🔧 Kubernetes Cluster Prerequisites

Verify staging cluster is ready:

```bash
# Check cluster connectivity
kubectl cluster-info

# Expected output:
# Kubernetes control plane is running at https://...
# CoreDNS is running at https://...

# Check nodes
kubectl get nodes

# Expected: All nodes in "Ready" state

# Check namespace exists
kubectl get namespace gauth-staging

# If missing, create it:
kubectl apply -f deployments/k8s/staging/namespace.yaml

# Verify required components
kubectl get pods -n ingress-nginx    # NGINX Ingress Controller
kubectl get pods -n cert-manager      # cert-manager for TLS
kubectl get pods -n kube-system | grep metrics-server  # metrics-server
```

### 🐳 Docker Registry Access

Test Docker registry authentication:

```bash
# For GitHub Container Registry (GHCR)
echo $GITHUB_TOKEN | docker login ghcr.io -u mauriciomferz --password-stdin

# Expected output:
# Login Succeeded

# Test push access (optional)
docker pull alpine:latest
docker tag alpine:latest ghcr.io/mauriciomferz/test:latest
docker push ghcr.io/mauriciomferz/test:latest
docker rmi ghcr.io/mauriciomferz/test:latest
```

### 📢 Slack Notifications

Test Slack webhook:

```bash
# Send test message
curl -X POST -H 'Content-type: application/json' \
  --data '{"text":"🚀 GAuth CI/CD Pipeline Test - Ready for deployment!"}' \
  https://hooks.slack.com/services/YOUR/WEBHOOK/URL

# Expected: HTTP 200 OK and message appears in Slack channel
```

---

## 2. Push to GitHub

### Step 1: Final Pre-Push Validation

Run quick validation script:

```bash
./scripts/validate-cicd.sh
```

**Expected output**:
```
✅ [1/4] Workflow YAML syntax valid
✅ [2/4] All required files present
✅ [3/4] Git repository status OK
✅ [4/4] Setup instructions displayed
```

### Step 2: Push Commits

```bash
# Push to main branch (triggers workflow)
git push origin main

# Expected output:
# Enumerating objects: XX, done.
# Counting objects: 100% (XX/XX), done.
# Delta compression using up to 12 threads
# Compressing objects: 100% (XX/XX), done.
# Writing objects: 100% (XX/XX), XX.XX KiB | XX.XX MiB/s, done.
# Total XX (delta XX), reused XX (delta XX), pack-reused 0
# remote: Resolving deltas: 100% (XX/XX), completed with XX local objects.
# To https://github.com/mauriciomferz/Gauth_go.git
#    791fb792..05721f73  main -> main
```

### Step 3: Verify Workflow Triggered

Immediately after pushing, navigate to GitHub Actions:

**URL**: https://github.com/mauriciomferz/Gauth_go/actions

**Look for**:
- Workflow run name: "Deploy to Staging"
- Trigger: Push to main
- Commit: 05721f73 (or latest commit hash)
- Status: 🟡 In Progress (yellow)

---

## 3. Pipeline Monitoring

### Workflow Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Workflow Trigger                         │
│  Event: push to main branch                                 │
│  Commit: 05721f73                                           │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│  Job 1: test (~4 minutes)                                   │
│  ├─ Checkout code                                           │
│  ├─ Set up Go 1.25.4                                        │
│  ├─ Cache Go modules                                        │
│  ├─ Download dependencies                                   │
│  ├─ Run unit tests (go test -race -coverprofile)           │
│  ├─ Run RFC compliance tests                               │
│  ├─ Run security regression tests                          │
│  └─ Upload coverage to Codecov                             │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│  Job 2: security (~2 minutes)                               │
│  ├─ Checkout code                                           │
│  ├─ Set up Go 1.25.4                                        │
│  ├─ Run gosec SAST scan                                     │
│  ├─ Run govulncheck CVE scan                               │
│  └─ Check for known vulnerabilities                        │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│  Job 3: build (~3 minutes)                                  │
│  ├─ Checkout code                                           │
│  ├─ Set up Docker Buildx                                    │
│  ├─ Login to Docker registry (GHCR)                        │
│  ├─ Extract metadata (tags, labels)                        │
│  ├─ Build Docker image (multi-stage)                       │
│  ├─ Run Trivy vulnerability scan                           │
│  └─ Push image to registry                                 │
│      Image: ghcr.io/mauriciomferz/gauth:staging            │
│             ghcr.io/mauriciomferz/gauth:main-05721f73       │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│  Job 4: deploy (~6 minutes)                                 │
│  ├─ Checkout code                                           │
│  ├─ Set up kubectl                                          │
│  ├─ Configure kubeconfig (from secret)                     │
│  ├─ Apply Kubernetes manifests (9 files)                   │
│  │   ├─ namespace.yaml                                      │
│  │   ├─ configmap.yaml                                      │
│  │   ├─ secrets.yaml                                        │
│  │   ├─ postgres-statefulset.yaml                          │
│  │   ├─ redis-statefulset.yaml                             │
│  │   ├─ deployment.yaml                                     │
│  │   ├─ service.yaml                                        │
│  │   ├─ ingress.yaml                                        │
│  │   └─ hpa.yaml                                           │
│  ├─ Wait for rollout (timeout: 5m)                         │
│  ├─ Run smoke tests                                        │
│  │   ├─ curl https://gauth-staging.yourdomain.com/healthz  │
│  │   ├─ curl .../api/v1/beta/health                        │
│  │   └─ curl .../metrics                                   │
│  └─ Send Slack notification (success)                      │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│  Job 5: rollback (only if deploy fails)                    │
│  ├─ Set up kubectl                                          │
│  ├─ Configure kubeconfig                                    │
│  ├─ kubectl rollout undo deployment/gauth-deployment       │
│  └─ Send Slack notification (rollback)                     │
└─────────────────────────────────────────────────────────────┘
```

### Expected Timeline

| Time | Job | Status | Details |
|------|-----|--------|---------|
| 0:00 | Workflow | 🟡 Queued | Waiting for runner |
| 0:15 | test | 🟡 Running | Starting Go tests |
| 1:00 | test | 🟡 Running | Unit tests (2500+ tests) |
| 3:00 | test | 🟡 Running | RFC compliance tests |
| 3:30 | test | 🟡 Running | Security regression tests |
| 4:00 | test | ✅ Complete | Coverage uploaded to Codecov |
| 4:00 | security | 🟡 Running | gosec SAST scan |
| 5:00 | security | 🟡 Running | govulncheck CVE scan |
| 6:00 | security | ✅ Complete | No critical vulnerabilities |
| 6:00 | build | 🟡 Running | Docker buildx setup |
| 7:00 | build | 🟡 Running | Building multi-stage image |
| 8:30 | build | 🟡 Running | Trivy vulnerability scan |
| 9:00 | build | ✅ Complete | Image pushed to GHCR |
| 9:00 | deploy | 🟡 Running | Applying Kubernetes manifests |
| 10:00 | deploy | 🟡 Running | Waiting for rollout |
| 14:00 | deploy | 🟡 Running | Running smoke tests |
| 15:00 | deploy | ✅ Complete | Deployment successful |
| 15:00 | Workflow | ✅ Complete | All jobs passed |

**Total Duration**: ~15 minutes (can vary based on runner availability)

### Monitoring Commands

While pipeline is running, monitor from terminal:

```bash
# Watch workflow status (requires gh CLI)
gh run watch

# List recent workflow runs
gh run list --limit 5

# View workflow logs (replace RUN_ID)
gh run view RUN_ID --log

# Check specific job logs
gh run view RUN_ID --log --job test
gh run view RUN_ID --log --job security
gh run view RUN_ID --log --job build
gh run view RUN_ID --log --job deploy
```

### Kubernetes Monitoring

Monitor deployment from terminal:

```bash
# Watch pods in staging namespace
kubectl get pods -n gauth-staging --watch

# Expected progression:
# NAME                                READY   STATUS              RESTARTS   AGE
# gauth-postgres-0                    0/1     ContainerCreating   0          10s
# gauth-postgres-0                    1/1     Running             0          30s
# gauth-redis-0                       0/1     ContainerCreating   0          10s
# gauth-redis-0                       1/1     Running             0          25s
# gauth-deployment-7d5f4c8b9d-abc12   0/1     ContainerCreating   0          5s
# gauth-deployment-7d5f4c8b9d-abc12   0/1     Running             0          15s
# gauth-deployment-7d5f4c8b9d-abc12   1/1     Running             0          30s
# (repeat for 3 replicas)

# Check rollout status
kubectl rollout status deployment/gauth-deployment -n gauth-staging

# Expected output:
# Waiting for deployment "gauth-deployment" rollout to finish: 0 of 3 updated replicas are available...
# Waiting for deployment "gauth-deployment" rollout to finish: 1 of 3 updated replicas are available...
# Waiting for deployment "gauth-deployment" rollout to finish: 2 of 3 updated replicas are available...
# deployment "gauth-deployment" successfully rolled out

# Check logs from new pods
kubectl logs -f deployment/gauth-deployment -n gauth-staging

# Monitor HPA
kubectl get hpa -n gauth-staging --watch
```

---

## 4. Pipeline Job Details

### Job 1: Test (Duration: ~4 minutes)

**Purpose**: Validate code quality, run tests, measure coverage

**Steps**:
1. **Checkout code**: Clone repository at commit 05721f73
2. **Set up Go 1.25.4**: Install Go runtime
3. **Cache Go modules**: Restore Go module cache for faster builds
4. **Download dependencies**: `go mod download`
5. **Run unit tests**: `go test -race -coverprofile=coverage.out ./...`
   - Expected: 2500+ tests pass
   - Expected: >80% coverage
6. **Run RFC compliance tests**: `go test -tags=compliance ./test/conformance/...`
   - Expected: All RFC 2104, RFC 5869, RFC 6238, RFC 6979, RFC 8032, RFC 8235, GAuth-RFC-001 (formerly RFC 111), RFC 7519 tests pass
7. **Run security regression tests**: `go test -tags=security ./test/security/...`
   - Expected: All SSRF, timing attack, injection tests pass
8. **Upload coverage**: Upload coverage.out to Codecov (optional)

**Success Criteria**:
- ✅ All tests pass (0 failures)
- ✅ Coverage ≥80%
- ✅ No race conditions detected
- ✅ RFC compliance validated
- ✅ Security regressions prevented

**Logs to Check**:
```bash
gh run view RUN_ID --log --job test | grep -E "(PASS|FAIL|coverage:)"
```

**Expected Output**:
```
=== RUN   TestPOASignature
=== RUN   TestPOASignature/RFC2104_HMAC-SHA256
--- PASS: TestPOASignature (0.05s)
    --- PASS: TestPOASignature/RFC2104_HMAC-SHA256 (0.01s)
...
PASS
coverage: 87.3% of statements
ok      github.com/mauriciomferz/Gauth_go/pkg/poa       5.123s  coverage: 87.3% of statements
```

---

### Job 2: Security (Duration: ~2 minutes)

**Purpose**: Detect security vulnerabilities in code and dependencies

**Steps**:
1. **Checkout code**: Clone repository
2. **Set up Go 1.25.4**: Install Go runtime
3. **Run gosec SAST scan**: Static application security testing
   - Command: `gosec -fmt=json -out=gosec-results.json ./...`
   - Detects: Hardcoded credentials, SQL injection, command injection, path traversal
4. **Run govulncheck CVE scan**: Check for known vulnerabilities in dependencies
   - Command: `govulncheck ./...`
   - Checks: Go vulnerability database at https://vuln.go.dev

**Success Criteria**:
- ✅ No HIGH or CRITICAL severity issues from gosec
- ✅ No known CVEs in dependencies
- ⚠️ Warnings allowed (manual review required)

**Logs to Check**:
```bash
gh run view RUN_ID --log --job security | grep -E "(HIGH|CRITICAL|vulnerability)"
```

**Expected Output**:
```
Running gosec security scanner...
[gosec] 2025/11/09 12:00:00 Analyzing...
[gosec] 2025/11/09 12:00:05 Scan completed
[gosec] Results: 0 HIGH, 0 MEDIUM, 2 LOW issues found

Running govulncheck...
govulncheck is checking for known vulnerabilities...
No vulnerabilities found.
```

**Failure Scenarios**:
- ❌ **HIGH/CRITICAL gosec issue**: Workflow fails, manual fix required
- ❌ **Known CVE in dependency**: Workflow fails, dependency upgrade required
- ⚠️ **MEDIUM/LOW gosec issue**: Warning only, workflow continues

---

### Job 3: Build (Duration: ~3 minutes)

**Purpose**: Build Docker image, scan for vulnerabilities, push to registry

**Steps**:
1. **Checkout code**: Clone repository
2. **Set up Docker Buildx**: Enable multi-platform builds
3. **Login to Docker registry**: Authenticate with GHCR
   - Registry: `ghcr.io`
   - Username: `mauriciomferz`
   - Password: `${{ secrets.DOCKER_PASSWORD }}`
4. **Extract metadata**: Generate tags and labels
   - Tags:
     * `ghcr.io/mauriciomferz/gauth:staging` (latest staging)
     * `ghcr.io/mauriciomferz/gauth:main-05721f73` (commit-specific)
   - Labels:
     * `org.opencontainers.image.source=https://github.com/mauriciomferz/Gauth_go`
     * `org.opencontainers.image.revision=05721f73`
     * `org.opencontainers.image.created=2025-11-09T12:00:00Z`
5. **Build Docker image**: Multi-stage build using `Dockerfile`
   - Stage 1: Build (golang:1.25.4-alpine)
   - Stage 2: Runtime (alpine:3.20)
   - Output: ~50MB image
6. **Run Trivy vulnerability scan**: Scan Docker image for OS and library vulnerabilities
   - Command: `trivy image --severity HIGH,CRITICAL ghcr.io/mauriciomferz/gauth:staging`
   - Checks: Alpine base image, Go runtime, application dependencies
7. **Push image to registry**: Upload to GHCR

**Success Criteria**:
- ✅ Docker build succeeds
- ✅ Image size <100MB
- ✅ No HIGH or CRITICAL vulnerabilities in Trivy scan
- ✅ Image pushed successfully

**Logs to Check**:
```bash
gh run view RUN_ID --log --job build | grep -E "(Step [0-9]+|pushed|vulnerability)"
```

**Expected Output**:
```
#1 [internal] load .dockerignore
#1 transferring context: 2B done
#2 [internal] load build definition from Dockerfile
#2 transferring dockerfile: 1.2kB done
...
#15 [stage-1 5/5] RUN addgroup -g 1000 gauth && adduser -D -u 1000 -G gauth gauth
#15 DONE 0.3s
...
#17 exporting to image
#17 exporting layers done
#17 writing image sha256:abc123def456... done
#17 naming to ghcr.io/mauriciomferz/gauth:staging done
#17 DONE 0.5s

Running Trivy vulnerability scanner...
2025-11-09T12:05:00.000Z        INFO    Vulnerability scanning is enabled
2025-11-09T12:05:05.000Z        INFO    Detected OS: alpine 3.20
2025-11-09T12:05:05.000Z        INFO    Number of language-specific files: 1
2025-11-09T12:05:10.000Z        INFO    Detected vulnerabilities: 0 CRITICAL, 0 HIGH, 2 MEDIUM, 5 LOW

Pushing image to ghcr.io...
staging: digest: sha256:abc123def456... size: 1234
main-05721f73: digest: sha256:abc123def456... size: 1234
```

**Image Details**:
- **Registry**: `ghcr.io/mauriciomferz/gauth`
- **Tags**: `staging`, `main-05721f73`
- **Size**: ~50MB (compressed)
- **Layers**: 5 (base alpine + binary + user + config + entrypoint)
- **Architecture**: linux/amd64 (can be extended to arm64)

---

### Job 4: Deploy (Duration: ~6 minutes)

**Purpose**: Deploy application to Kubernetes staging environment

**Steps**:
1. **Checkout code**: Clone repository
2. **Set up kubectl**: Install kubectl CLI
3. **Configure kubeconfig**: Decode `KUBE_CONFIG_STAGING` secret
   - Command: `echo "${{ secrets.KUBE_CONFIG_STAGING }}" | base64 -d > kubeconfig`
   - File: `~/.kube/config`
4. **Apply Kubernetes manifests**: Deploy 9 YAML files
   ```bash
   kubectl apply -f deployments/k8s/staging/namespace.yaml
   kubectl apply -f deployments/k8s/staging/configmap.yaml
   kubectl apply -f deployments/k8s/staging/secrets.yaml
   kubectl apply -f deployments/k8s/staging/postgres-statefulset.yaml
   kubectl apply -f deployments/k8s/staging/redis-statefulset.yaml
   kubectl apply -f deployments/k8s/staging/deployment.yaml
   kubectl apply -f deployments/k8s/staging/service.yaml
   kubectl apply -f deployments/k8s/staging/ingress.yaml
   kubectl apply -f deployments/k8s/staging/hpa.yaml
   ```
5. **Wait for rollout**: Monitor deployment progress (timeout: 5m)
   - Command: `kubectl rollout status deployment/gauth-deployment -n gauth-staging --timeout=5m`
   - Expected: 3 replicas running and healthy
6. **Run smoke tests**: Verify endpoints
   ```bash
   curl -f https://gauth-staging.yourdomain.com/healthz
   curl -f https://gauth-staging.yourdomain.com/api/v1/beta/health
   curl -f https://gauth-staging.yourdomain.com/metrics | grep gauth_
   ```
7. **Send Slack notification**: Post success message to Slack

**Success Criteria**:
- ✅ All manifests applied successfully
- ✅ Rollout completes within 5 minutes
- ✅ All 3 replicas running (1/1 Ready)
- ✅ PostgreSQL StatefulSet ready (1/1)
- ✅ Redis StatefulSet ready (1/1)
- ✅ Smoke tests pass (HTTP 200)
- ✅ Slack notification sent

**Logs to Check**:
```bash
gh run view RUN_ID --log --job deploy | grep -E "(configured|created|deployed|rollout)"
```

**Expected Output**:
```
namespace/gauth-staging configured
configmap/gauth-config configured
secret/gauth-secrets configured
statefulset.apps/gauth-postgres configured
statefulset.apps/gauth-redis configured
deployment.apps/gauth-deployment configured
service/gauth-service configured
ingress.networking.k8s.io/gauth-ingress configured
horizontalpodautoscaler.autoscaling/gauth-hpa configured

Waiting for deployment "gauth-deployment" rollout to finish: 0 of 3 updated replicas are available...
Waiting for deployment "gauth-deployment" rollout to finish: 1 of 3 updated replicas are available...
Waiting for deployment "gauth-deployment" rollout to finish: 2 of 3 updated replicas are available...
deployment "gauth-deployment" successfully rolled out

Running smoke tests...
✅ /healthz: HTTP 200
✅ /api/v1/beta/health: HTTP 200
✅ /metrics: HTTP 200 (gauth_requests_total found)

🎉 Deployment successful! Sending Slack notification...
```

**Kubernetes Resources Created/Updated**:

| Resource Type | Name | Replicas/Ready | CPU Limit | Memory Limit |
|---------------|------|----------------|-----------|--------------|
| Namespace | gauth-staging | - | - | - |
| ConfigMap | gauth-config | - | - | - |
| Secret | gauth-secrets | - | - | - |
| StatefulSet | gauth-postgres | 1/1 | 2000m | 4Gi |
| StatefulSet | gauth-redis | 1/1 | 1000m | 2Gi |
| Deployment | gauth-deployment | 3/3 | 2000m | 4Gi |
| Service | gauth-service | - | - | - |
| Ingress | gauth-ingress | - | - | - |
| HPA | gauth-hpa | min=3, max=10 | - | - |

**Total Resources**:
- **Pods**: 5 (3 GAuth + 1 PostgreSQL + 1 Redis)
- **CPU**: 11 cores (requested: 5.5 cores)
- **Memory**: 22Gi (requested: 11Gi)
- **Storage**: 25Gi (20Gi PostgreSQL + 5Gi Redis)

---

### Job 5: Rollback (Only if Deploy Fails)

**Purpose**: Automatically rollback to previous working version if deployment fails

**Trigger Conditions**:
- ❌ Deployment manifest apply fails
- ❌ Rollout timeout (>5 minutes)
- ❌ Smoke tests fail (HTTP 4xx/5xx)

**Steps**:
1. **Set up kubectl**: Install kubectl CLI
2. **Configure kubeconfig**: Decode secret
3. **Rollback deployment**: Undo to previous revision
   ```bash
   kubectl rollout undo deployment/gauth-deployment -n gauth-staging
   ```
4. **Wait for rollback**: Monitor rollout status
   ```bash
   kubectl rollout status deployment/gauth-deployment -n gauth-staging --timeout=3m
   ```
5. **Send Slack notification**: Alert team of rollback

**Success Criteria**:
- ✅ Rollback completes successfully
- ✅ Previous version running and healthy
- ✅ Slack notification sent

**Expected Output** (only if triggered):
```
⚠️  Deployment failed, initiating rollback...

Rollback to previous revision...
deployment.apps/gauth-deployment rolled back

Waiting for rollback to complete...
deployment "gauth-deployment" successfully rolled out

🔄 Rollback successful! Previous version restored.
Sending Slack notification...
```

**Slack Notification** (failure + rollback):
```
❌ GAuth Deployment Failed
Environment: staging
Commit: 05721f73
Reason: Deployment timeout after 5 minutes
Action: Automatically rolled back to previous version
Status: ✅ Rollback successful
View logs: https://github.com/mauriciomferz/Gauth_go/actions/runs/XXX
```

---

## 5. Slack Notifications

### Notification Types

#### 1. Deployment Started
```
🚀 GAuth Deployment Started
Environment: staging
Branch: main
Commit: 05721f73 - docs: Add Week 4 Day 3 CI/CD setup
Triggered by: mauriciomferz
View: https://github.com/mauriciomferz/Gauth_go/actions/runs/XXX
```

#### 2. Deployment Success
```
✅ GAuth Deployment Successful
Environment: staging
Commit: 05721f73
Duration: 15m 32s
Image: ghcr.io/mauriciomferz/gauth:staging
Endpoints:
  • Health: https://gauth-staging.yourdomain.com/healthz
  • Beta API: https://gauth-staging.yourdomain.com/api/v1/beta/health
  • Metrics: https://gauth-staging.yourdomain.com/metrics
Resources: 5 pods, 3 replicas
View: https://github.com/mauriciomferz/Gauth_go/actions/runs/XXX
```

#### 3. Deployment Failure
```
❌ GAuth Deployment Failed
Environment: staging
Commit: 05721f73
Duration: 8m 14s
Failed Job: deploy
Reason: Deployment rollout timeout after 5 minutes
Action: Automatically rolled back to previous version
View: https://github.com/mauriciomferz/Gauth_go/actions/runs/XXX
```

#### 4. Security Scan Failure
```
🔒 Security Scan Failed
Environment: staging
Commit: 05721f73
Failed Job: security
Issues Found:
  • HIGH: SQL injection vulnerability in pkg/auth/handler.go:45
  • CRITICAL: CVE-2024-1234 in golang.org/x/crypto v0.1.0
Action: Deployment blocked, manual fix required
View: https://github.com/mauriciomferz/Gauth_go/actions/runs/XXX
```

---

## 6. Troubleshooting Common Issues

### Issue 1: Workflow Not Triggered

**Symptoms**:
- No workflow appears at https://github.com/mauriciomferz/Gauth_go/actions after push
- Workflow shows "This workflow has a workflow_dispatch event trigger"

**Causes**:
- Workflow file not on main branch
- Workflow YAML syntax error
- GitHub Actions disabled for repository

**Solutions**:
```bash
# Verify workflow file exists
git ls-tree -r main --name-only | grep workflow

# Validate YAML syntax
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/deploy-staging.yml'))"

# Check GitHub Actions settings
# Navigate to: https://github.com/mauriciomferz/Gauth_go/settings/actions
# Ensure "Allow all actions and reusable workflows" is enabled
```

---

### Issue 2: Docker Login Failed

**Symptoms**:
- Build job fails with "unauthorized: authentication required"
- Error: "denied: permission_denied: write_package"

**Causes**:
- Invalid `DOCKER_PASSWORD` secret (GitHub PAT expired or wrong scope)
- `DOCKER_USERNAME` doesn't match repository owner
- Docker registry URL incorrect

**Solutions**:
```bash
# Test Docker login locally
echo $GITHUB_TOKEN | docker login ghcr.io -u mauriciomferz --password-stdin

# Verify GitHub PAT has correct scopes
# Navigate to: https://github.com/settings/tokens
# Required scopes: write:packages, delete:packages, read:packages

# Regenerate GitHub PAT if needed
# Update secret at: https://github.com/mauriciomferz/Gauth_go/settings/secrets/actions
```

---

### Issue 3: Kubernetes Connection Failed

**Symptoms**:
- Deploy job fails with "Unable to connect to the server"
- Error: "error: You must be logged in to the server (Unauthorized)"

**Causes**:
- Invalid `KUBE_CONFIG_STAGING` secret
- kubeconfig expired or revoked
- Cluster endpoint changed

**Solutions**:
```bash
# Test kubectl locally
kubectl cluster-info
kubectl get nodes

# Regenerate kubeconfig secret
cat ~/.kube/config | base64 | pbcopy
# Update secret at: https://github.com/mauriciomferz/Gauth_go/settings/secrets/actions
# Paste base64-encoded kubeconfig

# Verify kubeconfig has correct cluster endpoint
kubectl config view --minify
```

---

### Issue 4: Deployment Rollout Timeout

**Symptoms**:
- Deploy job fails after 5 minutes
- Error: "error: timed out waiting for the condition"
- Pods stuck in `ContainerCreating` or `ImagePullBackOff`

**Causes**:
- ImagePullBackOff: Cannot pull image from registry
- CrashLoopBackOff: Application crashes on startup
- Insufficient resources: Cluster doesn't have enough CPU/memory

**Solutions**:
```bash
# Check pod status
kubectl get pods -n gauth-staging
kubectl describe pod <POD_NAME> -n gauth-staging

# Check events
kubectl get events -n gauth-staging --sort-by='.lastTimestamp'

# Check logs
kubectl logs <POD_NAME> -n gauth-staging

# Common fixes:

# 1. ImagePullBackOff - Create ImagePullSecret
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=mauriciomferz \
  --docker-password=$GITHUB_TOKEN \
  --namespace=gauth-staging

# Update deployment to use imagePullSecrets
kubectl patch deployment gauth-deployment -n gauth-staging -p '
{
  "spec": {
    "template": {
      "spec": {
        "imagePullSecrets": [{"name": "ghcr-secret"}]
      }
    }
  }
}'

# 2. CrashLoopBackOff - Check application logs
kubectl logs <POD_NAME> -n gauth-staging --previous

# 3. Insufficient resources - Scale down or increase cluster capacity
kubectl top nodes
kubectl describe node <NODE_NAME>
```

---

### Issue 5: Smoke Tests Failed

**Symptoms**:
- Deploy job fails after rollout succeeds
- Error: "curl: (22) The requested URL returned error: 404"
- Endpoints return HTTP 5xx errors

**Causes**:
- Ingress not configured correctly
- Service selector doesn't match pod labels
- Application not listening on expected port
- TLS certificate not ready

**Solutions**:
```bash
# Check ingress
kubectl get ingress -n gauth-staging
kubectl describe ingress gauth-ingress -n gauth-staging

# Check service
kubectl get svc -n gauth-staging
kubectl describe svc gauth-service -n gauth-staging

# Check endpoints
kubectl get endpoints gauth-service -n gauth-staging

# Test pod directly (bypassing ingress)
kubectl port-forward -n gauth-staging deployment/gauth-deployment 8080:8080
curl http://localhost:8080/healthz

# Check ingress controller logs
kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller

# Check TLS certificate
kubectl get certificate -n gauth-staging
kubectl describe certificate gauth-staging-tls -n gauth-staging
```

---

### Issue 6: Trivy Scan Failed

**Symptoms**:
- Build job fails with "vulnerabilities found"
- Error: "CRITICAL: CVE-2024-XXXX found in package Y"

**Causes**:
- Known vulnerability in Alpine base image
- Vulnerable Go dependency
- Outdated system packages

**Solutions**:
```bash
# Update Alpine base image in Dockerfile
sed -i '' 's/alpine:3.19/alpine:3.20/g' Dockerfile

# Update Go dependencies
go get -u all
go mod tidy

# Test Trivy scan locally
docker build -t gauth:test .
trivy image --severity HIGH,CRITICAL gauth:test

# If vulnerability is false positive, add to .trivyignore
echo "CVE-2024-XXXX" >> .trivyignore
```

---

### Issue 7: Slack Notifications Not Received

**Symptoms**:
- Pipeline succeeds but no Slack notification
- Workflow logs show "Failed to send Slack notification"

**Causes**:
- Invalid `SLACK_WEBHOOK_URL` secret
- Slack webhook revoked or expired
- Network connectivity issues

**Solutions**:
```bash
# Test webhook locally
curl -X POST -H 'Content-type: application/json' \
  --data '{"text":"🚀 GAuth CI/CD Pipeline Test"}' \
  https://hooks.slack.com/services/YOUR/WEBHOOK/URL

# Expected response: ok
# Status code: 200

# Regenerate Slack webhook if needed
# Navigate to: https://api.slack.com/apps
# Select app → Incoming Webhooks → Add New Webhook to Workspace
# Update secret at: https://github.com/mauriciomferz/Gauth_go/settings/secrets/actions
```

---

## 7. Post-Deployment Verification

### Step 1: Verify All Pods Running

```bash
kubectl get pods -n gauth-staging

# Expected output:
# NAME                                READY   STATUS    RESTARTS   AGE
# gauth-deployment-7d5f4c8b9d-abc12   1/1     Running   0          5m
# gauth-deployment-7d5f4c8b9d-def34   1/1     Running   0          5m
# gauth-deployment-7d5f4c8b9d-ghi56   1/1     Running   0          5m
# gauth-postgres-0                    1/1     Running   0          6m
# gauth-redis-0                       1/1     Running   0          6m
```

### Step 2: Check Service Endpoints

```bash
kubectl get svc -n gauth-staging

# Expected output:
# NAME              TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
# gauth-service     ClusterIP   10.96.123.45    <none>        80/TCP     5m
# gauth-postgres    ClusterIP   10.96.123.46    <none>        5432/TCP   6m
# gauth-redis       ClusterIP   10.96.123.47    <none>        6379/TCP   6m
```

### Step 3: Verify Ingress

```bash
kubectl get ingress -n gauth-staging

# Expected output:
# NAME            CLASS   HOSTS                           ADDRESS         PORTS     AGE
# gauth-ingress   nginx   gauth-staging.yourdomain.com    34.123.45.67    80, 443   5m
```

### Step 4: Test Endpoints

```bash
# Health check
curl -v https://gauth-staging.yourdomain.com/healthz
# Expected: HTTP 200, {"status":"ok"}

# Beta health
curl -v https://gauth-staging.yourdomain.com/api/v1/beta/health
# Expected: HTTP 200, {"status":"healthy","version":"beta"}

# Metrics
curl -v https://gauth-staging.yourdomain.com/metrics | head -20
# Expected: HTTP 200, Prometheus metrics (gauth_requests_total, etc.)
```

### Step 5: Check Application Logs

```bash
# Tail logs from all replicas
kubectl logs -f -n gauth-staging -l app=gauth --tail=100

# Expected output:
# 2025-11-09T12:15:00Z INFO Starting GAuth server version=beta port=8080
# 2025-11-09T12:15:00Z INFO Database connection established host=gauth-postgres
# 2025-11-09T12:15:00Z INFO Redis connection established host=gauth-redis
# 2025-11-09T12:15:00Z INFO Server listening on :8080
```

### Step 6: Verify HPA

```bash
kubectl get hpa -n gauth-staging

# Expected output:
# NAME        REFERENCE                    TARGETS   MINPODS   MAXPODS   REPLICAS   AGE
# gauth-hpa   Deployment/gauth-deployment  25%/80%   3         10        3          5m
```

### Step 7: Check Resource Usage

```bash
kubectl top pods -n gauth-staging

# Expected output:
# NAME                                CPU(cores)   MEMORY(bytes)
# gauth-deployment-7d5f4c8b9d-abc12   150m         512Mi
# gauth-deployment-7d5f4c8b9d-def34   145m         508Mi
# gauth-deployment-7d5f4c8b9d-ghi56   148m         515Mi
# gauth-postgres-0                    80m          1Gi
# gauth-redis-0                       30m          256Mi
```

---

## 8. Next Steps

### ✅ Week 4 Day 4 Complete

After successful pipeline execution and verification:

1. **Tag commit** (optional):
   ```bash
   git tag -a week4-day4-complete -m "Week 4 Day 4: CI/CD pipeline execution successful"
   git push origin week4-day4-complete
   ```

2. **Document results**: Create `artifacts/preproduction_audit_week4_day4.md`

3. **Proceed to Week 4 Day 5**: Blue-Green Deployment Validation
   - Deploy to green environment
   - Test traffic switching
   - Verify zero-downtime
   - Test instant rollback

---

## 9. Summary Checklist

### Pre-Push
- [ ] Local validation passed (`./scripts/validate-cicd.sh`)
- [ ] GitHub secrets configured (6 required)
- [ ] Kubernetes cluster accessible
- [ ] Docker registry accessible
- [ ] Slack webhook tested

### Push and Monitor
- [ ] Code pushed to GitHub (`git push origin main`)
- [ ] Workflow triggered at https://github.com/mauriciomferz/Gauth_go/actions
- [ ] Test job passed (~4 min)
- [ ] Security job passed (~2 min)
- [ ] Build job passed (~3 min)
- [ ] Deploy job passed (~6 min)
- [ ] Slack notification received

### Post-Deployment
- [ ] All pods running (5 pods)
- [ ] Services accessible (3 services)
- [ ] Ingress configured (1 ingress)
- [ ] Endpoints responding (3 endpoints tested)
- [ ] Logs healthy (no errors)
- [ ] HPA configured (min=3, max=10)
- [ ] Resources within limits

### Documentation
- [ ] Week 4 Day 4 report created
- [ ] Screenshots captured (workflow, pods, services)
- [ ] Issues documented (if any)
- [ ] Commit tagged (optional)

---

## Appendix A: GitHub Actions Workflow Summary

**File**: `.github/workflows/deploy-staging.yml`  
**Lines**: 390  
**Jobs**: 5 (test, security, build, deploy, rollback)  
**Estimated Duration**: 15-20 minutes  
**Triggers**: Push to main, manual workflow_dispatch

**Key Features**:
- ✅ Comprehensive testing (unit, RFC compliance, security)
- ✅ SAST and CVE scanning (gosec, govulncheck)
- ✅ Container security (Trivy)
- ✅ Automated deployment with smoke tests
- ✅ Automatic rollback on failure
- ✅ Slack notifications for all outcomes
- ✅ Coverage reporting (Codecov)

**Secrets Required**: 6 (5 mandatory + 1 optional)

---

## Appendix B: Kubernetes Resources Summary

**Namespace**: `gauth-staging`

| Resource | Count | Total CPU | Total Memory | Total Storage |
|----------|-------|-----------|--------------|---------------|
| Deployments | 1 | 6000m (3 pods × 2000m) | 12Gi (3 pods × 4Gi) | - |
| StatefulSets | 2 | 3000m (2000m + 1000m) | 6Gi (4Gi + 2Gi) | 25Gi |
| Services | 3 | - | - | - |
| Ingress | 1 | - | - | - |
| HPA | 1 | - | - | - |
| ConfigMaps | 1 | - | - | - |
| Secrets | 1 | - | - | - |

**Total Resources**:
- **Pods**: 5 (3 GAuth + 1 PostgreSQL + 1 Redis)
- **CPU**: 11000m (11 cores requested)
- **Memory**: 22Gi (18Gi requested)
- **Storage**: 25Gi (20Gi PostgreSQL + 5Gi Redis)

**Cluster Requirements**:
- Kubernetes 1.28+
- 3+ worker nodes (HA)
- 16+ cores total
- 32Gi+ RAM total
- 50Gi+ storage

---

**Document Version**: 1.0  
**Created**: November 9, 2025  
**Author**: GitHub Copilot (GAuth Pre-Production Team)  
**Status**: Ready for execution
