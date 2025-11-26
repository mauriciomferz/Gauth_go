# GAuth Internal Standards - Namespace Clarification

**Effective Date**: November 26, 2025  
**Version**: 1.0

---

## Important Notice: Not IETF RFCs

The standards **GAuth-RFC-001** and **GAuth-RFC-002** are **internal specifications** developed by the Gimel Foundation for the GAuth authorization framework. These are **NOT** Internet Engineering Task Force (IETF) Request for Comments (RFC) documents.

### Namespace Clarification

**Historical Context**: Earlier versions of this documentation referenced "GAuth-RFC-001" and "GAuth-RFC-002" without proper namespace qualification. This created potential confusion with existing IETF standards:

- **IETF GAuth-RFC-001 (formerly RFC 111)**: "Standard Host Names" (1971) - Network Control Protocol
- **IETF GAuth-RFC-002 (formerly RFC 115)**: "Some Network Information Center Clerks Should Be Told About Network Procedures" (1971)

To eliminate this collision and ensure clarity for external auditors, compliance officers, and integration partners, we have renamed our internal standards:

| Old Reference | New Standard | Status |
|--------------|--------------|---------|
| GAuth-RFC-001 | **GAuth-RFC-001** | Active |
| GAuth-RFC-002 | **GAuth-RFC-002** | Active |

---

## GAuth-RFC-001: Core Authentication & Authorization Protocol

**Formerly**: GAuth-RFC-001 (internal designation)  
**Current Name**: GAuth-RFC-001  
**Scope**: Core authentication, authorization, and Power-of-Attorney (PoA) issuance protocol

### Key Features

1. **JWT-Based Authorization**
   - Cryptographic proof of delegation
   - Time-bound permissions
   - Scope constraints

2. **Hierarchical Delegation**
   - Principal → Agent delegation chains
   - Scope narrowing enforcement
   - Revocation propagation

3. **Audit Trail**
   - Immutable operation logs
   - Cryptographic verification
   - Compliance reporting

### Implementation

- **Package**: `github.com/mauriciomferz/Gauth_go/pkg/gauth_rfc_001`
- **Specification**: `docs/specifications/GAuth-RFC-001.md`
- **Status**: Production-ready
- **Version**: 1.0.0

---

## GAuth-RFC-002: Extended Delegation & Policy Enforcement

**Formerly**: GAuth-RFC-002 (internal designation)  
**Current Name**: GAuth-RFC-002  
**Scope**: Advanced delegation policies, semantic allow-lists, and policy enforcement

### Key Features

1. **Semantic Allow-Lists**
   - Contract-specific permissions
   - Exact address matching (no wildcards)
   - Function signature constraints
   - Hard limits (max transaction value, daily limits)

2. **Policy Enforcement**
   - Real-time policy evaluation
   - Circuit breakers
   - Multi-signature requirements
   - Principal approval thresholds

3. **Delegation Guidelines**
   - Sub-delegation constraints
   - Time-to-live (TTL) limits
   - Scope inheritance rules
   - Revocation policies

### Implementation

- **Package**: `github.com/mauriciomferz/Gauth_go/pkg/gauth_rfc_002` (planned)
- **Specification**: `docs/specifications/GAuth-RFC-002.md` (planned)
- **Status**: In development
- **Version**: 0.9.0 (beta)

---

## Compliance & Audit Guidance

### For External Auditors

When reviewing GAuth documentation references to "RFC compliance," please note:

✅ **GAuth-RFC-001/002**: Internal GAuth standards (this framework)  
❌ **IETF GAuth-RFC-001 (formerly RFC 111)/115**: Historic IETF network protocols (not related)

### Verification Questions

**Q**: "Does this system implement IETF GAuth-RFC-002 (formerly RFC 115)?"  
**A**: No. This system implements **GAuth-RFC-001** and (planned) **GAuth-RFC-002**, which are internal authorization standards developed by Gimel Foundation. These are not IETF standards.

**Q**: "How do we verify GAuth-RFC-001 compliance?"  
**A**: Review the specification at `docs/specifications/GAuth-RFC-001.md` and compare against the implementation in `pkg/gauth_rfc_001/`. The test suite provides comprehensive compliance verification.

**Q**: "What is the relationship to IETF standards?"  
**A**: GAuth uses standard IETF protocols (JWT/RFC 7519, Ed25519/RFC 8032, WebAuthn/W3C) but GAuth-RFC-001/002 are framework-specific extensions. There is no connection to IETF GAuth-RFC-001 (formerly RFC 111)/115.

