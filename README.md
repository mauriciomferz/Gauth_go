---
title: AgentAuth 1.0 README
category: overview
status: completed
lastUpdated: 2025-12-31
owners: core-maintainers
---
# AgentAuth - The Trust Layer for Autonomous AI Agents

[![CI Build](https://github.com/mauriciomferz/AgentAuth/actions/workflows/ci.yml/badge.svg)](https://github.com/mauriciomferz/AgentAuth/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)](https://github.com/mauriciomferz/AgentAuth/releases/tag/v1.0.0)

**Author**: Mauricio A. Fernandez Fernandez

---

## What is AgentAuth?

**AgentAuth** is an open-source authorization framework designed for the **Agentic Economy**. It provides a "Legal Layer" for AI Agents, enabling them to carry verifiable, cryptographically signed authority when negotiating, transacting, or operating across systems.

> [!IMPORTANT]
> **AgentAuth v1.0.0 is GOLD MASTER** - Production-ready with 100% AAP-001/AAP-002 compliance.

> [!NOTE]
> **Independent Standard**: AgentAuth implements **AAP-001 (Agent Identity)** and **AAP-002 (Delegation)** protocols. It is a distinct alternative to OAuth 2.0, optimized for machine-to-machine liability chains rather than user-to-app consent.

---

## The Problem

**By 2030, AI agents will initiate $25 trillion in transactions annually.** Yet OAuth 2.0—designed for "Login with Google"—gives agents **access**, not **authority**.

**Real Incident**: An AI procurement agent with valid OAuth credentials ordered $2.3M from a sanctioned supplier. Result: **$18M fine**.

OAuth checked: "Does this token have the right scope?" ✅  
OAuth didn't check: "Is this supplier sanctioned?" ❌

---

## The Solution: Proof of Authorization

AgentAuth issues **PoA (Proof of Authorization) tokens** with:

- 📜 **Delegation Chains**: Traceable from board resolution to agent
- 💶 **Liability Caps**: "Max €50K per transaction"
- 🌍 **Jurisdiction Support**: German, US, EU legal frameworks
- ⚡ **Offline Verification**: Works when auth server is down (Degraded Mode)
- 🔐 **Cryptographic Audit**: Merkle tree-based revocation with inclusion proofs

---

## Key Features

### 1. Agent Identity (AAP-001)
Agents possess self-sovereign cryptographic identities (Ed25519/ECDSA) wrapped in verifiable metadata profiles.

### 2. Delegation Chains (AAP-002)
Implements "Proof of Authorization" for software. CEO → CFO → Manager → Agent, each cryptographically signed.

### 3. Fiduciary Logic
Tokens embed business rules:
- **Liability Caps**: "Max spend €50K/day"
- **Dual Control**: "Transactions >€100K require 2-of-3 signatures"
- **Jurisdiction**: "No payments to sanctioned countries"
- **Time Bounds**: "Valid Mon-Fri 08:00-18:00 CET"

### 4. Multi-Jurisdiction Support
Legal adapters for eIDAS (EU), GmbHG (Germany), Apparent Authority (US).

### 5. Degraded Mode
Offline authorization for edge devices, military, industrial IoT.

---

## Comparison: OAuth 2.0 vs AgentAuth

| Feature | OAuth 2.0 | AgentAuth |
| :--- | :--- | :--- |
| **Primary Unit** | User Consent | Legal Authority |
| **Token Type** | Bearer Access Token | Proof-of-Authorization (PoA) |
| **Delegation** | Flat (User → App) | Chained (A → B → C → D) |
| **Validation** | Centralized | Cryptographic + offline |
| **Constraints** | Application layer | Protocol layer |
| **Use Case** | "Login with Google" | "Agent signs binding contract" |

---

## Quick Start

### Prerequisites
- Go 1.25+
- Docker & Docker Compose (optional)
- PostgreSQL 15+ (optional, for persistence)

### Installation

```bash
# Clone the repository
git clone https://github.com/mauriciomferz/AgentAuth.git
cd AgentAuth

# Option 1: Docker (recommended)
docker-compose up -d --build

# Option 2: Local development
go mod download
# Run with development security settings and a demo signing key
AGENTAUTH_JWT_SIGNING_KEY=devkey_must_be_long_enough_for_32_bytes_12345678901234567890 \
GO_ENV=development \
go run cmd/web-server/main.go

# Check health
curl http://localhost:8080/api/v1/beta/health
```

### First PoA Token

```bash
# Issue a PoA token
curl -X POST http://localhost:8080/api/v1/poa/issue \
  -H "Content-Type: application/json" \
  -d '{
    "principal": "urn:agentauth:org:acme",
    "agent": "urn:agentauth:agent:procurement-bot",
    "actions": ["procurement:create"],
    "constraints": {
      "liability_cap": {"amount": 50000, "currency": "EUR"}
    }
  }'
```

---

## Architecture

AgentAuth is built as a modular Go application:

```
AgentAuth/
├── cmd/              # CLI and server entry points
├── web/              # REST API handlers (Gin framework)
├── pkg/
│   ├── agentauth/    # Core authorization engine
│   ├── poa/          # PoA token management
│   ├── delegation/   # Chain validation
│   ├── crypto/       # Key management, signing
│   └── audit/        # Cryptographic audit trails
└── docs/             # 650+ documentation files
```

---

## Documentation

### 📚 Books & Guides
- **[The Agent's Signature](docs/strategy/BOOK_MANUSCRIPT_ILLUSTRATED.md)** (1,360 lines, 22+ diagrams)
  - Comprehensive technical & legal guide
  - Real-world case studies
  - Multi-jurisdiction analysis
- **[Architecture Guide](docs/ARCHITECTURE.md)** - System design deep-dive
- **[Getting Started](docs/GETTING_STARTED.md)** - Developer quickstart

### 🚀 Marketing Materials
- **[Launch Blog Post](docs/marketing/BLOG_LAUNCH.md)** - $18M case study
- **[One-Pager](docs/marketing/ONE_PAGER.md)** - PDF-ready overview
- **[Social Media](docs/marketing/SOCIAL_MEDIA_CONTENT.md)** - Twitter, LinkedIn, YouTube
- **[Marketing Guide](docs/marketing/README.md)** - Complete launch strategy

### 📋 Release Notes
- **[v1.0.0 Release Notes](docs/release_notes/v1.0.0.md)** - Gold Master status

---

## Use Cases

### Financial Services
Trading algorithms with liability caps, dual-signature requirements, volatility-based constraints.

### Supply Chain
Autonomous procurement with supplier verification, sanction screening, spending limits.

### Healthcare
AI medical advocates with authorization chains to legal guardians, audit trails.

### Industrial IoT
Edge agents operating in contested networks with offline verification.

---

## Project Status

| Metric | Status |
|--------|--------|
| **Version** | v1.0.0 (Gold Master) ✅ |
| **Standards** | AAP-001 ✅ AAP-002 ✅ |
| **Test Coverage** | 80%+ core packages |
| **Build** | Passing |
| **License** | Apache 2.0 (Open Source) |
| **Documentation** | 650+ markdown files |
| **Marketing** | Ready for launch |

---

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Priority Areas
- Python/JavaScript SDKs
- eIDAS integration
- Post-quantum cryptography
- Additional jurisdiction adapters
- Performance optimization

---

## Community

- **GitHub**: [Issues](https://github.com/mauriciomferz/AgentAuth/issues) | [Discussions](https://github.com/mauriciomferz/AgentAuth/discussions)
- **Book**: "The Agent's Signature: Identity & Law in the Age of AI"
- **Author**: Mauricio A. Fernandez Fernandez

---

## License

This project is licensed under the **Apache License 2.0**. See [LICENSE](LICENSE) for details.

**AgentAuth** is a community-driven open-source project, not affiliated with any proprietary foundation.

---

*"Stop giving your agents keys. Start giving them mandates."*

**AgentAuth v1.0.0 - The Trust Layer for the Agentic Economy**
