# GitHub Actions Setup Guide

This repository uses GitHub Actions for CI/CD pipeline automation. Here's how to properly configure all features.

## 🔧 Required Configuration

### 1. Slack Notifications (Optional)

To enable Slack notifications for deployment status:

1. **Create a Slack Webhook:**
   - Go to your Slack workspace
   - Navigate to https://api.slack.com/apps
   - Create a new app → "From scratch"
   - Choose your workspace
   - Go to "Incoming Webhooks" → "Activate Incoming Webhooks"
   - Click "Add New Webhook to Workspace"
   - Select the channel (e.g., #deployments)
   - Copy the webhook URL

2. **Add Secret to GitHub:**
   - Go to your repository → Settings → Secrets and variables → Actions
   - Click "New repository secret"
   - Name: `SLACK_WEBHOOK_URL`
   - Value: Your Slack webhook URL
   - Click "Add secret"

### 2. Container Registry Access

The workflow uses GitHub Container Registry (ghcr.io) which should work automatically with the `GITHUB_TOKEN`.

### 3. Environment Protection Rules

To enable environment-specific deployments:

1. Go to Settings → Environments
2. Create environments: `staging` and `production`
3. Add protection rules (optional):
   - Required reviewers
   - Wait timer
   - Branch restrictions

## ��️ Security Scanning

The pipeline includes:

- **Gosec**: Static security analysis for Go code
- **Trivy**: Vulnerability scanning for dependencies and containers
- **StaticCheck**: Advanced Go static analysis

Results are automatically uploaded to the GitHub Security tab.

## 🚀 Deployment Flow

- **Develop Branch**: Deploys to staging environment
- **Main Branch**: Deploys to production environment
- **Pull Requests**: Run tests and security scans only

## 📊 Coverage Reports

Code coverage reports are uploaded to Codecov automatically. No additional setup required for public repositories.

## 🔍 Troubleshooting

### Common Issues:

1. **"SLACK_WEBHOOK_URL not set"**: This is expected if you haven't configured Slack notifications. The pipeline will continue normally.

2. **Security scan failures**: The pipeline continues even if security scans find issues. Check the Security tab for details.

3. **Docker build failures**: Ensure your Dockerfile is properly configured and all dependencies are available.

### Logs and Debugging:

- Check the Actions tab for detailed logs
- Each job shows individual step results
- Failed steps include error messages and suggestions

## 📝 Customization

To customize the pipeline:

1. **Modify `.github/workflows/ci-cd.yml`**
2. **Update environment names** in the deployment sections
3. **Add additional steps** as needed for your deployment process
4. **Configure different notification channels** by modifying the Slack step

## 🎯 Best Practices

- Always test changes in a development branch first
- Use environment protection rules for production deployments
- Regularly update action versions for security patches
- Monitor the Security tab for vulnerability reports
- Keep secrets up to date and rotate them regularly

---

For more information, see the [GitHub Actions documentation](https://docs.github.com/en/actions).
