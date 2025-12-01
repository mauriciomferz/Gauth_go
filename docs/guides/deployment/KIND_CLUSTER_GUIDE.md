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

- **Cluster Name**: `gauth-staging`
- **Type**: kind (Kubernetes in Docker)
- **Namespace**: `gauth-staging`
- **Created**: November 10, 2025

## Current Deployments

### Blue Environment
- **Deployment**: `gauth-blue`
- **Replicas**: 2
- **Image**: `gauth-mock:blue`
- **Status**: Running

### Green Environment
- **Deployment**: `gauth-green`
- **Replicas**: 2
- **Image**: `gauth-mock:green`
- **Status**: Running

### Service
- **Name**: `gauth-service`
- **Type**: ClusterIP
- **IP**: `10.96.227.186`
- **Port**: 80 → 8080

## Common Commands

### View Cluster Status
```bash
# List all clusters
kind get clusters

# Get cluster info
kubectl cluster-info --context kind-gauth-staging

# View all resources
kubectl get all -n gauth-staging

# View pods with version labels
kubectl get pods -n gauth-staging -L version
```

### Test Deployments
```bash
# Test blue environment (if service points to blue)
kubectl run test-client --rm -i --tty \
  --image=curlimages/curl:latest \
  --restart=Never -n gauth-staging \
  -- curl -s http://gauth-service/api/v1/beta/health

# Expected output: {"status":"healthy","version":"blue"}
# or: {"status":"healthy","version":"green"}
```

### Traffic Switching
```bash
# Switch to green
kubectl patch service gauth-service -n gauth-staging \
  -p '{"spec":{"selector":{"app":"gauth","version":"green"}}}'

# Switch to blue (rollback)
kubectl patch service gauth-service -n gauth-staging \
  -p '{"spec":{"selector":{"app":"gauth","version":"blue"}}}'

# Check current routing
kubectl get service gauth-service -n gauth-staging -o yaml | grep -A 3 selector
```

### View Logs
```bash
# Blue logs
kubectl logs -l version=blue -n gauth-staging --tail=50

# Green logs
kubectl logs -l version=green -n gauth-staging --tail=50

# Follow logs
kubectl logs -f deployment/gauth-blue -n gauth-staging
```

### Scale Deployments
```bash
# Scale blue
kubectl scale deployment gauth-blue -n gauth-staging --replicas=3

# Scale green
kubectl scale deployment gauth-green -n gauth-staging --replicas=3

# Scale down
kubectl scale deployment gauth-blue -n gauth-staging --replicas=1
```

### Load Testing
```bash
# Run the load test script
./load-test.sh

# Or manually with kubectl
kubectl run load-test --rm -i --tty \
  --image=curlimages/curl:latest \
  --restart=Never -n gauth-staging \
  -- sh -c 'for i in $(seq 1 50); do curl -s http://gauth-service/api/v1/beta/health; done'
```

## Cluster Management

### Stop the Cluster
```bash
# Stop the cluster (preserves state)
docker stop gauth-staging-control-plane
```

### Start the Cluster
```bash
# Start the cluster
docker start gauth-staging-control-plane

# Verify it's running
kubectl get nodes
```

### Delete the Cluster
```bash
# Delete the entire cluster
kind delete cluster --name gauth-staging

# Verify deletion
kind get clusters
```

### Recreate the Cluster
```bash
# Create new cluster with port mapping
kind create cluster --name gauth-staging --config - <<EOF
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
kubectl create namespace gauth-staging

# Load images
kind load docker-image gauth-mock:blue gauth-mock:green --name gauth-staging

# Deploy applications
kubectl apply -f k8s-test-blue.yaml
kubectl apply -f k8s-test-green.yaml
```

## Troubleshooting

### Pods Not Starting
```bash
# Check pod status
kubectl get pods -n gauth-staging

# Describe pod for events
kubectl describe pod <pod-name> -n gauth-staging

# Check pod logs
kubectl logs <pod-name> -n gauth-staging
```

### Service Not Routing
```bash
# Check service endpoints
kubectl get endpoints gauth-service -n gauth-staging

# Check service details
kubectl describe service gauth-service -n gauth-staging

# Verify selectors match pod labels
kubectl get pods -n gauth-staging --show-labels
```

### Image Not Found
```bash
# List loaded images in kind
docker exec -it gauth-staging-control-plane crictl images | grep gauth

# Reload image if needed
kind load docker-image gauth-mock:blue --name gauth-staging
```

### Cluster Connection Issues
```bash
# Check if cluster is running
docker ps | grep gauth-staging

# Check kubectl context
kubectl config current-context

# Switch context if needed
kubectl config use-context kind-gauth-staging
```

## Resource Cleanup

### Delete Deployments Only
```bash
# Delete all resources in namespace (keeps namespace)
kubectl delete all --all -n gauth-staging

# Delete specific deployment
kubectl delete deployment gauth-blue -n gauth-staging
kubectl delete deployment gauth-green -n gauth-staging
```

### Delete Namespace
```bash
# Delete namespace and all resources
kubectl delete namespace gauth-staging
```

## Performance Notes

- Mock server responses in <1ms
- Traffic switch is atomic (instant)
- Rollback measured at ~0.2s
- Health checks every 5-10s
- Each pod uses ~64Mi RAM, 100m CPU (requests)

## Next Steps for Production

1. **Replace mock server with actual GAuth application**
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
