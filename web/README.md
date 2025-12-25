---
title: Readme
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

# GAuth Beta Web Interface

> Last Updated: 2025-10-17
> Status: Active

⚠️ **BETA IMPLEMENTATION NOTICE**
# GAuth Beta Web Interface
This web interface is a beta demonstration environment. It showcases the RFC-0150 implementation and related authorization concepts for evaluation and feedback. It is NOT production ready and should not be deployed in any production environment or relied upon for security decisions.

This directory contains a modern, interactive web interface for the GAuth RFC-0150 beta implementation. The interface provides hands-on experiential workflows for understanding authorization concepts, power-of-attorney flows, and AI-native authentication patterns.
## Features

### 🧪 Beta Demonstration Focus
- **Interactive Demos**: Hands-on exploration of token management, authorization flows, and event systems
- **Real-time Visualization**: Live demonstration of GAuth concepts with immediate feedback
- **Comprehensive Examples**: Access to 37+ working code examples from the repository
- **Beta Disclaimers**: Clear warnings that this is an evaluation/demo environment. NOT production ready.
### 🔐 GAuth Concepts Demonstrated
- **Token Lifecycle Management**: Create, validate, and revoke beta demo tokens
- **Event System**: Typed events with structured metadata handling
- **Audit Trail**: Compliance logging and reporting capabilities

### 🎨 Modern Interface
- **Responsive Design**: Works on desktop, tablet, and mobile devices
- server_clean.go         # Go web server for beta demo (BetaServer only)
- **Interactive Console**: Terminal-like output for hands-on learning
- **Tabbed Navigation**: Organized learning modules for different concepts
- **Visual Architecture**: Clear system architecture diagrams and explanations

## Directory Structure

```
web/
    └── index.html        # Main beta demonstration interface
├── server.go              # Go web server for beta demo
├── README.md             # This file
├── static/               # Static web assets
│   ├── css/
│   │   └── style.css     # Custom styles and animations
│   └── js/
│       └── app.js        # Interactive JavaScript functionality
└── templates/            # HTML templates
   └── index.html        # Main beta interface
```

## Running the Beta Demo

### Makefile Targets (Web Demo)

| Target                | Description |
|-----------------------|-------------|
| `web-start`           | Start the web demo (scripts/start-web-demo.sh) |
| `web-stop`            | Stop the web demo (scripts/stop-web-demo.sh) |
| `web-restart`         | Restart the web demo (stop+start) |
3. **Access the beta interface:**
| `web-health`          | Health check for web demo (scripts/health.sh) |
| `web-tail-logs`       | Tail logs for web demo (scripts/tail-logs.sh) |
   - Beta API: `http://localhost:8080/api/v1/beta/`
   - Health check: `http://localhost:8080/api/v1/beta/health`
| `web-integration-test`| Run integration tests for web demo (integration tag) |

#### Example Usage
```bash
make web-start
- All primary API endpoints are exposed under `/api/v1/beta`. Only Beta endpoints are available. Legacy/educational endpoints have been fully removed.
make web-health
make web-logs
 - `GET /` - Minimal beta landing page
 - `GET /index.html` - Full beta demonstration interface
 - `GET /api/v1/beta/health` - System health and info
 - `GET /api/v1/beta/info` - Build and feature metadata
 - `GET /api/v1/beta/ping` - Liveness ping
 - `GET /api/v1/poa/metrics` - Power-of-attorney metrics (beta)
 - Deprecated: `GET /api/v1/educational/health|info|ping` (with deprecation headers)

For local development with hot reload:
 - `GET /api/v1/beta/examples/catalog` - List runnable example catalog entries
 - `POST /api/v1/beta/examples/run` - Queue an example execution job
 - `GET /api/v1/beta/examples/run/:id/status` - Poll job status
 - `GET /api/v1/beta/examples/run/:id/logs` - Stream job logs (SSE)
 - `GET /api/v1/beta/examples/run/jobs` - List active/running jobs
 - `POST /api/v1/beta/examples/run/jobs/:id/cancel` - Cancel a running job
 
### Prerequisites
- Go 1.21 or higher
- Gin web framework (`go get github.com/gin-gonic/gin`)
   ```bash
   cd /path/to/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0
   ```

2. **(Quick) Ephemeral run (auto-generated secrets each restart):**
   ```bash
   ```bash
   go run web/server.go 9090              # CLI arg overrides
   GAUTH_WEB_PORT=7070 go run web/server.go
   GAUTH_PORT=9091 scripts/start-web-demo.sh
   ```

3. **Access the beta interface:**
   - Open your browser to: `http://localhost:8080`
   - Beta API: `http://localhost:8080/api/v1/beta/`
   - Health check: `http://localhost:8080/api/v1/beta/health`

This beta interface is part of the broader GAuth evaluation ecosystem:
### Port Resolution Order
**Beta Demonstration Notice**: This web interface is designed exclusively for evaluation and demonstration. It implements beta demonstration versions of GAuth concepts to help users understand authorization patterns, power-of-attorney flows, and AI-native authentication. This implementation is NOT production ready and should not be used in production environments or for any security-critical applications.

