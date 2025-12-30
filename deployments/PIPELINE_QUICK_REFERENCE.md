---
title: CI/CD Pipeline Quick Reference
category: cicd-quickref
status: active
lastUpdated: 2025-11-12
owners: platform-eng
source: internal
refreshCadence: quarterly
---

# CI/CD Pipeline Quick Reference

**Week 4 Day 4** | Last Updated: November 9, 2025

---

## 🚀 Quick Start: Push to GitHub

```bash
# 1. Validate locally (optional)
./scripts/validate-cicd.sh

# 2. Push to GitHub (triggers pipeline)
git push origin main

# 3. Open GitHub Actions
open https://github.com/mauriciomferz/AgentAuth/actions
# Or: gh run watch
```

**Expected**: Workflow "Deploy to Staging" starts immediately

---

## 📊 Pipeline Overview

| Job | Duration | Purpose |
|-----|----------|---------|
| 1️⃣ test | ~4 min | Run 2500+ unit tests, RFC compliance, security regression |
| 2️⃣ security | ~2 min | gosec SAST scan, govulncheck CVE scan |
| 3️⃣ build | ~3 min | Build Docker image, Trivy scan, push to GHCR |
| 4️⃣ deploy | ~6 min | Deploy to Kubernetes, smoke tests, Slack notification |
| 5️⃣ rollback | auto | Only if deploy fails - automatic rollback |

**Total Time**: ~15 minutes

---

## 🎯 Monitor Pipeline

### Via GitHub Web UI
```
https://github.com/mauriciomferz/AgentAuth/actions
```

### Via GitHub CLI
```bash
# Watch workflow in real-time
gh run watch

# List recent runs
gh run list --limit 5

# View specific run logs
gh run view <RUN_ID> --log

# View specific job
gh run view <RUN_ID> --log --job test
gh run view <RUN_ID> --log --job security
gh run view <RUN_ID> --log --job build
gh run view <RUN_ID> --log --job deploy
```

### Via kubectl
```bash
# Watch pods
kubectl get pods -n agentauth-staging --watch

# Check rollout status
kubectl rollout status deployment/agentauth-deployment -n agentauth-staging

# Tail logs
kubectl logs -f deployment/agentauth-deployment -n agentauth-staging
```

---

## ✅ Success Indicators

### GitHub Actions
- ✅ All 5 jobs show green checkmark
- ✅ Workflow status: "Success"
- ✅ Duration: ~15-20 minutes

### Kubernetes
```bash
# All pods running
kubectl get pods -n agentauth-staging
# Expected: 5 pods, all 1/1 Ready

# Deployment rolled out
kubectl rollout status deployment/agentauth-deployment -n agentauth-staging
# Expected: "successfully rolled out"

# Endpoints responding
curl https://agentauth-staging.yourdomain.com/healthz
# Expected: HTTP 200 {"status":"ok"}
```

### Slack
- ✅ Notification received: "🎉 AgentAuth Deployment Successful"

---

## ❌ Common Failures

### 1. Docker Login Failed
**Error**: `unauthorized: authentication required`

**Fix**:
```bash
# Regenerate GitHub PAT
# https://github.com/settings/tokens/new
# Scopes: write:packages, read:packages

# Update secret
# https://github.com/mauriciomferz/AgentAuth/settings/secrets/actions
# Secret: DOCKER_PASSWORD
```

### 2. Kubernetes Connection Failed
**Error**: `Unable to connect to the server`

**Fix**:
```bash
# Test locally
kubectl cluster-info

# Regenerate secret
cat ~/.kube/config | base64 | pbcopy

# Update secret
# https://github.com/mauriciomferz/AgentAuth/settings/secrets/actions
# Secret: KUBE_CONFIG_STAGING
```

### 3. Rollout Timeout
**Error**: `timed out waiting for the condition`

