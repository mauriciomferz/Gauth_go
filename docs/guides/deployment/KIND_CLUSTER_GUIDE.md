---
title: KIND Local Kubernetes Cluster Guide
category: local-cluster-guide
status: active
lastUpdated: 2025-11-12
owners: platform-eng
source: internal
refreshCadence: on-change
---

# Local Kind Cluster Management Guide

## Cluster Information

- **Cluster Name**: `agentauth-staging`
- **Type**: kind (Kubernetes in Docker)
- **Namespace**: `agentauth-staging`
- **Created**: November 10, 2025

## Current Deployments

### Blue Environment
- **Deployment**: `agentauth-blue`
- **Replicas**: 2
- **Image**: `agentauth-mock:blue`
- **Status**: Running

### Green Environment
- **Deployment**: `agentauth-green`
- **Replicas**: 2
- **Image**: `agentauth-mock:green`
- **Status**: Running

### Service
- **Name**: `agentauth-service`
- **Type**: ClusterIP
- **IP**: `10.96.227.186`
- **Port**: 80 → 8080

## Common Commands

### View Cluster Status
```bash
# List all clusters
kind get clusters

# Get cluster info
kubectl cluster-info --context kind-agentauth-staging

# View all resources
kubectl get all -n agentauth-staging

# View pods with version labels
kubectl get pods -n agentauth-staging -L version
```

### Test Deployments
```bash
# Test blue environment (if service points to blue)
kubectl run test-client --rm -i --tty \
  --image=curlimages/curl:latest \
  --restart=Never -n agentauth-staging \
  -- curl -s http://agentauth-service/api/v1/beta/health

# Expected output: {"status":"healthy","version":"blue"}
# or: {"status":"healthy","version":"green"}
```

### Traffic Switching
```bash
# Switch to green
kubectl patch service agentauth-service -n agentauth-staging \
  -p '{"spec":{"selector":{"app":"agentauth","version":"green"}}}'

# Switch to blue (rollback)
kubectl patch service agentauth-service -n agentauth-staging \
  -p '{"spec":{"selector":{"app":"agentauth","version":"blue"}}}'

# Check current routing
kubectl get service agentauth-service -n agentauth-staging -o yaml | grep -A 3 selector
```

### View Logs
```bash
# Blue logs
kubectl logs -l version=blue -n agentauth-staging --tail=50

# Green logs
kubectl logs -l version=green -n agentauth-staging --tail=50

# Follow logs
kubectl logs -f deployment/agentauth-blue -n agentauth-staging
```

### Scale Deployments
```bash
# Scale blue
kubectl scale deployment agentauth-blue -n agentauth-staging --replicas=3

# Scale green
kubectl scale deployment agentauth-green -n agentauth-staging --replicas=3

# Scale down
kubectl scale deployment agentauth-blue -n agentauth-staging --replicas=1
```

### Load Testing
```bash
# Run the load test script
./load-test.sh

# Or manually with kubectl
kubectl run load-test --rm -i --tty \
  --image=curlimages/curl:latest \
  --restart=Never -n agentauth-staging \
  -- sh -c 'for i in $(seq 1 50); do curl -s http://agentauth-service/api/v1/beta/health; done'
```

## Cluster Management

### Stop the Cluster
```bash
# Stop the cluster (preserves state)
docker stop agentauth-staging-control-plane
```

### Start the Cluster
```bash
# Start the cluster
docker start agentauth-staging-control-plane

# Verify it's running
kubectl get nodes
```

### Delete the Cluster
```bash
# Delete the entire cluster
kind delete cluster --name agentauth-staging

# Verify deletion
kind get clusters
```

### Recreate the Cluster
```bash
# Create new cluster with port mapping
kind create cluster --name agentauth-staging --config - <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 80
    hostPort: 30080
    protocol: TCP
EOF

# Create namespace
kubectl create namespace agentauth-staging

# Load images
kind load docker-image agentauth-mock:blue agentauth-mock:green --name agentauth-staging

# Deploy applications
kubectl apply -f k8s-test-blue.yaml
kubectl apply -f k8s-test-green.yaml
```

## Troubleshooting

### Pods Not Starting
```bash
# Check pod status
kubectl get pods -n agentauth-staging

# Describe pod for events
kubectl describe pod <pod-name> -n agentauth-staging

# Check pod logs
kubectl logs <pod-name> -n agentauth-staging
```

### Service Not Routing
```bash
# Check service endpoints
kubectl get endpoints agentauth-service -n agentauth-staging

# Check service details
kubectl describe service agentauth-service -n agentauth-staging

# Verify selectors match pod labels
kubectl get pods -n agentauth-staging --show-labels
```

### Image Not Found
```bash
# List loaded images in kind
docker exec -it agentauth-staging-control-plane crictl images | grep agentauth

# Reload image if needed
kind load docker-image agentauth-mock:blue --name agentauth-staging
```

### Cluster Connection Issues
```bash
# Check if cluster is running
docker ps | grep agentauth-staging

# Check kubectl context
kubectl config current-context

# Switch context if needed
kubectl config use-context kind-agentauth-staging
```

## Resource Cleanup

### Delete Deployments Only
```bash
# Delete all resources in namespace (keeps namespace)
kubectl delete all --all -n agentauth-staging

# Delete specific deployment
kubectl delete deployment agentauth-blue -n agentauth-staging
kubectl delete deployment agentauth-green -n agentauth-staging
```

### Delete Namespace
```bash
# Delete namespace and all resources
kubectl delete namespace agentauth-staging
```

## Performance Notes

- Mock server responses in <1ms
- Traffic switch is atomic (instant)
- Rollback measured at ~0.2s
- Health checks every 5-10s
- Each pod uses ~64Mi RAM, 100m CPU (requests)

## Next Steps for Production

1. **Replace mock server with actual AgentAuth application**
   - Build proper multi-arch Docker image
   - Update k8s-test-blue.yaml and k8s-test-green.yaml
   - Load new images into cluster

2. **Add Ingress Controller**
   ```bash
   kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
   ```

3. **Deploy Monitoring**
   - Prometheus for metrics
   - Grafana for dashboards
   - Configure service monitors

4. **Add Databases**
   - PostgreSQL for persistent data
   - Redis for caching
   - ConfigMaps and Secrets for config

---

**Status**: Cluster active and ready for development/testing  
**Last Updated**: November 10, 2025
