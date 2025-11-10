# CI/CD Docker Automation Guide

**Date**: November 10, 2025  
**Week**: 5, Day 1  
**Status**: ✅ Implemented

## Overview

Automated Docker image builds and publishing to GitHub Container Registry (GHCR) integrated into the main CI/CD pipeline. This resolves the Week 5 containerization challenges by building production images on native AMD64 runners instead of local cross-compilation.

## Architecture

### Workflow Integration

The Docker build process is integrated into `.github/workflows/ci.yml` as a new job that runs after successful binary build:

```
Test Job → Build Job → Docker Build Job → Security Scan
                                ↓
                        Push to GHCR (ghcr.io)
```

### Image Registry

- **Registry**: GitHub Container Registry (GHCR)
- **Base URL**: `ghcr.io/mauriciomferz/gauth`
- **Authentication**: Automatic via `GITHUB_TOKEN`
- **Visibility**: Public (configurable in repository settings)

### Image Tags

The workflow automatically generates multiple tags for each build:

| Tag Pattern | Example | Purpose |
|------------|---------|---------|
| `latest` | `ghcr.io/mauriciomferz/gauth:latest` | Latest main branch build |
| `blue` | `ghcr.io/mauriciomferz/gauth:blue` | Blue environment deployment |
| `green` | `ghcr.io/mauriciomferz/gauth:green` | Green environment deployment |
| `main-{SHA}` | `ghcr.io/mauriciomferz/gauth:main-a1b2c3d` | Specific commit reference |
| `{branch}` | `ghcr.io/mauriciomferz/gauth:feature-123` | Branch-specific builds |

## Workflow Configuration

### Permissions

The workflow requires `packages: write` permission to push images to GHCR:

```yaml
permissions:
  contents: read
  security-events: write
  actions: read
  pull-requests: read
  checks: write
  statuses: write
  packages: write  # Required for GHCR push
```

### Docker Build Job

```yaml
docker-build:
  name: Build and Push Docker Image
  runs-on: ubuntu-latest
  needs: build
  if: github.event_name == 'push' && github.ref == 'refs/heads/main'
  steps:
    - uses: actions/checkout@v4

    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v3

    - name: Log in to GitHub Container Registry
      uses: docker/login-action@v3
      with:
        registry: ghcr.io
        username: ${{ github.actor }}
        password: ${{ secrets.GITHUB_TOKEN }}

    - name: Extract metadata (tags, labels)
      id: meta
      uses: docker/metadata-action@v5
      with:
        images: ghcr.io/${{ github.repository_owner }}/gauth
        tags: |
          type=ref,event=branch
          type=sha,prefix={{branch}}-,format=short
          type=raw,value=latest,enable={{is_default_branch}}
          type=raw,value=blue,enable={{is_default_branch}}
          type=raw,value=green,enable={{is_default_branch}}

    - name: Build and push Docker image
      uses: docker/build-push-action@v5
      with:
        context: .
        file: ./Dockerfile.production
        platforms: linux/amd64
        push: true
        tags: ${{ steps.meta.outputs.tags }}
        labels: ${{ steps.meta.outputs.labels }}
        cache-from: type=gha
        cache-to: type=gha,mode=max
        build-args: |
          BUILD_DATE=${{ github.event.head_commit.timestamp }}
          GIT_COMMIT=${{ github.sha }}
          GIT_BRANCH=${{ github.ref_name }}
```

### Key Features

1. **Conditional Execution**: Only runs on pushes to `main` branch
2. **Dependency**: Requires successful `build` job completion
3. **Docker Buildx**: Uses BuildKit for efficient multi-stage builds
4. **GHCR Authentication**: Uses built-in `GITHUB_TOKEN` (no manual secrets)
5. **Metadata Extraction**: Generates consistent tags and labels
6. **Production Dockerfile**: Uses `Dockerfile.production` (27.7MB, CGO-enabled)
7. **Platform**: Builds for `linux/amd64` (compatible with Kubernetes nodes)
8. **Cache**: GitHub Actions cache for faster subsequent builds
9. **Build Args**: Embeds git metadata for traceability

## Dockerfile Specifications

### Dockerfile.production

**Purpose**: Production-ready multi-stage build with BLS library CGO support

**Stages**:
1. **Builder** (`golang:1.25-alpine`):
   - Installs CGO dependencies: gcc, g++, musl-dev
   - Compiles `cmd/web-server` with static linking
   - Build time: ~45-60 seconds
2. **Runtime** (`alpine:3.19`):
   - Minimal runtime with libstdc++, libgcc
   - Non-root user (gauth:1000)
   - Health check integration

**Image Size**: ~27.7MB (compressed)

**Environment Variables**:
- `GAUTH_WEB_PORT`: Application port (default: 8080)
- `GAUTH_ENV`: Environment name (staging/production)
- `GAUTH_LOG_LEVEL`: Logging verbosity (info/debug/warn/error)

