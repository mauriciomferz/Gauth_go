# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.5] - 2025-09-25
### Security
- **ZERO VULNERABILITIES ACHIEVED** - Complete security audit resolution
- Fixed G115 (High): Integer overflow prevention in exponential backoff
- Fixed G304 (Medium): File inclusion attack prevention with path validation
- Fixed G101 (High): Hardcoded credentials resolution with proper annotations
- Gosec scan: 0 issues across 303 files (44,032 lines)

### Changed
- Enhanced file operations with comprehensive path validation
- Improved security annotations for test/demo code patterns
- Strengthened bounds checking in retry mechanisms

## [1.0.4] - 2025-09-24
### Security
- Enhanced credential management with environment variables
- Improved HTTP timeout settings for production stability
- File permissions security updates (0600 for sensitive files)
- Crypto/rand usage improvements across examples

### Added
- Production-ready HTTP timeout configurations
- Enhanced credential management patterns
- Improved error handling and recovery mechanisms

### Fixed
- Build issues with empty Go files
- Module dependency resolution
- Example applications stability improvements

## [1.0.3] - 2025-09-23
### Added
- Enhanced interactive web application with real-time features
- WebSocket support for live event streaming
- Modern glassmorphism UI design
- Mobile-responsive interface
- Live system metrics dashboard

### Improved
- Token management with real-time validation
- API documentation and endpoint coverage
- User experience with keyboard shortcuts and animations
- Progressive enhancement for better accessibility

## [1.0.2] - 2025-09-22
### Added
- Complete demo web applications for RFC111 and RFC115
- Production deployment configurations
- Docker and Kubernetes support
- Comprehensive monitoring and observability

### Fixed
- Go module resolution issues
- Build process improvements
- Documentation consistency updates

## [1.0.1] - 2025-09-21
### Fixed
- Initial bug fixes and stability improvements
- Documentation updates and corrections
- Build process optimizations

## [1.0.0] - 2025-09-13
### Added
- Initial open-source release of GAuth
- Modular Go library with clear separation of core, demo, and internal code
- Strong type safety: No public `map[string]interface{}` in APIs
- Comprehensive onboarding: `README.md`, `GETTING_STARTED.md`, `LIBRARY.md`, and package-level docs
- Demo apps in `/demo` and `/cmd/demo`
- Audit and event system for protocol traceability
- RFC111 compliance: protocol steps, roles, and exclusions mapped to code
- GitHub Actions CI and release automation
- Community contribution guidelines
