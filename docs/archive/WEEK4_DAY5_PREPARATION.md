# Week 4 Day 5 - Preparation and Objectives

**Date**: November 10, 2025
**Previous Session**: Week 4 Day 4 - COMPLETE ✅
**Status**: Ready to begin

---

## Week 4 Day 4 Recap

### Completed Objectives ✅

1. **Test Fixes**: 5 production race conditions eliminated
2. **Blue-Green Deployment**: Validated with 12/12 tests passing
3. **CI/CD Pipeline**: Verified and passing
4. **Documentation**: Comprehensive report created (728 lines)

### Commits Delivered

- `f59f387d`: pkg/authz + ai_capability_demo fixes
- `e16b1081`: pkg/rfc0111 date comparison fix
- `a755ce3d`: web modelLimits mutex protection
- `c594b2fe`: Blue-green validation script
- `a3c262b8`: Week 4 Day 4 completion report
- `a34a33b6`: Report update with CI/CD results

### Current System Status

✅ **Production Ready** (with documented exceptions)
- All critical race conditions fixed
- Blue-green deployment strategy validated
- CI/CD pipelines passing
- Security scan clean

⏸️ **Deferred to Week 5**
- `internal/crypto`: TenantScheduler architectural refactoring
- `test/load`: Performance optimization

---

## Week 4 Day 5 Objectives

### Primary Goal
**Deploy AgentAuth to actual Kubernetes staging cluster and validate blue-green deployment strategy under real-world conditions.**

### Detailed Objectives

#### 1. Kubernetes Cluster Setup
- [ ] Provision or access Kubernetes staging cluster
- [ ] Configure kubectl with cluster credentials
- [ ] Create `gauth-staging` namespace
- [ ] Verify cluster connectivity and permissions

#### 2. Deploy Blue Environment
- [ ] Apply blue deployment manifest
  ```bash
  kubectl apply -f deployments/k8s/staging/bluegreen/gauth-deployment-blue.yaml
  ```
- [ ] Verify blue pods are running and healthy
- [ ] Check readiness and liveness probes
- [ ] Validate service selectors (version: blue)

#### 3. Deploy Green Environment
- [ ] Apply green deployment manifest
  ```bash
  kubectl apply -f deployments/k8s/staging/bluegreen/gauth-deployment-green.yaml
  ```
- [ ] Verify green pods are running and healthy
- [ ] Ensure both environments coexist independently

#### 4. Configure Services and Ingress
- [ ] Apply services manifest
  ```bash
  kubectl apply -f deployments/k8s/staging/bluegreen/gauth-services.yaml
  ```
- [ ] Apply ingress manifest
  ```bash
  kubectl apply -f deployments/k8s/staging/bluegreen/gauth-ingress.yaml
  ```
- [ ] Verify ingress routes to blue service initially

#### 5. Test Traffic Switching
- [ ] Execute traffic switch script
  ```bash
  cd deployments/k8s/staging/bluegreen
  ./switch-traffic.sh green
  ```
- [ ] Monitor ingress backend switching
- [ ] Verify zero-downtime during switch
- [ ] Test with real HTTP requests

#### 6. Load Testing
- [ ] Generate realistic traffic load
- [ ] Monitor metrics during traffic switch
- [ ] Measure actual switch latency
- [ ] Verify session affinity (ClientIP)

#### 7. Rollback Testing
- [ ] Execute rollback to blue
  ```bash
  ./switch-traffic.sh blue
  ```
- [ ] Measure rollback time (target: <10s)
- [ ] Verify instant rollback capability
- [ ] Test multiple back-and-forth switches

#### 8. Monitoring and Observability
- [ ] Validate Prometheus metrics collection
- [ ] Check Grafana dashboards (if available)
- [ ] Review logs from both environments
- [ ] Verify alerting rules (if configured)

#### 9. Documentation Updates
- [ ] Document actual deployment experience
- [ ] Capture lessons learned
- [ ] Update procedures based on findings
- [ ] Create troubleshooting guide

#### 10. Week 4 Day 5 Report
- [ ] Document deployment process
- [ ] Record actual metrics (rollback time, latency, etc.)
- [ ] Compare expected vs. actual behavior
- [ ] Identify optimization opportunities

---

## Prerequisites Checklist

### Access and Credentials