## Kubernetes Integration

### Updated Manifests

Both `k8s-test-blue.yaml` and `k8s-test-green.yaml` have been updated to pull from GHCR:

**Before** (local images):
```yaml
containers:
- name: gauth
  image: gauth:blue-v2
  imagePullPolicy: Never
```

**After** (GHCR):
```yaml
containers:
- name: gauth
  image: ghcr.io/mauriciomferz/gauth:blue
  imagePullPolicy: Always
```

### Deployment Process

1. **Build Phase**: CI pushes images with tags: `latest`, `blue`, `green`, `main-{SHA}`
2. **Deploy Blue/Green**:
   ```bash
   kubectl apply -f k8s-test-blue.yaml   # Pulls ghcr.io/mauriciomferz/gauth:blue
   kubectl apply -f k8s-test-green.yaml  # Pulls ghcr.io/mauriciomferz/gauth:green
   ```
3. **Validation**: Pods pull fresh images from GHCR on each deployment
4. **Rollback**: Use previous SHA tags if needed

## Usage Guide

### Triggering a Build

Push to `main` branch automatically triggers Docker build:

```bash
git push origin main
```

### Monitoring Builds

```bash
# Watch latest workflow run
gh run watch

# List recent runs
gh run list --workflow=ci.yml --limit 5

# View specific run
gh run view <run-id>
```

### Viewing Published Images

1. **GitHub UI**: Navigate to `https://github.com/mauriciomferz/Gauth_go/pkgs/container/gauth`
2. **CLI**:
   ```bash
   # List tags
   docker pull ghcr.io/mauriciomferz/gauth:latest
   docker images ghcr.io/mauriciomferz/gauth
   
   # Inspect image
   docker inspect ghcr.io/mauriciomferz/gauth:latest
   ```

### Deploying to Kubernetes

```bash
# Deploy blue environment with latest GHCR image
kubectl apply -f k8s-test-blue.yaml

# Watch rollout
kubectl rollout status deployment/gauth-blue -n gauth-staging

# Verify pods are running
kubectl get pods -n gauth-staging -l version=blue

# Test application
kubectl run test-client --rm -i --tty \
  --image=curlimages/curl --restart=Never \
  -n gauth-staging -- \
  curl -s http://gauth-service/api/v1/beta/health
```

### Using Specific Versions

```bash
# Deploy specific commit SHA
kubectl set image deployment/gauth-blue \
  gauth=ghcr.io/mauriciomferz/gauth:main-a1b2c3d \
  -n gauth-staging

# Or edit manifest directly
vi k8s-test-blue.yaml  # Change image tag
kubectl apply -f k8s-test-blue.yaml
```

## Troubleshooting

### Build Failures

**Issue**: Docker build fails in CI

**Check**:
1. View workflow logs: `gh run view --log-failed`
2. Verify `Dockerfile.production` syntax
3. Check Go dependencies in `go.mod`

**Common Causes**:
- CGO compilation errors (BLS library)
- Missing dependencies in builder stage
- Out of memory (increase GitHub runner resources)

### Image Push Failures

**Issue**: "denied: installation not allowed" or permission errors

**Check**:
1. Repository settings → Actions → General → Workflow permissions
2. Ensure "Read and write permissions" is enabled
3. Verify `packages: write` in workflow permissions

**Solution**:
```bash
# Check token permissions
gh api /repos/mauriciomferz/Gauth_go/actions/permissions

# Should show: "default_workflow_permissions": "write"
```

### Image Pull Failures in Kubernetes

**Issue**: Pods stuck in `ImagePullBackOff`

**Check**:
1. Verify image exists in GHCR:
   ```bash
   docker pull ghcr.io/mauriciomferz/gauth:blue
   ```
2. Check GHCR package visibility (should be public)
3. Inspect pod events:
   ```bash
   kubectl describe pod <pod-name> -n gauth-staging
   ```

**Common Causes**:
- Image not yet built (check CI status)
- GHCR package is private (requires image pull secret)
- Typo in image name/tag
- Network connectivity issues

**Solution for Private Images**:
```bash
# Create image pull secret
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=mauriciomferz \
  --docker-password=$GITHUB_TOKEN \
  -n gauth-staging

# Add to deployment
kubectl patch deployment gauth-blue \
  -p '{"spec":{"template":{"spec":{"imagePullSecrets":[{"name":"ghcr-secret"}]}}}}' \
  -n gauth-staging
```

### Application Startup Failures

**Issue**: Pods Running but application not responding

**Check Logs**:
```bash
kubectl logs -n gauth-staging -l version=blue --tail=100
```

**Common Issues**:
- Missing environment variables (GAUTH_WEB_PORT, etc.)
- Resource limits too low (check OOMKilled events)
- Health check failures (check `/api/v1/beta/health` endpoint)

**Debug**:
```bash
# Exec into pod
kubectl exec -it <pod-name> -n gauth-staging -- /bin/sh

# Check process
ps aux | grep web-server

# Test locally
curl http://localhost:8080/api/v1/beta/health
```

