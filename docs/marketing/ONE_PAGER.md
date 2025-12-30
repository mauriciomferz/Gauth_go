# AgentAuth: One-Page Overview
## The Trust Layer for Autonomous AI Agents

---

## The Problem

**By 2030, AI agents will initiate $25 trillion in transactions annually. Yet our authorization infrastructure treats them like web apps from 2005.**

### The OAuth Trap

OAuth 2.0 solved "delegated access" for humans clicking buttons. But autonomous AI agents need **authority**, not just access.

| What OAuth Answers | What AI Agents Need |
|-------------------|---------------------|
| "Can this app read my email?" | "Can this agent bind my company to a contract?" |
| Access token validity | Legal accountability |
| Scope matching | Fiduciary duty enforcement |

### Real Consequences

**Case Study**: A Fortune 500 procurement AI with valid OAuth credentials ordered $2.3M in chemicals from Belarus (sanctioned).

- **Technical authorization**: ✅ Valid token, valid scope
- **Legal authorization**: ❌ Violated OFAC sanctions
- **Result**: $18M fine, criminal investigation

---

## The Solution: AgentAuth

**AgentAuth** is an open-source authorization framework that treats AI agents as **legal entities with fiduciary duties**.

### Proof of Authorization (PoA) Tokens

Instead of simple Bearer tokens, AgentAuth issues cryptographically signed mandates:

```json
{
  "principal": "Acme Corporation (DE:HRB:123456)",
  "agent": "urn:agentauth:agent:procurement-bot-7",
  "authorization": {
    "actions": ["procurement:create", "payment:domestic"],
    "resources": ["office_supplies", "it_equipment"]
  },
  "constraints": {
    "liability_cap": "€50,000 per transaction",
    "daily_limit": "€200,000",
    "valid_hours": "Mon-Fri 08:00-18:00 CET",
    "excluded_jurisdictions": ["BY", "KP", "IR", "SY"]
  },
  "delegation_chain": [
    {"from": "Board", "to": "CFO", "type": "statutory"},
    {"from": "CFO", "to": "Agent", "type": "delegated"}
  ]
}
```

---

## Key Features

### 1. Cryptographic Delegation Chains
Every authorization traces to root authority (e.g., board resolution) via verifiable signatures.

### 2. Offline Verification (Degraded Mode)
Authorization rules embedded in token → works when auth server is down. Critical for industrial/military edge.

### 3. Multi-Jurisdiction Support
- **Germany**: Commercial Register (Handelsregister) integration, GmbHG §35 compliance
- **United States**: Apparent Authority doctrine, electronic signatures
- **European Union**: eIDAS trust levels, cross-border recognition

### 4. Smart Constraints
- Liability caps per transaction/period
- Time-of-day restrictions
- Geographic boundaries
- Dual-control thresholds
- Supplier whitelists/blacklists

### 5. Revocation Transparency
Merkle tree-based revocation with cryptographic inclusion proofs. Cascade revocation automatically revokes all derived delegations.

---

## Architecture

```
┌─────────────┐
│ AI Agent    │ Presents PoA Token
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────┐
│ Policy Enforcement Point (PEP)  │
│                                 │
│ ✓ Signature verification        │
│ ✓ Revocation check              │
│ ✓ Chain validation              │
│ ✓ Constraint matching           │
│ ✓ Jurisdiction compliance       │
└──────┬──────────────────────────┘
       │
       ▼
┌─────────────┐
│ Decision:   │ PERMIT / DENY + Audit
└─────────────┘
```

---

## Technical Specifications

| Feature | Implementation |
|---------|---------------|
| **Cryptography** | EdDSA (Ed25519), ECDSA (P-256/384), BLS12-381 aggregate |
| **Revocation** | Merkle trees with SHA-256, O(log n) inclusion proofs |
| **Backend** | Go 1.25+, 80%+ test coverage |
| **Storage** | PostgreSQL (policies), Redis (cache) |
| **Observability** | Prometheus metrics, OpenTelemetry traces, structured logs |
| **Standards** | AAP-001 (Identity), AAP-002 (Delegation) |

---

## Use Cases

### Financial Services
**Challenge**: Trading algorithms with market access
**Solution**: Liability caps, dual-signature requirements, volatility-based constraints

### Supply Chain
**Challenge**: Autonomous procurement across jurisdictions
**Solution**: Supplier verification, sanction screening, spending limits

### Healthcare
**Challenge**: AI medical advocates making treatment decisions
**Solution**: Authorization chains to legal guardians, time-bound mandates, audit trails

### Industrial IoT
**Challenge**: Edge agents operating in contested networks
**Solution**: Offline verification (Degraded Mode), fail-safe policies

---

## Getting Started

```bash
# Clone repository
git clone https://github.com/mauriciomferz/AgentAuth

# Install dependencies
go mod download

# Run server
go run cmd/server/main.go

# Generate PoA token
curl -X POST http://localhost:8080/api/v1/poa/issue \
  -H "Content-Type: application/json" \
  -d @examples/poa-request.json
```

---

## Project Status

| Metric | Value |
|--------|-------|
| **Version** | v1.0.0 (Gold Master) |
| **License** | MIT (Open Source) |
| **Test Coverage** | 80%+ across core packages |
| **Standards Compliance** | AAP-001 ✅ AAP-002 ✅ |
| **Documentation** | 1,360-page technical book |

---

## Resources

📖 **Technical Book**: "The Agent's Signature: Identity & Law in the Age of AI"  
🔗 **GitHub**: github.com/mauriciomferz/AgentAuth  
📰 **Blog**: "Why OAuth Isn't Enough for AI Agents"  
📧 **Contact**: mauricio@agentauth.org

---

## The Bottom Line

**Stop giving your agents keys. Start giving them mandates.**

OAuth grants access. AgentAuth grants authority.

The Agentic Economy needs legal infrastructure, not just API infrastructure.

AgentAuth is production-ready. Open source. Jurisdiction-aware.

*Join us in building the trust layer for the age of autonomous AI.*

---

**Author**: Mauricio A. Fernandez Fernandez  
**Date**: December 31, 2025  
**License**: MIT