---

## Integration Guidelines

### For Developers

**Import Statements**:
```go
// Correct (new)
import "github.com/mauriciomferz/Gauth_go/pkg/gauth_rfc_001"

// Deprecated (old - do not use)
// import "github.com/mauriciomferz/Gauth_go/pkg/rfc0111"
```

**Documentation References**:
- Use: "GAuth-RFC-001 compliant"
- Avoid: "GAuth-RFC-001 (formerly RFC 111) compliant" (ambiguous)

### For Documentation Writers

When documenting GAuth features:

✅ **Correct**: "This module implements GAuth-RFC-001 authentication."  
❌ **Incorrect**: "This module implements GAuth-RFC-001 (formerly RFC 111) authentication."

✅ **Correct**: "Follows GAuth-RFC-001 §4.2 delegation semantics."  
❌ **Incorrect**: "Follows GAuth-RFC-001 §4.2 delegation semantics."

---

## Migration Guide

### Codebase Changes (November 26, 2025)

The following changes have been implemented:

1. **Package Rename**:
   - `pkg/rfc0111/` → `pkg/gauth_rfc_001/`
   - `pkg/rfc115/` → `pkg/gauth_rfc_002/` (when created)

2. **Import Path Updates**:
   - All Go files updated to use new import paths
   - No breaking API changes (only package names)

3. **Documentation Updates**:
   - All `.md` files updated with historical context
   - "GAuth-RFC-001 (formerly RFC 111)" → "GAuth-RFC-001 (formerly GAuth-RFC-001 (formerly RFC 111))"
   - "GAuth-RFC-002 (formerly RFC 115)" → "GAuth-RFC-002 (formerly GAuth-RFC-002 (formerly RFC 115))"

4. **Test Coverage**:
   - All tests updated and passing
   - No functional changes to implementation

### External Systems

If your system integrates with GAuth:

- ✅ **API Endpoints**: No changes (HTTP paths unchanged)
- ✅ **JWT Structure**: No changes (token format unchanged)
- ✅ **Validation Logic**: No changes (crypto unchanged)
- ⚠️  **Documentation**: Update references from "GAuth-RFC-001/115" to "GAuth-RFC-001/002"

---

## Standards Governance

### Change Control

Changes to GAuth-RFC-001/002 specifications follow this process:

1. **Proposal**: Submit GitHub issue with proposed changes
2. **Review**: Technical review by Gimel Foundation architecture team
3. **Testing**: Implementation + test coverage for proposed changes
4. **Approval**: Consensus approval from core maintainers
5. **Documentation**: Update specifications + migration guide
6. **Release**: Version bump following SemVer

### Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2025-11-26 | Renamed GAuth-RFC-001 to GAuth-RFC-001 (namespace clarification) |
| 0.9.0 | 2025-06-15 | Initial production release |

---

## Legal & Compliance

### Intellectual Property

GAuth-RFC-001 and GAuth-RFC-002 specifications are:
- **Copyright**: © 2025 Gimel Foundation
- **License**: Apache 2.0 (implementation), CC BY 4.0 (specifications)
- **Patents**: No patent claims (royalty-free implementation)

### Standards Bodies

GAuth is **not** affiliated with:
- Internet Engineering Task Force (IETF)
- World Wide Web Consortium (W3C)
- International Organization for Standardization (ISO)

GAuth **does comply** with established standards:
- IETF RFC 7519 (JWT)
- IETF RFC 8032 (Ed25519)
- W3C WebAuthn
- FIDO2 specifications

---

## Contact & Support

### Questions About Standards

- **Technical Questions**: `dev@gimel.foundation`
- **Compliance Inquiries**: `compliance@gimel.foundation`
- **Security Disclosures**: `security@gimel.foundation` (PGP: [link])

### Documentation

- **Specifications**: `docs/specifications/`
- **API Reference**: `docs/api/`
- **Migration Guides**: `docs/migrations/`

---

## Disclaimer

**This document clarifies namespace usage only.** No changes have been made to the underlying protocol, API, or cryptographic implementation. The rename from "GAuth-RFC-001/115" to "GAuth-RFC-001/002" is purely cosmetic to avoid confusion with IETF standards.

**For Auditors**: If you are conducting a compliance audit and need verification that this system meets specific requirements, please reference the **GAuth-RFC-001 specification** rather than searching for "GAuth-RFC-001 (formerly RFC 111)" (which will return unrelated IETF network protocols).

---

**Document Version**: 1.0  
**Last Updated**: November 26, 2025  
**Status**: Official
