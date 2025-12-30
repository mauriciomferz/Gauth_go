---
title: GitHub Actions CI/CD Setup Guide
category: cicd-guide
status: active
lastUpdated: 2025-11-12
owners: platform-eng
source: internal
refreshCadence: quarterly
---

# GitHub Actions CI/CD Setup Guide

This guide walks through setting up the GitHub Actions CI/CD pipeline for AgentAuth staging deployment.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [GitHub Repository Secrets](#github-repository-secrets)
3. [Docker Registry Setup](#docker-registry-setup)
4. [Kubernetes Cluster Setup](#kubernetes-cluster-setup)
5. [Slack Notifications Setup](#slack-notifications-setup)
6. [Testing the Pipeline](#testing-the-pipeline)
7. [Troubleshooting](#troubleshooting)

---

## 1. Prerequisites

**Required Tools**:
- Docker (for building and pushing images)
- kubectl (for Kubernetes cluster access)
- Access to a Kubernetes cluster (staging environment)
- GitHub repository admin access (to configure secrets)

**Required Accounts**:
- Docker registry account (GitHub Container Registry, AWS ECR, or Google GCR)
- Kubernetes cluster (AWS EKS, GCP GKE, Azure AKS, or self-managed)
- Slack workspace (for notifications)

---

## 2. GitHub Repository Secrets

Navigate to your GitHub repository:
1. Go to **Settings** → **Secrets and variables** → **Actions**
2. Click **New repository secret** for each of the following:

### Required Secrets

#### `DOCKER_REGISTRY`
**Purpose**: Docker registry URL where images will be pushed  
**Examples**:
- GitHub Container Registry: `ghcr.io`
- AWS ECR: `123456789012.dkr.ecr.us-east-1.amazonaws.com`
- Google GCR: `gcr.io/your-project-id`
- Docker Hub: `docker.io`

**Value**: `ghcr.io` (or your chosen registry)

---

#### `DOCKER_USERNAME`
**Purpose**: Username for Docker registry authentication  
**Examples**:
- GitHub Container Registry: Your GitHub username (e.g., `mauriciomferz`)
- AWS ECR: `AWS` (literal string)
- Google GCR: `_json_key` (literal string)
- Docker Hub: Your Docker Hub username

**Value**: (depends on registry choice)

---

#### `DOCKER_PASSWORD`
**Purpose**: Password/token for Docker registry authentication  
**Examples**:
- GitHub Container Registry: GitHub Personal Access Token (PAT) with `write:packages` scope
- AWS ECR: AWS Secret Access Key
- Google GCR: Full JSON content of GCP service account key
- Docker Hub: Docker Hub password or access token

**GitHub PAT Creation** (for GHCR):
1. Go to GitHub **Settings** → **Developer settings** → **Personal access tokens** → **Tokens (classic)**
2. Click **Generate new token (classic)**
3. Name: `AgentAuth CI/CD`
4. Scopes: 
   - `write:packages` (upload packages to GitHub Package Registry)
   - `delete:packages` (delete packages from GitHub Package Registry)
   - `read:packages` (download packages from GitHub Package Registry)
5. Click **Generate token**
6. Copy token and save as `DOCKER_PASSWORD` secret

**Value**: (depends on registry choice)

---

#### `KUBE_CONFIG_STAGING`
**Purpose**: Base64-encoded kubeconfig for staging Kubernetes cluster  
**Format**: Base64-encoded YAML

**Generation Steps**:

**Option 1: Full kubeconfig (simple)**:
```bash
# Copy your existing kubeconfig
cat ~/.kube/config | base64 | pbcopy  # macOS
cat ~/.kube/config | base64 -w 0     # Linux
```

**Option 2: Create dedicated kubeconfig (recommended)**:
```bash
# Create a service account for CI/CD
kubectl create serviceaccount gauth-cicd -n gauth-staging

# Create ClusterRoleBinding (or RoleBinding for namespace-scoped)
kubectl create clusterrolebinding gauth-cicd-binding \
  --clusterrole=cluster-admin \
  --serviceaccount=gauth-staging:gauth-cicd

# Get the service account token (Kubernetes 1.24+)
kubectl create token gauth-cicd -n gauth-staging --duration=876000h > /tmp/sa-token

# Get cluster info
CLUSTER_NAME=$(kubectl config view -o jsonpath='{.clusters[0].name}')
CLUSTER_SERVER=$(kubectl config view -o jsonpath='{.clusters[0].cluster.server}')
CLUSTER_CA=$(kubectl config view --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}')

# Create dedicated kubeconfig
cat > /tmp/gauth-cicd-kubeconfig.yaml <<EOF
apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: ${CLUSTER_CA}
    server: ${CLUSTER_SERVER}
  name: ${CLUSTER_NAME}
contexts:
- context:
    cluster: ${CLUSTER_NAME}
    namespace: gauth-staging
    user: gauth-cicd
  name: gauth-cicd-context
current-context: gauth-cicd-context
users:
- name: gauth-cicd
  user:
    token: $(cat /tmp/sa-token)
EOF

# Base64 encode for GitHub secret
cat /tmp/gauth-cicd-kubeconfig.yaml | base64 | pbcopy  # macOS
cat /tmp/gauth-cicd-kubeconfig.yaml | base64 -w 0     # Linux

# Cleanup
rm /tmp/sa-token /tmp/gauth-cicd-kubeconfig.yaml
```

**Value**: Base64-encoded kubeconfig (paste from clipboard)

**Security Note**: Consider using [OIDC federation](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/configuring-openid-connect-in-amazon-web-services) for AWS EKS or [Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) for GCP GKE to avoid storing long-lived credentials.

---

#### `SLACK_WEBHOOK_URL`
**Purpose**: Slack incoming webhook URL for CI/CD notifications  
**Format**: `https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX`

**Creation Steps**:
1. Go to your Slack workspace
2. Navigate to **Apps** → **Incoming Webhooks** (or create one at https://api.slack.com/apps)
3. Click **Add to Slack**
4. Choose channel (e.g., `#gauth-cicd`)
5. Click **Add Incoming WebHooks integration**
6. Copy **Webhook URL**

**Value**: Slack webhook URL (paste from Slack)

---

#### `CODECOV_TOKEN` (Optional)
**Purpose**: Token for uploading test coverage to Codecov  
**Format**: UUID string

**Creation Steps**:
1. Go to https://codecov.io/
2. Sign in with GitHub
3. Add your repository (`mauriciomferz/Gauth_go`)
4. Copy **Repository Upload Token**

**Value**: Codecov token (or leave blank if not using Codecov)

---

## 3. Docker Registry Setup

### Option A: GitHub Container Registry (GHCR) - Recommended

**Advantages**: 
- Free for public repos, included with GitHub
- No additional service setup
- Tight integration with GitHub Actions

**Setup**:
```bash
# Login locally (test)
echo $GITHUB_PAT | docker login ghcr.io -u mauriciomferz --password-stdin

# Tag and push test image
docker build -t ghcr.io/mauriciomferz/gauth:test .
docker push ghcr.io/mauriciomferz/gauth:test

# Verify image
docker pull ghcr.io/mauriciomferz/gauth:test
```

**GitHub Secrets**:
- `DOCKER_REGISTRY`: `ghcr.io`
- `DOCKER_USERNAME`: `mauriciomferz`
- `DOCKER_PASSWORD`: (GitHub PAT with `write:packages`)

**Update Workflow** (if using GHCR):
No changes needed. Workflow is pre-configured for any registry.

---

### Option B: AWS Elastic Container Registry (ECR)

**Advantages**:
- Integrated with AWS ecosystem
- Private by default
- Image scanning included

**Setup**:
```bash
# Install AWS CLI
brew install awscli  # macOS
apt install awscli   # Linux

# Configure AWS credentials
aws configure

# Create ECR repository
aws ecr create-repository --repository-name gauth --region us-east-1

# Get registry URL
aws ecr describe-repositories --repository-names gauth --region us-east-1 \
  --query 'repositories[0].repositoryUri' --output text
# Output: 123456789012.dkr.ecr.us-east-1.amazonaws.com/gauth

# Login locally (test)
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin \
  123456789012.dkr.ecr.us-east-1.amazonaws.com

# Tag and push test image
docker build -t 123456789012.dkr.ecr.us-east-1.amazonaws.com/gauth:test .
docker push 123456789012.dkr.ecr.us-east-1.amazonaws.com/gauth:test
```

**GitHub Secrets**:
- `DOCKER_REGISTRY`: `123456789012.dkr.ecr.us-east-1.amazonaws.com`
- `DOCKER_USERNAME`: `AWS`
- `DOCKER_PASSWORD`: (AWS Secret Access Key)

**IAM Permissions Required**:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecr:GetAuthorizationToken",
        "ecr:BatchCheckLayerAvailability",
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage",
        "ecr:PutImage",
        "ecr:InitiateLayerUpload",
        "ecr:UploadLayerPart",
        "ecr:CompleteLayerUpload"
      ],
      "Resource": "*"
    }
  ]
}
```

---

### Option C: Google Container Registry (GCR)

**Advantages**:
- Integrated with Google Cloud
- Fast in GCP regions
- Vulnerability scanning

**Setup**:
```bash
# Install gcloud CLI
brew install --cask google-cloud-sdk  # macOS

# Authenticate
gcloud auth login

# Set project
gcloud config set project YOUR_PROJECT_ID

# Enable Container Registry API
gcloud services enable containerregistry.googleapis.com

# Create service account
gcloud iam service-accounts create gauth-cicd \
  --display-name "AgentAuth CI/CD" \
  --project YOUR_PROJECT_ID

# Grant permissions
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member="serviceAccount:gauth-cicd@YOUR_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/storage.admin"

# Create and download key
gcloud iam service-accounts keys create ~/gauth-cicd-key.json \
  --iam-account gauth-cicd@YOUR_PROJECT_ID.iam.gserviceaccount.com

# Login locally (test)
cat ~/gauth-cicd-key.json | docker login -u _json_key --password-stdin gcr.io

# Tag and push test image
docker build -t gcr.io/YOUR_PROJECT_ID/gauth:test .
docker push gcr.io/YOUR_PROJECT_ID/gauth:test
```

**GitHub Secrets**:
- `DOCKER_REGISTRY`: `gcr.io/YOUR_PROJECT_ID`
- `DOCKER_USERNAME`: `_json_key`
- `DOCKER_PASSWORD`: (Full JSON content of `~/gauth-cicd-key.json`)

**Security Note**: Store JSON key securely. Consider using [Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) for GKE.

---

## 4. Kubernetes Cluster Setup

### Prerequisites

**Cluster Requirements**:
- Kubernetes 1.28+ (for StatefulSet volumeClaimTemplates)
- NGINX Ingress Controller installed
- cert-manager installed (for TLS certificates)
- metrics-server installed (for HPA)
- StorageClass available (standard, gp3, pd-ssd)

### Install Required Components

#### NGINX Ingress Controller
```bash
# Install via Helm
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update
helm install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx \
  --create-namespace \
  --set controller.service.type=LoadBalancer

# Verify
kubectl get pods -n ingress-nginx
kubectl get svc -n ingress-nginx
```

#### cert-manager (for Let's Encrypt TLS)
```bash
# Install via Helm
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set installCRDs=true

# Verify
kubectl get pods -n cert-manager

# Create ClusterIssuer for Let's Encrypt
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: your-email@example.com  # REPLACE
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
```

#### metrics-server (for HPA)
```bash
# Install via Helm
helm repo add metrics-server https://kubernetes-sigs.github.io/metrics-server/
helm repo update
helm install metrics-server metrics-server/metrics-server \
  --namespace kube-system \
  --set args[0]="--kubelet-insecure-tls"  # Only for development

# Verify
kubectl top nodes
kubectl top pods -A
```

### Create Namespace and Secrets

```bash
# Create namespace
kubectl apply -f deployments/k8s/staging/namespace.yaml

# Generate secrets (IMPORTANT: Replace placeholders)
# JWT Keys (RSA 2048-bit)
openssl genrsa -out /tmp/jwt-private.pem 2048
openssl rsa -in /tmp/jwt-private.pem -pubout -out /tmp/jwt-public.pem

# Ed25519 Keys (for signatures)
openssl genpkey -algorithm ed25519 -out /tmp/ed25519-private.pem
openssl pkey -in /tmp/ed25519-private.pem -pubout -out /tmp/ed25519-public.pem

# Database passwords
POSTGRES_PASSWORD=$(openssl rand -base64 32)
REDIS_PASSWORD=$(openssl rand -base64 32)

# Update secrets.yaml with actual values
sed -i "s|jwt-private-key: .*|jwt-private-key: $(cat /tmp/jwt-private.pem | base64 -w 0)|" deployments/k8s/staging/secrets.yaml
sed -i "s|jwt-public-key: .*|jwt-public-key: $(cat /tmp/jwt-public.pem | base64 -w 0)|" deployments/k8s/staging/secrets.yaml
sed -i "s|ed25519-private-key: .*|ed25519-private-key: $(cat /tmp/ed25519-private.pem | base64 -w 0)|" deployments/k8s/staging/secrets.yaml
sed -i "s|ed25519-public-key: .*|ed25519-public-key: $(cat /tmp/ed25519-public.pem | base64 -w 0)|" deployments/k8s/staging/secrets.yaml
sed -i "s|postgres-password: .*|postgres-password: $(echo -n $POSTGRES_PASSWORD | base64 -w 0)|" deployments/k8s/staging/secrets.yaml
sed -i "s|redis-password: .*|redis-password: $(echo -n $REDIS_PASSWORD | base64 -w 0)|" deployments/k8s/staging/secrets.yaml

# Apply secrets
kubectl apply -f deployments/k8s/staging/secrets.yaml

# Cleanup
rm /tmp/jwt-*.pem /tmp/ed25519-*.pem
```

### Verify Cluster Access

```bash
# Check connectivity
kubectl cluster-info

# Check nodes
kubectl get nodes

# Check storage classes
kubectl get storageclass

# Check namespace
kubectl get namespace gauth-staging
```

---

## 5. Slack Notifications Setup

### Create Slack App

1. Go to https://api.slack.com/apps
2. Click **Create New App** → **From scratch**
3. App Name: `AgentAuth CI/CD`
4. Workspace: Select your workspace
5. Click **Create App**

### Enable Incoming Webhooks

1. In your app settings, go to **Incoming Webhooks**
2. Toggle **Activate Incoming Webhooks** to **On**
3. Click **Add New Webhook to Workspace**
4. Select channel: `#gauth-cicd` (or create new channel)
5. Click **Allow**
6. Copy **Webhook URL** (looks like `https://hooks.slack.com/services/...`)
7. Save as `SLACK_WEBHOOK_URL` GitHub secret

### Test Webhook

```bash
# Test Slack notification
curl -X POST -H 'Content-type: application/json' \
  --data '{
    "text": "Test message from AgentAuth CI/CD",
    "attachments": [{
      "color": "good",
      "fields": [{
        "title": "Status",
        "value": "✅ Test Successful",
        "short": true
      }]
    }]
  }' \
  YOUR_SLACK_WEBHOOK_URL
```

### Customize Notification Format

The workflow sends notifications in this format:
```json
{
  "text": "🚀 AgentAuth Deployment - Success",
  "attachments": [{
    "color": "good",
    "fields": [
      {"title": "Environment", "value": "staging", "short": true},
      {"title": "Commit", "value": "abc1234", "short": true},
      {"title": "Author", "value": "mauriciomferz", "short": true},
      {"title": "Workflow", "value": "https://github.com/...", "short": false}
    ]
  }]
}
```

---

## 6. Testing the Pipeline

### Local Testing with `act`

**Install act** (test GitHub Actions locally):
```bash
# macOS
brew install act

# Linux
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash
```

**Create `.secrets` file** (for act):
```bash
cat > .secrets <<EOF
DOCKER_REGISTRY=ghcr.io
DOCKER_USERNAME=mauriciomferz
DOCKER_PASSWORD=ghp_your_github_pat
KUBE_CONFIG_STAGING=$(cat ~/.kube/config | base64)
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...
EOF
```

**Run workflow locally**:
```bash
# List available workflows
act -l

# Run push event (triggers deploy-staging.yml)
act push --secret-file .secrets

# Run specific job
act push --secret-file .secrets --job test
act push --secret-file .secrets --job security
act push --secret-file .secrets --job build

# Note: Deploy job requires actual Kubernetes cluster access
```

**Cleanup**:
```bash
rm .secrets  # Don't commit secrets!
```

### Push to GitHub

**Option 1: Push to main** (triggers deployment):
```bash
# Ensure all changes committed
git status

# Push to GitHub
git push origin main

# Watch workflow
# Go to: https://github.com/mauriciomferz/Gauth_go/actions
```

**Option 2: Manual workflow dispatch** (test without push):
```bash
# Go to GitHub Actions UI
# 1. Navigate to: https://github.com/mauriciomferz/Gauth_go/actions
# 2. Click "Deploy to Staging" workflow
# 3. Click "Run workflow"
# 4. Select environment: staging
# 5. Skip tests: false (run all tests)
# 6. Click "Run workflow"
```

### Monitor Pipeline Execution

**Via GitHub UI**:
1. Go to **Actions** tab
2. Click on latest workflow run
3. Monitor each job:
   - ✅ test (unit tests, RFC compliance, security regression)
   - ✅ security (gosec, govulncheck)
   - ✅ build (Docker build, Trivy scan)
   - ✅ deploy (kubectl apply, smoke tests)
   - ⚠️ rollback (only if deploy fails)

**Via GitHub CLI** (optional):
```bash
# Install gh CLI
brew install gh  # macOS

# Authenticate
gh auth login

# Watch workflow
gh run watch

# View logs
gh run view --log
```

**Via kubectl** (check deployed pods):
```bash
# Watch pods
watch -n 2 kubectl get pods -n gauth-staging

# Check rollout status
kubectl rollout status deployment/gauth-deployment -n gauth-staging

# Check logs
kubectl logs -f deployment/gauth-deployment -n gauth-staging

# Check service
kubectl get svc -n gauth-staging

# Check ingress
kubectl get ingress -n gauth-staging
```

---

## 7. Troubleshooting

### Common Issues

#### Issue: `Error: failed to push image`
**Cause**: Invalid Docker registry credentials  
**Solution**:
```bash
# Verify credentials locally
docker login $DOCKER_REGISTRY -u $DOCKER_USERNAME -p $DOCKER_PASSWORD

# Check GitHub secret values (Settings → Secrets)
# Re-generate Docker password if needed
```

---

#### Issue: `Error: unable to connect to Kubernetes cluster`
**Cause**: Invalid kubeconfig or expired token  
**Solution**:
```bash
# Test kubeconfig locally
kubectl cluster-info

# Verify base64 encoding
cat ~/.kube/config | base64 -d | kubectl cluster-info --kubeconfig=-

# Re-generate KUBE_CONFIG_STAGING secret with fresh token
```

---

#### Issue: `Error: ImagePullBackOff`
**Cause**: Kubernetes can't pull image from registry  
**Solution**:
```bash
# Check if image exists
docker pull $DOCKER_REGISTRY/$IMAGE_NAME:staging

# For private registries, create ImagePullSecret
kubectl create secret docker-registry regcred \
  --docker-server=$DOCKER_REGISTRY \
  --docker-username=$DOCKER_USERNAME \
  --docker-password=$DOCKER_PASSWORD \
  --namespace=gauth-staging

# Add imagePullSecrets to deployment
# spec:
#   imagePullSecrets:
#   - name: regcred
```

---

#### Issue: `Error: Rollout timeout after 5m`
**Cause**: Pods not becoming ready (health checks failing)  
**Solution**:
```bash
# Check pod status
kubectl get pods -n gauth-staging

# Describe pod
kubectl describe pod <pod-name> -n gauth-staging

# Check logs
kubectl logs <pod-name> -n gauth-staging

# Common causes:
# - Database connection failure (check postgres-service)
# - Redis connection failure (check redis-service)
# - Missing secrets (check gauth-secrets)
# - Health check misconfiguration
```

---

#### Issue: `Error: Trivy found vulnerabilities`
**Cause**: Docker image contains CRITICAL or HIGH severity vulnerabilities  
**Solution**:
```bash
# Run Trivy locally
docker pull $DOCKER_REGISTRY/$IMAGE_NAME:staging
trivy image --severity CRITICAL,HIGH $DOCKER_REGISTRY/$IMAGE_NAME:staging

# Update base image in Dockerfile
# FROM golang:1.21-alpine -> golang:1.21.5-alpine (patch version)

# Or temporarily disable Trivy check (NOT RECOMMENDED)
# exit-code: 0  # In deploy-staging.yml
```

---

#### Issue: `Error: gosec or govulncheck failed`
**Cause**: Security vulnerabilities detected in Go code  
**Solution**:
```bash
# Run locally
gosec ./...
govulncheck ./...

# Fix vulnerabilities in code
# Update dependencies: go get -u ./...
# Run tests: go test ./...
```

---

#### Issue: Slack notifications not received
**Cause**: Invalid webhook URL or Slack app disabled  
**Solution**:
```bash
# Test webhook locally
curl -X POST -H 'Content-type: application/json' \
  --data '{"text":"Test"}' \
  $SLACK_WEBHOOK_URL

# Check Slack app status in https://api.slack.com/apps
# Verify webhook URL in GitHub secrets
```

---

### Debug Mode

Enable debug logging in GitHub Actions:

1. Go to **Settings** → **Secrets and variables** → **Actions**
2. Add secret: `ACTIONS_STEP_DEBUG` = `true`
3. Add secret: `ACTIONS_RUNNER_DEBUG` = `true`
4. Re-run workflow

This will show detailed logs for each step.

---

### Useful Commands

**Check workflow syntax**:
```bash
# Install actionlint
brew install actionlint  # macOS

# Lint workflow file
actionlint .github/workflows/deploy-staging.yml
```

**View GitHub Actions logs**:
```bash
# Install gh CLI
brew install gh

# List runs
gh run list

# View specific run
gh run view <run-id>

# Download logs
gh run download <run-id>
```

**Check Kubernetes events**:
```bash
# All events in namespace
kubectl get events -n gauth-staging --sort-by='.lastTimestamp'

# Watch events
kubectl get events -n gauth-staging --watch
```

---

## Summary Checklist

Before pushing to GitHub, verify:

- [ ] `DOCKER_REGISTRY` secret configured
- [ ] `DOCKER_USERNAME` secret configured
- [ ] `DOCKER_PASSWORD` secret configured (PAT or AWS key)
- [ ] `KUBE_CONFIG_STAGING` secret configured (base64-encoded)
- [ ] `SLACK_WEBHOOK_URL` secret configured
- [ ] Kubernetes cluster accessible (`kubectl cluster-info`)
- [ ] NGINX Ingress Controller installed
- [ ] cert-manager installed with ClusterIssuer
- [ ] metrics-server installed
- [ ] Namespace `gauth-staging` created
- [ ] Secrets applied with actual JWT/Ed25519 keys and passwords
- [ ] Docker registry accessible (test `docker login`)
- [ ] Slack webhook tested (test `curl`)

Once all checklist items are verified:
```bash
git push origin main
```

Then monitor the workflow at: https://github.com/mauriciomferz/Gauth_go/actions

---

**Documentation Version**: 1.0  
**Last Updated**: November 2024  
**Maintainer**: GitHub Copilot
