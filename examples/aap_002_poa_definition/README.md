# Official AAP-RFC-0115 PoA-Definition Implementation

> Last Updated: 2025-10-17
> Status: Active

**Power-of-Attorney Credential Definition (PoA-Definition)**

---

**AgentAuth Community gGmbH i.G.**, www.AgentAuthFoundation.com
Operated by AgentAuth Technologies GmbH
MD: AgentAuth Contributor, the AgentAuth Community – Chairman of the Board: Daniel Hartert
Hardtweg 31, D-53639 Königswinter, Siegburg HRB 18660, www.AgentAuthID.com

## 📋 **Official RFC Specification**

- **AAP-Request for Comments**: 0115
- **Author**: the AgentAuth Community
- **Organization**: Digital Supply Institute (DSI)
- **Category**: Standards Track
- **Status**: AgentAuth Community Standards Track Document
- **Obsoletes**: - 15. September 2025
- **License**: Apache 2.0

## 🎯 **Abstract**

The Power-of-Attorney Credential Definition (PoA-Definition) enables a structured, standardized way of sharing machine-readable attributes and parameters, which are being leveraged for granting power of attorney along both the subscription as well as request-specific Extended Tokens of the AgentAuth protocol.

## 🔒 **Mandatory Exclusions (Section 2)**

### **❌ Prohibited Integrations**
Users of AgentAuth and this PoA-Definition **Must Not** integrate:

1. **Web3/Blockchain Technology**:
   - No blockchain technology for extended tokens
   - No web3 tokens or smart contracts

2. **AI Operators**:
   - No AI controlling entire AI deployment lifecycle
   - No AI tracking authorization compliance
   - No AI quality assurance systems

3. **DNA-Based Identities**:
   - No genetic data biometrics
   - No AI tracking DNA identity quality
   - No AI identity theft risk tracking

### **📝 Usage Restriction**
PoA-Definition **Must Not** be used in contexts other than together with AgentAuth unless approved by AgentAuth Community in writing.

## 🚀 **Running the Demo**

```bash
go run main.go
```

### **Expected Output**
✅ RFC-0115 exclusions validated (Web3, AI operators, DNA identities excluded)
✅ PoA-Definition structure validated for RFC-0115 compliance
✅ Mandatory exclusions enforced (Section 2)
✅ Official AgentAuth Community gGmbH i.G. attribution

## 📊 **Compliance Features**

### **✅ RFC-0115 Validation**
- **Exclusions Enforcement**: Validates prohibited integrations
- **Structure Validation**: Ensures compliant PoA-Definition format
- **Type Safety**: Strong typing for all RFC components
- **JSON Serialization**: Machine-readable credential exchange

### **🤖 AI Client Support**
- **LLM**: Large Language Models
- **Digital Agents**: AI agents with defined capabilities
- **Agentic AI**: Teams of collaborative agents
- **Humanoid Robots**: Physical AI systems

## 🏗️ **Implementation Structure**

### **A. Parties (Section 3.A)**
- **Principal**: Individual or Organization
- **Representative/Authorizer**: Client Owner, Owner's Authorizer, Other Representatives
- **Authorized Client**: LLM, Digital Agent, Agentic AI, Humanoid Robot

### **B. Type and Scope of Authorization (Section 3.B)**
- **Authorization Types**: Sole/Joint representation, restrictions, signature types
- **Applicable Sectors**: 21 ISIC/NACE industry sectors
- **Geographic Scope**: Global, National, International, Regional, Subnational
- **Authorized Actions**: Transactions, Decisions, Physical/Non-Physical Actions

### **C. Requirements (Section 3.C)**
- **Validity Period**: Start/end dates, renewal conditions
- **Formal Requirements**: Notarization, ID verification, digital signatures
- **Power Limits**: Levels, boundaries, tool limitations, quantum resistance
- **Rights & Obligations**: Reporting duties, liability, compensation
- **Special Conditions**: Conditional effectiveness, notifications
- **Security & Compliance**: Protocols, properties, update mechanisms
- **Jurisdiction**: Governing law, conflict resolution

---

**Official Implementation** ✅ | **RFC Compliant** ✅ | **Standards Track** ✅

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