**Fix**:
```bash
# Check pod status
kubectl get pods -n agentauth-staging
kubectl describe pod <POD_NAME> -n agentauth-staging

# Check events
kubectl get events -n agentauth-staging --sort-by='.lastTimestamp'

# Common causes:
# - ImagePullBackOff: Create ImagePullSecret
# - CrashLoopBackOff: Check logs with kubectl logs
# - Insufficient resources: Scale down or add nodes
```

---

## 🔍 Post-Deployment Verification

```bash
# 1. Check all pods running
kubectl get pods -n agentauth-staging
# Expected: 5 pods (3 agentauth + 1 postgres + 1 redis)

# 2. Test health endpoint
curl https://agentauth-staging.yourdomain.com/healthz
# Expected: HTTP 200 {"status":"ok"}

# 3. Test beta API
curl https://agentauth-staging.yourdomain.com/api/v1/beta/health
# Expected: HTTP 200 {"status":"healthy","version":"beta"}

# 4. Check metrics
curl https://agentauth-staging.yourdomain.com/metrics | grep agentauth_
# Expected: Prometheus metrics

# 5. Verify logs
kubectl logs -f deployment/agentauth-deployment -n agentauth-staging
# Expected: No errors, "Server listening on :8080"

# 6. Check HPA
kubectl get hpa -n agentauth-staging
# Expected: min=3, max=10, current=3

# 7. Check resource usage
kubectl top pods -n agentauth-staging
# Expected: CPU <2000m, Memory <4Gi per pod
```

---

## 📝 Required GitHub Secrets

| Secret | Value | How to Get |
|--------|-------|------------|
| `DOCKER_REGISTRY` | `ghcr.io` | (Use as-is for GHCR) |
| `DOCKER_USERNAME` | `mauriciomferz` | GitHub username |
| `DOCKER_PASSWORD` | `ghp_xxx...` | Generate at https://github.com/settings/tokens/new |
| `KUBE_CONFIG_STAGING` | `base64` | `cat ~/.kube/config \| base64` |
| `SLACK_WEBHOOK_URL` | `https://hooks.slack.com/...` | Create at https://api.slack.com/apps |
| `CODECOV_TOKEN` | (optional) | Get from https://codecov.io |

**Configure at**: https://github.com/mauriciomferz/AgentAuth/settings/secrets/actions

---

## 🎯 Next Steps (Week 4 Day 5)

After successful pipeline execution:

1. **Blue-Green Deployment Testing**
   ```bash
   # Deploy to green environment
   kubectl apply -f deployments/k8s/staging/bluegreen/agentauth-deployment-green.yaml
   
   # Switch traffic
   ./deployments/k8s/staging/bluegreen/switch-traffic.sh green
   
   # Test zero-downtime
   # (Run continuous health checks during switch)
   ```

2. **Create Week 4 Day 4 Report**
   - Screenshot workflow success
   - Document pipeline execution times
   - Capture pod status, logs, metrics

3. **Tag commit** (optional)
   ```bash
   git tag -a week4-day4-complete -m "Week 4 Day 4: CI/CD pipeline execution"
   git push origin week4-day4-complete
   ```

---

## 📚 Additional Resources

- **Setup Guide**: `deployments/GITHUB_ACTIONS_SETUP.md`
- **Validation Script**: `scripts/validate-cicd.sh`
- **Pre-flight Check**: `scripts/preflight-check.sh`
- **Blue-Green Guide**: `deployments/k8s/staging/bluegreen/README.md`
- **GitHub Actions**: https://github.com/mauriciomferz/AgentAuth/actions
- **Kubernetes Dashboard**: (if installed)

---

## ⚡ Emergency Rollback

If deployment fails and automatic rollback doesn't trigger:

```bash
# Manual rollback
kubectl rollout undo deployment/agentauth-deployment -n agentauth-staging

# Check status
kubectl rollout status deployment/agentauth-deployment -n agentauth-staging

# Verify previous version running
kubectl get pods -n agentauth-staging
```

---

**Quick Help**: See `deployments/PIPELINE_EXECUTION_GUIDE.md` for detailed troubleshooting