### Stable vs Ephemeral Tokens
If you do not export `GAUTH_CLIENT_SECRET` / `GAUTH_SIGNING_KEY`, the server generates ephemeral values (tokens invalid after restart). Using the script + `.env` gives deterministic secrets for the session (still insecure for real use).

## Beta Learning Path

### 1. Overview Section
- Understanding GAuth framework concepts
- RFC-0150 compliance and standards
- AI-native authorization principles

### 2. Interactive Demo Tabs

#### Token Management
- Create beta tokens with RFC-compliant structure
- Validate token integrity and expiration
- Revoke tokens and manage blacklists
- **Learning Focus**: Token lifecycle, JWT patterns, security considerations

#### Authorization Flow
- Test different resource actions (read, write, admin, delegate)
- Experience power-of-attorney delegation chains
- Understand policy evaluation logic
- **Learning Focus**: RBAC/ABAC patterns, delegation flows, policy decisions

#### Event System
- Publish typed events with structured metadata
- Subscribe to event streams with pattern matching
- See real-time event handling and processing
- **Learning Focus**: Event-driven architecture, pub/sub patterns, metadata handling

#### Audit Trail
- View comprehensive audit logs from all interactions
- Generate compliance reports with detailed breakdowns
- Understand audit requirements for authorization systems
- **Learning Focus**: Compliance logging, audit patterns, regulatory requirements

### 3. Architecture Explorer
- Visual system architecture with component relationships
- Deep dive into each layer and its responsibilities
- Understanding RFC compliance implementation
- **Learning Focus**: System design, component interaction, standards compliance

### 4. Examples Repository
- Browse 37+ working code examples
- Understand implementation patterns and best practices
- See real Go code demonstrating GAuth concepts
- **Learning Focus**: Practical implementation, code patterns, best practices

## Beta API Endpoints

All API endpoints include beta warnings and are designed for learning:

### Beta API Endpoints

All API endpoints are under `/api/v1/beta`:

#### Core Endpoints
- `GET /` - Minimal beta landing page
- `GET /index.html` - Full beta demonstration interface
- `GET /api/v1/beta/health` - System health and info
- `GET /api/v1/beta/info` - Build and feature metadata
- `GET /api/v1/beta/ping` - Liveness ping
- `GET /api/v1/poa/metrics` - Power-of-attorney metrics (beta)
- `GET /api/v1/beta/examples/catalog` - List runnable example catalog entries
- `POST /api/v1/beta/examples/run` - Queue an example execution job
- `GET /api/v1/beta/examples/run/:id/status` - Poll job status
- `GET /api/v1/beta/examples/run/:id/logs` - Stream job logs (SSE)
- `GET /api/v1/beta/examples/run/jobs` - List active/running jobs
- `POST /api/v1/beta/examples/run/jobs/:id/cancel` - Cancel a running job

## Technology Stack

### Backend
- **Go**: Primary implementation language
- **Gin Web Framework**: HTTP routing and middleware

### Frontend  
- **HTML5**: Modern semantic markup
- **Tailwind CSS**: Utility-first styling framework
- **Vanilla JavaScript**: Interactive functionality without heavy frameworks
- **Font Awesome**: Professional icons and visual elements







## Customization and Extension

### Adding New Demos
1. **Add new tab in HTML**: Extend the tab system in `templates/index.html`
2. **Implement JavaScript functions**: Add interactive functionality in `static/js/app.js`
3. **Create API endpoints**: Add beta endpoints in `server_clean.go`
4. **Update styling**: Extend CSS in `static/css/style.css`





1. **Report Issues**: Use GitHub issues for beta content problems
2. **Suggest Improvements**: Propose better beta experiences or explanations
3. **Add Examples**: Contribute new beta examples or use cases
4. **Enhance Documentation**: Improve explanations and learning materials

## Related Resources

### Documentation
- [Main Project README](../README.md) - Overall project documentation
- [Architecture Guide](../docs/ARCHITECTURE.md) - Detailed system architecture
- [API Reference](../docs/API_REFERENCE.md) - Complete API documentation
- [Examples Repository](../examples/) - 37+ working code examples

### RFC Standards
- **GiFo-RFC-0111**: Power of Attorney Framework
- **GiFo-RFC-0115**: Authorization Implementation
- **GiFo-RFC-0150**: Go Implementation Guidelines

### External Links
- [Gimel Foundation](https://gimelfoundation.com) - Organization behind GAuth
- [GitHub Repository](https://github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0) - Source code
- [RFC Repository](https://github.com/Gimel-Foundation/RFCs) - Official RFC documents

---

**Beta Implementation Notice**: This web interface is designed exclusively for evaluation and demonstration. It implements beta versions of GAuth concepts to help users understand authorization patterns, power-of-attorney flows, and AI-native authentication. This implementation is NOT production ready and should not be used in production environments or for any security-critical applications.

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
