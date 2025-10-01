# 📁 Project Structure

This document describes the organized structure of the Gauth_go project after comprehensive cleanup and reorganization.

## 🏗️ **Root Directory Structure**

```
Gauth_go/
├── 📁 archive/           # Historical documentation and archived artifacts
├── 📁 build/             # Build artifacts and compiled binaries
│   └── 📁 bin/          # All compiled executables (16+ binaries)
├── 📁 cmd/               # Command-line applications and entry points
├── 📁 docker/            # All Docker configurations and compose files
├── 📁 docs/              # Project documentation (organized by category)
├── 📁 examples/          # Example implementations and usage demonstrations
├── 📁 gauth-demo-app/    # Complete web application with demos
├── 📁 internal/          # Private application code
├── 📁 k8s/               # Kubernetes deployment configurations
├── 📁 logs/              # Log files and runtime logs
├── 📁 monitoring/        # Monitoring and observability configurations
├── 📁 pkg/               # Public library code
├── 📁 reports/           # SARIF reports and analysis results
├── 📁 scripts/           # All build, deployment, and utility scripts
├── 📁 test/              # Test files and test data
├── 📁 .github/           # GitHub workflows and templates
├── 📄 go.mod             # Go module definition
├── 📄 go.sum             # Go module checksums
├── 📄 Makefile           # Build automation
├── 📄 README.md          # Project overview and quick start
├── 📄 CHANGELOG.md       # Version history and changes
├── 📄 CONTRIBUTING.md    # Contribution guidelines
├── 📄 GETTING_STARTED.md # Quick start guide
├── 📄 LICENSE            # Apache 2.0 license
├── 📄 LIBRARY.md         # Library documentation
├── 📄 SECURITY.md        # Security policy and reporting
├── 📄 PROJECT_STRUCTURE.md # This file
└── 📄 CLEANUP_SUMMARY.md   # Cleanup and reorganization summary
```

## 📚 **Documentation Structure** (`docs/`)

```
docs/
├── 📁 architecture/     # System design and architecture docs
│   ├── ARCHITECTURE.md
│   ├── PACKAGE_STRUCTURE.md
│   ├── PROPOSED_STRUCTURE.md
│   ├── TYPE_SAFETY.md
│   └── TYPED_*.md
├── 📁 development/      # Development guides and processes
│   ├── CODE_*.md
│   ├── DEVELOPMENT.md
│   ├── TESTING.md
│   └── TROUBLESHOOTING.md
├── 📁 guides/           # User and developer guides
│   ├── EXAMPLES.md
│   ├── GETTING_STARTED.md
│   └── PATTERNS_GUIDE.md
├── BENCHMARKS.md        # Performance benchmarks
├── EVENT_SYSTEM*.md     # Event system documentation
├── IMPROVEMENTS.md      # Planned improvements
├── LIBRARY.md          # Library usage guide
├── MANUAL_TESTING.md   # Manual testing procedures
├── ORGANIZATION.md     # Project organization
├── PERFORMANCE.md      # Performance documentation
└── WEB_APP_README.md   # Web application documentation
```

## 🛠️ **Scripts Directory** (`scripts/`)

```
scripts/
├── check-ci-status.sh          # CI/CD status checking
├── cleanup.sh                  # Project cleanup utilities
├── create-web-app-package.sh   # Web app packaging
├── docker-build-robust.sh      # Robust Docker builds
├── docker-build-test.sh        # Docker build testing
├── publish-to-gimel-app-0001.sh # Publication script
├── test-docker-build.sh        # Docker build validation
└── verify-security-fix.sh      # Security verification
```

## 📦 **Package Structure** (`pkg/`)

```
pkg/
├── audit/         # Audit trail and logging
├── auth/          # Authentication mechanisms
├── authz/         # Authorization logic
├── common/        # Shared utilities
├── errors/        # Error handling
├── events/        # Event system
├── gauth/         # Core GAuth implementation
├── mesh/          # Service mesh integration
├── metrics/       # Metrics collection
├── monitoring/    # Monitoring utilities
├── rate/          # Rate limiting
├── resilience/    # Resilience patterns
├── resources/     # Resource management
├── store/         # Data storage
├── token/         # Token handling
├── tokenstore/    # Token persistence
├── types/         # Type definitions
└── util/          # General utilities
```

## 🎯 **Demo Applications** (`gauth-demo-app/`)

```
gauth-demo-app/
├── web/
│   ├── backend/           # Go backend services
│   ├── rfc111-benefits/   # RFC111 benefits demo
│   ├── rfc111-rfc115-paradigm/ # RFC115 paradigm demo
│   └── index.html         # Main web interface
├── API_REFERENCE.md       # API documentation
├── README.md             # Demo app documentation
└── *.sh                  # Deployment scripts
```

## 🗄️ **Archive Directory** (`archive/`)

Contains historical documentation, status reports, and project artifacts that are no longer actively used but preserved for reference:

- CI/CD resolution reports
- Publication status documents  
- Security fix reports
- Implementation success records
- Deployment summaries

## 🚀 **Quick Navigation**

| **I want to...** | **Go to...** |
|-------------------|--------------|
| **Start using GAuth** | [`docs/guides/GETTING_STARTED.md`](docs/guides/GETTING_STARTED.md) |
| **Understand the architecture** | [`docs/architecture/`](docs/architecture/) |
| **See examples** | [`examples/`](examples/) |
| **Run the demo** | [`gauth-demo-app/`](gauth-demo-app/) |
| **Contribute** | [`CONTRIBUTING.md`](CONTRIBUTING.md) |
| **Build the project** | [`Makefile`](Makefile) |
| **Deploy with Docker** | [`Dockerfile`](Dockerfile) |

## 🔧 **Build Commands**

```bash
# Build all components
make build

# Run tests
make test

# Start demo application
cd gauth-demo-app/web && ./deploy.sh

# Build Docker image
docker build -t gauth .

# Run with Docker Compose
docker-compose up
```

---

**📝 Note**: This structure was reorganized on September 28, 2025, to improve project maintainability and developer experience. Historical files have been preserved in the `archive/` directory.