- [ ] Kubernetes cluster access (staging environment)
- [ ] `kubectl` configured with correct context
- [ ] Cluster admin or namespace admin permissions
- [ ] Docker registry access (for image pull)

### Configuration Files

- [x] Blue deployment manifest (gauth-deployment-blue.yaml)
- [x] Green deployment manifest (gauth-deployment-green.yaml)
- [x] Services manifest (gauth-services.yaml)
- [x] Ingress manifest (gauth-ingress.yaml)
- [x] Traffic switch script (switch-traffic.sh)
- [x] Validation script (validate-bluegreen.sh)

### Container Images

- [ ] Docker image built and pushed to registry
- [ ] Image tagged appropriately (e.g., `gauth:v1.0.0`)
- [ ] Image pull secrets configured (if private registry)

### Dependencies

- [x] kubectl installed and configured
- [ ] Helm (if using Helm charts)
- [ ] curl/httpie for testing endpoints
- [ ] Load testing tool (e.g., k6, ab, wrk)

---

## Expected Outcomes

### Success Criteria

1. **Blue Environment**: Successfully deployed and healthy
2. **Green Environment**: Successfully deployed and healthy
3. **Traffic Switch**: Zero-downtime switch from blue to green
4. **Session Affinity**: User sessions preserved during switch
5. **Rollback**: Instant rollback (<10s) validated
6. **Monitoring**: Metrics and logs accessible
7. **Documentation**: Complete deployment runbook

### Metrics to Capture

- **Deployment Time**: Time to deploy each environment
- **Pod Startup Time**: Time for pods to become ready
- **Switch Latency**: Time for traffic to shift between environments
- **Rollback Time**: Time to revert to previous environment
- **Error Rate**: Before, during, and after switch
- **Request Latency**: P50, P95, P99 during operations
- **Session Preservation**: Percentage of sessions maintained

---

## Potential Challenges

### Known Issues

1. **Kubernetes Cluster**: May not have access to actual cluster
   - **Mitigation**: Use minikube/kind for local testing
   - **Alternative**: Document dry-run deployment process

2. **Docker Registry**: May need registry configuration
   - **Mitigation**: Use local registry or GHCR
   - **Alternative**: Document image push requirements

3. **Ingress Controller**: Requires nginx ingress controller
   - **Mitigation**: Install ingress controller if missing
   - **Alternative**: Use port-forward for testing

4. **Load Balancer**: External LB may not be available
   - **Mitigation**: Use NodePort or port-forward
   - **Alternative**: Document LoadBalancer requirements

### Troubleshooting Guide

**Issue**: Pods stuck in `Pending` state
- Check resource limits and cluster capacity
- Verify image pull secrets
- Review pod events: `kubectl describe pod <pod-name>`

**Issue**: Ingress not routing traffic
- Verify ingress controller is installed
- Check ingress backend service selectors
- Review ingress events and logs

**Issue**: Traffic switch not working
- Verify ingress patch command syntax
- Check service selector labels match pods
- Ensure session affinity is configured

**Issue**: Rollback fails
- Ensure both environments are running
- Verify ingress can switch back to blue
- Check for resource conflicts

---

## Next Steps After Week 4 Day 5

1. **Week 5**: Internal/crypto TenantScheduler refactoring
2. **Production Readiness**: Final security audit
3. **Performance Optimization**: Address test/load timeout
4. **Production Deployment**: Apply blue-green to production

---

## Resources

### Documentation
- [Blue-Green Deployment README](deployments/k8s/staging/bluegreen/README.md)
- [Validation Script](deployments/k8s/staging/bluegreen/validate-bluegreen.sh)
- [Week 4 Day 4 Report](WEEK4_DAY4_REPORT.md)

### Kubernetes Resources
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Ingress Nginx](https://kubernetes.github.io/ingress-nginx/)
- [Blue-Green Deployments Best Practices](https://kubernetes.io/blog/2018/04/30/zero-downtime-deployment-kubernetes-jenkins/)

### Monitoring
- [Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator)
- [Grafana Dashboards](https://grafana.com/grafana/dashboards/)

---

**Prepared**: November 10, 2025, 00:15 UTC
**Status**: Ready to begin Week 4 Day 5 deployment
**Contact**: Repository owner for cluster access credentials
