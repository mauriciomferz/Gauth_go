# Kubernetes Deployment for GAuth Development

This directory contains Kubernetes manifests for deploying the GAuth development prototype.

## ⚠️ Development Status

**Important**: These manifests are for **development and testing purposes only**. GAuth is currently a development prototype and not ready for production use.

## Prerequisites

- Kubernetes cluster (local or cloud)
- kubectl configured
- Docker for building images
- Health endpoints working (✅ implemented)

## Quick Start

### 1. Build the Application Image

```bash
# From the project root
docker build -t gauth-demo:dev .
```

### 2. Deploy to Kubernetes

```bash
# Deploy all components
kubectl apply -f k8s/development/

# Check deployment status
kubectl get pods -n gauth-development
```

### 3. Access the Application

```bash
# Port forward to access locally
kubectl port-forward -n gauth-development svc/gauth-service 8080:80

# Test health endpoints
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

## Components

### Application (`gauth-deployment.yaml`)
- **Namespace**: `gauth-development`
- **Replicas**: 3 (with auto-scaling 3-10)
- **Health Checks**: ✅ `/health` and `/ready` endpoints implemented
- **Security**: Non-root containers, read-only filesystems
- **Resources**: 128Mi-512Mi memory, 100m-500m CPU

### PostgreSQL (`postgres-deployment.yaml`)
- **Storage**: 100Gi persistent volume
- **Configuration**: Optimized for development
- **Security**: SSL enabled, restricted access

### Redis (`redis-deployment.yaml`)
- **Storage**: 20Gi persistent volume
- **Configuration**: Memory-optimized for caching
- **Security**: Password protected, limited commands

## Development Features

### Health Monitoring
- Liveness probe: `/health` (checks if application is running)
- Readiness probe: `/ready` (checks if application can serve traffic)
- Metrics endpoint: Port 9090 for Prometheus monitoring

### Scaling
- Horizontal Pod Autoscaler: 3-10 replicas based on CPU/memory
- Pod Disruption Budget: Maintains at least 2 replicas during updates

### Security
- Security contexts with non-root users
- Read-only root filesystems
- Capability dropping
- Network policies ready

## Configuration

### Environment Variables
- `CONFIG_FILE`: Application configuration path
- `VAULT_TOKEN`: Vault authentication (if using Vault)
- `JWT_SIGNING_KEY`: JWT signing key

### Secrets
Update these base64-encoded secrets before deployment:
- `vault-token`: Vault access token
- `jwt-signing-key`: JWT signing key
- `postgres-password`: PostgreSQL password
- `redis-password`: Redis password

## Troubleshooting

### Common Issues

1. **Image Pull Errors**
   ```bash
   # Build image locally if using local cluster
   docker build -t gauth-demo:dev .
   
   # For minikube, use minikube's docker daemon
   eval $(minikube docker-env)
   docker build -t gauth-demo:dev .
   ```

2. **Health Check Failures**
   ```bash
   # Check if health endpoints are responding
   kubectl port-forward -n gauth-development svc/gauth-service 8080:80
   curl http://localhost:8080/health
   ```

3. **Database Connection Issues**
   ```bash
   # Check PostgreSQL pod logs
   kubectl logs -n gauth-development statefulset/postgres
   ```

### Useful Commands

```bash
# View all resources
kubectl get all -n gauth-development

# Check pod logs
kubectl logs -n gauth-development deployment/gauth-deployment

# Execute into pod
kubectl exec -it -n gauth-development deployment/gauth-deployment -- /bin/sh

# Delete deployment
kubectl delete namespace gauth-development
```

## Development Notes

- This is a **development prototype** - not production ready
- Health endpoints are implemented and functional
- Image references point to local builds
- TLS certificates and production security features may need additional setup
- Monitoring integration available but optional

## Next Steps

1. Implement proper secret management
2. Add network policies
3. Configure monitoring stack (Prometheus/Grafana)
4. Set up CI/CD pipeline for automated deployments
5. Add development vs staging environment configurations

---

**Status**: ✅ **Functional for Development** - Health endpoints implemented, namespaces updated to development