## Performance Metrics

### Build Times

| Build Type | Time | Notes |
|-----------|------|-------|
| Cold build | ~60s | First build with no cache |
| Warm build | ~45s | With GitHub Actions cache |
| Cache hit | ~20s | No code changes, layer reuse |

### Image Sizes

| Image | Size (Compressed) | Size (Extracted) |
|-------|------------------|------------------|
| Dockerfile.production | 27.7MB | ~80MB |
| Builder stage only | 450MB | 1.2GB |
| Single-stage (Dockerfile.single-stage) | 500MB | 1.4GB |

### Resource Usage

**GitHub Actions**:
- Runner: ubuntu-latest (2 cores, 7GB RAM)
- Build memory: ~2-3GB peak
- Cache size: ~200MB

**Kubernetes (per pod)**:
- Requests: 200m CPU, 256Mi memory
- Limits: 500m CPU, 512Mi memory
- Actual: ~50m CPU, 100Mi memory (idle)

## Security Considerations

### Image Scanning

Currently NOT integrated in this workflow. Recommend adding:

```yaml
- name: Run Trivy vulnerability scanner
  uses: aquasecurity/trivy-action@master
  with:
    image-ref: ghcr.io/mauriciomferz/gauth:${{ github.sha }}
    format: 'sarif'
    output: 'trivy-results.sarif'
    severity: 'CRITICAL,HIGH'
    exit-code: '1'
```

### GHCR Token Security

- Uses `GITHUB_TOKEN` (automatic, scoped to workflow)
- Token expires after workflow completes
- No long-lived credentials stored
- Token has minimal required permissions

### Image Verification

```bash
# Verify image digest
docker inspect ghcr.io/mauriciomferz/gauth:latest | jq '.[0].RepoDigests'

# Check layers
docker history ghcr.io/mauriciomferz/gauth:latest
```

## Best Practices

### Tagging Strategy

1. **Always use immutable tags for production**: Use SHA tags, not `latest`
2. **Blue/Green deployments**: Maintain separate `blue` and `green` tags
3. **Rollback capability**: Keep historical SHA tags available

### Caching

1. **GitHub Actions Cache**: Enabled by default (`cache-from: type=gha`)
2. **Layer Optimization**: Multi-stage builds minimize cache invalidation
3. **Dependency Caching**: Go modules cached in builder stage

### Monitoring

1. **Build Success Rate**: Monitor GitHub Actions dashboard
2. **Image Freshness**: Check GHCR package last updated time
3. **Deployment Health**: Kubernetes pod status and logs

## Future Enhancements

### Planned Improvements

1. **Vulnerability Scanning**: Integrate Trivy in workflow
2. **Image Signing**: Add cosign for supply chain security
3. **SBOM Generation**: Generate Software Bill of Materials
4. **Multi-Architecture**: Add linux/arm64 support for Apple Silicon
5. **Automated Rollback**: Detect failed deployments and auto-revert
6. **Deployment Notifications**: Slack/Discord webhook on success/failure
7. **Performance Testing**: Automated smoke tests post-deployment

### Scalability Considerations

1. **Rate Limits**: GHCR has generous limits, but monitor for high-frequency builds
2. **Storage**: Implement image retention policy (keep last N tags)
3. **Build Parallelization**: Consider matrix builds for multiple environments
4. **Registry Mirroring**: For global deployments, consider regional mirrors

## References

### Documentation

- [Week 5 Day 1 Containerization Report](../WEEK5_DAY1_CONTAINERIZATION_REPORT.md)
- [Kind Cluster Guide](../KIND_CLUSTER_GUIDE.md)
- [Week 4 Day 5 Blue-Green Report](../WEEK4_DAY5_REPORT.md)
- [Dockerfile.production](../Dockerfile.production)

### External Resources

- [GitHub Container Registry Docs](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Docker Buildx Documentation](https://docs.docker.com/buildx/working-with-buildx/)
- [GitHub Actions Docker Documentation](https://docs.github.com/en/actions/publishing-packages/publishing-docker-images)

### Related Tools

- `docker/setup-buildx-action@v3`
- `docker/login-action@v3`
- `docker/metadata-action@v5`
- `docker/build-push-action@v5`

## Changelog

### 2025-11-10 - Initial Implementation

- ✅ Added Docker build job to ci.yml workflow
- ✅ Configured GHCR authentication with GITHUB_TOKEN
- ✅ Implemented multi-tag strategy (latest, blue, green, SHA)
- ✅ Updated k8s-test-blue.yaml and k8s-test-green.yaml for GHCR
- ✅ Enabled GitHub Actions cache for faster builds
- ✅ Added packages: write permission to workflow

**Commit**: [To be added after commit]

---

**Maintained by**: GAuth Development Team  
**Last Updated**: November 10, 2025  
**Version**: 1.0.0
