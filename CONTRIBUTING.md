# Contributing to AgentAuth

Thank you for your interest in contributing to the AgentAuth framework! We are building the Trust Layer for the Agentic Economy, and we welcome contributions from the community.

## Code of Conduct

This project adheres to a standard Code of Conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior to `conduct@agentauth.org`.

## How to Contribute

### 1. Reporting Bugs
- Use GitHub Issues.
- Provide a clear reproduction case (code snippets or `curl` commands).
- Tag with `bug`.

### 2. Proposing Features (RFCs)
- **Do not open a PR blindly.**
- Open an Issue tagged `proposal` or `rfc` first.
- AgentAuth is a security-critical protocol. Changes to the core spec (AAP-001/AAP-002) require a formal review process.

### 3. Pull Requests
1.  Fork the repo and create your branch from `main`.
2.  Run tests locally: `go test ./...`
3.  Ensure `go vet ./...` passes.
4.  Commit with semantic messages (e.g., `feat: add eIDAS adapter support`).
5.  Open the PR.

## Development Setup

```bash
# Clone
git clone https://github.com/agentauth/agentauth.git

# Run dependencies
docker-compose up -d

# Run tests
go test ./...
```

## Security Vulnerabilities

**DO NOT REPORT SECURITY ISSUES PUBLICALY.**
If you discover a vulnerability, please email `security@agentauth.org` immediately. We offer a bug bounty for critical disclosures.

## License

By contributing, you agree that your contributions will be licensed under the MIT License of the project.
