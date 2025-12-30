---
title: AgentAuth 1.0 README
category: overview
status: completed
lastUpdated: 2025-12-30
owners: core-maintainers
---
# AgentAuth - The Protocol for Agentic Identity

[![CI Build](https://github.com/mauriciomferz/AgentAuth/actions/workflows/ci.yml/badge.svg)](https://github.com/mauriciomferz/AgentAuth/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## What is AgentAuth?

**AgentAuth** is an open-source authorization framework designed specifically for the **Agentic Economy**. It provides a "Legal Layer" for AI Agents, enabling them to carry verifiable, cryptographically signed authority when negotiating, transacting, or operating across systems.

> [!NOTE]
> **Independent Standard**: AgentAuth implements the **AAP-001 (Agent Identity)** and **AAP-002 (Delegation)** protocols. It is a distinct alternative to OAuth 2.0, optimized for machine-to-machine liability chains rather than user-to-app consent.

## Key Features

1.  **🤖 Agent Identity (AAP-001)**: Agents aren't just rows in a database. They possess self-sovereign cryptographic identities (Ed25519/ECDSA) wrapped in verifiable metadata profiles.
2.  **📜 Delegation Chains (AAP-002)**: Implements "Power of Attorney" for software. A CEO can authorize a CFO, who authorizes a Manager, who creates a limited-scope token for a specific Billing Bot.
3.  **⚖️ Fiduciary Logic**: Tokens can embed "Duty of Care" logic, such as:
    *   **Liability Caps**: "Max spend $5,000/day".
    *   **Dual Control**: "Transactions >$10k require a human Co-Signer".
    *   **Kill Switches**: Revocation triggers based on market data or external oracles.
4.  **🌍 Multi-Jurisdiction**: Architecture supports "Legal Adapters" to map generic authority claims to specific local laws (e.g., eIDAS in EU, Common Law in US).
5.  **🌑 Degraded Mode**: Operating without a central database ("Headless Authority") for high-resilience environments like military or industrial IoT.

## Comparison: OAuth 2.0 vs AgentAuth

| Feature | OAuth 2.0 | AgentAuth |
| :--- | :--- | :--- |
| **Primary Unit** | User Consent | Legal Authority |
| **Token Type** | Bearer Access Token | Proof-of-Authorization (PoA) |
| **Delegation** | Flat (User -> App) | Chained (A -> B -> C -> D) |
| **Validation** | Centralized Introspection | Decentralized Crypto & Chain Verification |
| **Use Case** | "Login with Google" | "Agent X negotiates Contract Y" |

## Getting Started

### Prerequisites
*   Go 1.22+
*   Docker & Docker Compose

### Quick Start
```bash
# Clone the repository
git clone https://github.com/mauriciomferz/AgentAuth.git
cd AgentAuth

# Start the full stack (Database, Redis, AgentAuth Server)
docker-compose up -d --build

# Check health
curl http://localhost:8080/healthz
```

## Architecture

AgentAuth is built as a modular Go application:
*   `web/`: Core server logic (Gin framework).
*   `pkg/identity`: Cryptographic identity primitives.
*   `pkg/delegation`: PoA token construction and validation logic.
*   `pkg/adapter`: Connector framework for diverse legal jurisdictions.

## License

This project is licensed under the **MIT License**. See [LICENSE](LICENSE) for details.

This is a community-driven open source project. It is not affiliated with any specific proprietary foundation.
