# The Agent’s Signature
## Identity, Authorization, and Legal Authority for Autonomous Systems

**Author:** Mauricio A. Fernandez Fernandez  
**Revision:** Technical Rewrite – Working Draft  
**Audience:** Protocol designers, security engineers, cryptographers, regulators, system architects

---

### Author’s Note on Original Contributions
This manuscript proposes a novel authorization model for autonomous agents that explicitly binds cryptographic authorization to legal authority and liability. While it builds on established work in cryptography, distributed systems, and law, the following elements are original contributions of this work:

1.  **Proof of Authorization (PoA)** as a first‑class primitive distinct from access tokens and capability tokens.
2.  **Delegation chains** with legal semantics, explicitly tied to statutory and fiduciary authority.
3.  **Machine‑verifiable liability constraints** embedded in authorization artifacts.
4.  **Revocation transparency** for legal authority, inspired by but distinct from certificate transparency.
5.  A **protocol‑level bridge** between AI systems and jurisdiction‑specific legal doctrines (e.g., apparent authority, statutory representation).

All external concepts, standards, and prior art are cited explicitly using IEEE / RFC‑style numbered references.

---

# Part I – Problem Statement and Threat Model

## 1. Autonomous Action vs. Authorized Action

### 1.1 The Category Error in Modern Authorization Systems
Modern authorization systems (OAuth 2.0, API keys, IAM roles) answer the question:

> **Is this principal allowed to access this resource?**

They do not answer:

> **Is the acting entity legally empowered to bind another party?**  
> **Under which jurisdiction is this action valid?**  
> **Who bears liability if the action exceeds intent?**

This distinction mirrors the legal separation between access, agency, and authority [1], [2].

### 1.2 Threat Model
We consider adversaries capable of:
*   Compromising agent credentials
*   Replaying valid authorization artifacts
*   Exploiting over‑broad delegation
*   Exploiting jurisdictional ambiguity

We explicitly do not assume:
*   Trusted execution environments
*   Honest principals
*   Continuous online verification

---

# Part II – Related Work and Prior Art

This section surveys existing authorization, delegation, and trust frameworks relevant to autonomous agents. The purpose is twofold: (1) to establish the state of the art with precise citations, and (2) to delineate the boundaries of prior work relative to the original contributions of this manuscript.

## 2. Capability-Based Authorization Systems

### 2.1 OAuth 2.0 and Token-Based Access Control
OAuth 2.0 [3] defines a delegation framework for access to protected resources. Its security model assumes:
*   A human principal
*   An interactive consent step
*   Short-lived bearer tokens

OAuth tokens convey permission to access an API, not authority to perform legally binding acts. The specification is explicit that OAuth is not an authorization protocol in the legal sense and does not model agency, delegation chains, or liability.

This limitation is structural, not accidental. OAuth deliberately avoids semantics beyond access control to remain application-agnostic.

### 2.2 Capability Tokens: Macaroons, Biscuit, UCAN
Capability-based systems improve on OAuth by making authority attenuable and composable.
*   **Macaroons** [4] introduce first-order caveats, enabling contextual restriction of authority.
*   **Biscuit** [5] extends this model using Datalog-style logic, enabling formal verification of authorization decisions.
*   **UCAN** [6] generalizes capabilities into a decentralized, self-certifying delegation model.

These systems share important properties:
*   Cryptographic attenuation of authority
*   Offline verification
*   Fine-grained scoping

However, they intentionally treat principals as abstract cryptographic identities. They do not encode:
*   Legal personhood
*   Jurisdictional validity
*   Fiduciary duty
*   Liability attribution

As a result, while they are suitable for distributed systems authorization, they are insufficient for systems whose actions have legal or regulatory consequences.

### 2.3 Delegation and Naming Systems: SPKI/SDSI
SPKI/SDSI [7] introduced a rigorous model of delegated authority using public-key-linked names and certificates. It remains one of the most formally precise treatments of authorization.

Key contributions include:
*   Explicit delegation chains
*   Local name spaces
*   Authorization certificates distinct from identity certificates

AgentAuth is conceptually closer to SPKI/SDSI than OAuth. However, SPKI deliberately omits:
*   Revocation transparency
*   Liability semantics
*   Legal identity binding

Its revocation model relies on short-lived certificates or external distribution, which is insufficient for regulatory auditability.

## 3. Trust Transparency and Revocation Systems

### 3.1 Public Key Infrastructure and Certificate Transparency
Traditional PKI binds public keys to identities but does not express authority. Certificate Transparency (CT) [9] improves PKI by making certificate issuance publicly auditable via append-only logs.

CT provides three key properties:
1.  Append-only data structures
2.  Cryptographic inclusion proofs
3.  External verifiability

AgentAuth adopts these properties but applies them to authorization revocation, not key issuance. This distinction is critical: the subject of transparency is legal authority, not identity.

### 3.2 Software Supply Chain Trust: Sigstore
Sigstore [10] demonstrates how transparency logs can be used to bind software artifacts to identities and build trust without long-lived secrets.

While Sigstore addresses software provenance, it does not model delegation, scope, or authority to act. AgentAuth borrows the transparency pattern but applies it to agent authorization lifecycles.

## 4. Policy-Based Access Control
XACML [11] and related policy languages separate:
*   Policy decision points (PDP)
*   Policy enforcement points (PEP)
*   Policy information points (PIP)

These systems are expressive but assume that policy itself is authoritative. They do not answer *where* policy authority originates or how it maps to legal principals.

AgentAuth treats policy as *derivative* of cryptographically provable delegation, not as a root of authority.

## 5. Legal and Regulatory Foundations

### 5.1 Agency Law
Agency law formalizes the relationship between a principal and an agent [1]. Core concepts include:
*   Actual authority
*   Apparent authority
*   Scope and limitation
*   Revocation

These concepts are well-defined in law but have no native representation in contemporary authorization protocols.

### 5.2 Electronic Signatures and Trust Services
The eIDAS Regulation [12] establishes legal equivalence between qualified electronic signatures and handwritten signatures within the EU. It defines assurance levels but does not address autonomous delegation.

AgentAuth does not attempt to replace eIDAS. Instead, it composes with it by treating identity assurance as an input to authorization, not a substitute for it.

## 6. Summary of Gaps in Prior Art
Across cryptographic, systems, and legal literature, no existing framework simultaneously provides:
1.  Cryptographically verifiable delegation chains
2.  Explicit legal principal identification
3.  Machine-enforceable liability constraints
4.  Transparent, auditable revocation of authority

The remainder of this manuscript proposes such a framework.

---

# Part III – System Model and Protocol Overview

## 3. Entities and Identities (AAP‑001)
An entity is any legally or operationally distinct actor:
*   Natural person
*   Legal person (corporation, trust, state)
*   Autonomous agent

Each entity is represented by a cryptographically signed **Entity Profile** containing:
*   Public keys
*   Jurisdictional identity
*   Legal classification
*   Validity period
(Full formal definition in §3.3.)

### 3.1 Design Invariants
The system enforces the following invariants:
*   No agent may receive authority exceeding its delegator.
*   All authority must be traceable to a legally recognized root.
*   Revocation must be externally verifiable.
*   Authorization must fail closed under uncertainty.

---

# Part IV – Proof of Authorization (AAP‑002)

## 4. Proof of Authorization (PoA)
A PoA is a cryptographically signed artifact asserting that:

> **Entity A is authorized to perform action X on behalf of Entity B under constraints C and jurisdiction J.**

Formally, a PoA binds:
*   Principal identity
*   Agent identity
*   Delegation chain
*   Scope
*   Constraints
*   Validity

### 4.1 PoA vs. Access Tokens

| Property | OAuth Token | PoA |
| :--- | :--- | :--- |
| Legal authority | ✗ | ✓ |
| Delegation traceability | ✗ | ✓ |
| Liability encoding | ✗ | ✓ |
| Jurisdiction binding | ✗ | ✓ |

# Part IV‑A – Formal Proof of Authorization Specification (Normative)
This section provides a normative, protocol‑level specification of the Proof of Authorization (PoA) primitive. It refines and formalizes the descriptive overview given in Part IV and is intended to be read as a systems and protocol design document.

## 4A.1 Definition
A Proof of Authorization (PoA) is a cryptographically verifiable statement asserting that:

> A legally identifiable principal has delegated a bounded set of authorities to a specific agent, under explicit constraints, within a defined jurisdiction and validity interval.

Formally, a PoA is a signed tuple:

```
PoA := Sign_k(
    Principal_ID,
    Agent_ID,
    Delegation_Chain,
    Scope,
    Constraints,
    Jurisdiction,
    Validity
)
```
where `k` is a signing key whose authority is itself provable through the included delegation chain.

## 4A.2 Design Goals
PoA is designed to satisfy the following properties:
*   **Legal meaningfulness** – Each field corresponds to a recognizable legal concept of agency or authority.
*   **Cryptographic verifiability** – Any relying party can verify a PoA offline.
*   **Delegation traceability** – Authority is traceable to a legally competent root.
*   **Bounded liability** – Explicit constraints limit the effects of agent action.
*   **Auditability** – Authorization and revocation events are externally verifiable.

## 4A.3 Non‑Goals
PoA explicitly does not attempt to:
*   Prove agent intent or decision correctness
*   Encode contractual obligations
*   Replace judicial or statutory interpretation

PoA is an **authorization artifact, not a contract**.

## 4A.4 Principal and Agent Semantics

### Principal
A principal is an entity capable of bearing legal rights and obligations. A principal MUST be:
*   Jurisdiction‑qualified
*   Cryptographically bound to a verifiable identity profile
*   Legally competent to delegate the asserted authority

### Agent
An agent is an entity that performs actions on behalf of a principal. In this work, agents are assumed to be autonomous software systems. An agent identity MUST be:
*   Cryptographically unique
*   Non‑transferable
*   Explicitly bound to the PoA

## 4A.5 Delegation Chains
Authority in PoA flows through an explicit delegation chain:

`Delegation_Chain := [ D₀, D₁, …, Dₙ ]`

Each delegation statement `Dᵢ` asserts that the delegator possessed authority `Aᵢ` and delegated a strict subset `Aᵢ₊₁` to the next entity.

A delegation chain is valid if and only if:
1.  Each signature verifies correctly
2.  Each delegator is authorized by the previous link
3.  Authority is monotonically non‑increasing
4.  The root authority is legally competent

This model is related to, but stricter than, capability attenuation systems [4], [6].

## 4A.6 Scope Model
The scope defines the actions an agent may perform:
`Scope := { Actions, Resources, Exclusions }`
PoA does not impose a global ontology. Scope interpretation is delegated to the relying system.

## 4A.7 Constraints and Liability Encoding
Constraints bind authorization to enforceable limits, including:
*   Temporal bounds
*   Financial liability caps
*   Jurisdictional restrictions
*   Multi‑party approval thresholds

Any action violating constraints MUST be treated as unauthorized, even if scope matches.

## 4A.8 Jurisdiction Binding
Each PoA MUST include an explicit **jurisdiction identifier**. Absent jurisdiction binding, legal authority is ambiguous by construction.

## 4A.9 Validity and Expiration
PoA artifacts MUST be time‑bounded. Long‑lived authority is an anti‑pattern in autonomous systems.

## 4A.10 Revocation Semantics
Revocation invalidates a PoA prior to expiration. Verification MUST yield one of:
1.  Proof of revocation
2.  Proof of non‑revocation
3.  Explicit degraded‑mode indication

Revocation evidence MUST be externally auditable using append‑only data structures [9].

## 4A.11 Comparison to Capability Tokens
| Property | Capability Token | PoA |
| :--- | :--- | :--- |
| Legal principal | Optional | Mandatory |
| Delegation traceability | Partial | Complete |
| Liability constraints | ✗ | ✓ |
| Jurisdiction binding | ✗ | ✓ |
| Revocation transparency | Rare | Required |

PoA may be viewed as a legally constrained capability, but the reverse does not hold.

---

# Part V – Cryptographic Architecture

This section specifies the cryptographic primitives and constructions required to implement PoA securely. The goal is not novelty in cryptography, but correct composition under realistic adversarial assumptions.

## 5. Cryptographic Design Principles
The cryptographic architecture follows four principles:
1.  **Explicit trust boundaries** – No implicit trust in execution environments or network locality.
2.  **Minimal cryptographic assumptions** – Standard, well-analyzed primitives only.
3.  **Offline verifiability** – Authorization decisions must not require online lookups.
4.  **Evidence preservation** – Artifacts must remain verifiable years after issuance.

### 5.1 Signature Schemes
PoA relies on digital signatures with EUF-CMA security guarantees [8]. Acceptable schemes include:
*   Ed25519 (current default)
*   ECDSA over P-256 (interoperability)
*   CRYSTALS-Dilithium (post-quantum roadmap)

Signature keys MUST be bound to entity identities as defined in AAP-001.

### 5.2 Hash Functions and Canonicalization
All PoA fields MUST be serialized using a canonical encoding prior to signing. Hash functions MUST be collision-resistant (e.g., SHA-256 or stronger). canonicalization errors are treated as authorization failures.

### 5.3 Delegation Chain Verification
Each delegation link is verified independently. Verification MUST fail if:
*   Any signature is invalid
*   Any delegation exceeds the authority of its parent
*   Any link is expired or revoked

No delegation shortcutting is permitted.

### 5.4 Revocation Transparency Mechanism
Revocation information is published to an append-only, publicly auditable log, inspired by Certificate Transparency [9]. The log MUST provide:
*   Append-only guarantees
*   Cryptographic inclusion proofs
*   Consistency proofs

The subject of the log is **authority revocation**, not key issuance.

### 5.5 Degraded Operation
When revocation status cannot be conclusively determined, relying parties MUST enter a degraded mode that:
*   Prevents irreversible actions
*   Preserves audit evidence
*   Signals uncertainty explicitly

Fail-open behavior is forbidden for legally binding actions.

### 5.6 Cryptographic Agility
PoA structures are versioned. Algorithm agility is mandatory to allow migration without invalidating historical evidence.

---

# Part VI – Legal Mapping Across Jurisdictions

This section maps PoA semantics to established doctrines of authority and representation across major jurisdictions. The objective is not legal harmonization, but explicit alignment: PoA makes jurisdictional assumptions visible and verifiable instead of implicit.

## 6. Authority as a Legal Concept
Across jurisdictions, authority answers a distinct question from identity:

> **Who is empowered to bind whom, to what extent, and with what consequences?**

PoA treats authority as a first-class, verifiable artifact rather than an implied property of authentication.

### 6.1 Germany: Statutory Representation and Register-Based Authority
German law models authority primarily through *statutory representation* (gesetzliche Vertretung) and *register-backed mandates*.

**Key properties:**
*   Authority is often conferred by statute (e.g., managing directors of a GmbH)
*   Scope and limitations are externally visible via public registers
*   Third parties are entitled to rely on register accuracy (*Publizitätswirkung*)

Relevant sources include:
*   German Civil Code (BGB), §§ 164–181
*   Commercial Register (Handelsregister)

**Implications for PoA:**
*   Delegation chains MUST root in a legally registered authority where applicable
*   Revocation MUST be externally visible to avoid reliance conflicts
*   Over-delegation increases principal liability regardless of internal policy

PoA’s explicit delegation chain mirrors register-based authority, while revocation transparency approximates register publicity.

### 6.2 United States: Actual vs. Apparent Authority
U.S. agency law distinguishes sharply between:
*   **Actual authority** – authority intentionally granted by the principal
*   **Apparent authority** – authority reasonably perceived by third parties

Courts routinely bind principals based on apparent authority, even where internal limits were exceeded.

Representative sources include:
*   Restatement (Third) of Agency [1]
*   Case law addressing reliance and representation

**Implications for PoA:**
*   Publicly verifiable PoA artifacts reduce ambiguity about apparent authority
*   Liability caps and scope constraints must be externally observable to be effective
*   Silent or private revocation is legally insufficient

PoA’s design favors external verifiability over internal intent, aligning with U.S. reliance doctrine.

### 6.3 European Union: eIDAS and Trust Services
The eIDAS Regulation establishes a framework for:
*   Electronic identification
*   Trust services (signatures, seals, timestamps)

While eIDAS provides strong identity and signature assurances, it does not define *how* authority is delegated or limited between parties.

**Implications for PoA:**
*   eIDAS-qualified identities can serve as principals or roots of authority
*   PoA complements eIDAS by defining *what* an identified entity is authorized to do
*   Assurance level does not substitute for scope or liability modeling

### 6.4 Cross-Border and Conflict-of-Law Considerations
In cross-border systems, authority validity depends on:
*   Governing law
*   Recognized venue
*   Conflict-of-law rules

PoA requires explicit jurisdiction binding to:
*   Prevent implicit forum shopping
*   Enable ex ante risk assessment
*   Allow systems to fail closed under legal uncertainty

Absent jurisdiction binding, autonomous action creates indeterminate liability.

### 6.5 Regulatory and Compliance Implications
Explicit, machine-verifiable authority supports:
*   Auditability under financial and operational regulations
*   Clear attribution in incident response
*   Separation of technical failure from governance failure

PoA enables regulators to inspect authority evidence, not internal policy.

### 6.6 Summary

| Jurisdiction | Legal Construct | PoA Implications |
| :--- | :--- | :--- |
| **Germany** | Statutory representation | Chain must root in register |
| **United States** | Apparent authority | Over‑broad grants increase liability |
| **EU** | eIDAS trust services | Identity assurance tiers |

The protocol explicitly models these differences instead of abstracting them away.

---

# Part VII – Operational and Regulatory Considerations

## 7. Auditability and Compliance
Every authorization decision produces a verifiable audit record suitable for:
*   Regulatory inspection
*   Legal discovery
*   Internal governance

Audit records are cryptographically chained and externally anchorable.

---

# Part VIII – Non‑Goals

This system explicitly does not attempt to:
*   Prove agent intent
*   Replace contract law
*   Eliminate human accountability

Its purpose is to make authority explicit, verifiable, and bounded.

---

# Part IX – Security Considerations
This section summarizes security considerations following RFC conventions.

### 9.1 Key Compromise
Compromise of an agent key does not imply compromise of the principal. Damage is bounded by:
*   Scope restrictions
*   Temporal limits
*   Liability caps

Rapid revocation and transparent evidence are mandatory responses.

### 9.2 Over-Delegation Risk
Excessive delegation increases systemic risk. Systems SHOULD:
*   Prefer narrow scopes
*   Use short validity intervals
*   Require explicit human approval for high-impact authority

### 9.3 Replay and Reuse
PoA artifacts MAY be replayed within their validity window. Relying systems MUST ensure idempotency or contextual binding where required.

### 9.4 Jurisdictional Ambiguity
Absent or conflicting jurisdiction identifiers MUST cause authorization failure. Silent fallback is prohibited.

### 9.5 Transparency Log Attacks
Log equivocation or withholding undermines revocation guarantees. Independent monitoring is REQUIRED for high-assurance deployments.

---

# References

[1] Restatement (Third) of Agency, American Law Institute, 2006.

[2] F. H. Easterbrook, “Agency Law and Contractual Authority,” University of Chicago Law Review, vol. 70, no. 2, pp. 375–399, 2003.

[3] D. Hardt, “The OAuth 2.0 Authorization Framework,” RFC 6749, IETF, Oct. 2012.

[4] A. Birgisson, J. G. Politz, Ú. Erlingsson, A. Taly, M. Vrable, “Macaroons: Cookies with Contextual Caveats for Decentralized Authorization in the Cloud,” NDSS, 2014.

[5] Biscuit Authorization Token, EPFL, https://biscuitsec.org (specification).

[6] UCAN Specification v0.9, https://ucan.xyz.

[7] C. Ellison, B. Frantz, B. Lampson, R. Rivest, B. Thomas, T. Ylönen, “SPKI Certificate Theory,” RFC 2693, IETF, Sept. 1999.

[8] S. Goldwasser, S. Micali, R. Rivest, “A Digital Signature Scheme Secure Against Adaptive Chosen-Message Attacks,” SIAM Journal on Computing, vol. 17, no. 2, pp. 281–308, 1988.

[9] B. Laurie, A. Langley, E. Kasper, “Certificate Transparency,” RFC 6962, IETF, June 2013.

[10] Sigstore Project Documentation, Linux Foundation.

[11] OASIS, “eXtensible Access Control Markup Language (XACML) Version 3.0,” 2013.

[12] Regulation (EU) No 910/2014 of the European Parliament and of the Council of 23 July 2014 (eIDAS Regulation).

[13] Bürgerliches Gesetzbuch (BGB), §§ 164–181.

---

# Appendix A – Original Contributions and Explicit Non-Goals

This appendix is provided to remove ambiguity regarding authorship, originality, and scope.

### A.1 Original Contributions
The following contributions are original to this manuscript and do not appear as a unified framework in prior literature:
1.  **Proof of Authorization (PoA)** as a protocol-level primitive that binds cryptographic authorization to legal authority rather than mere access control.
2.  **Delegation chains** with explicit legal semantics, including statutory competence, fiduciary scope, and third-party reliance considerations.
3.  **Machine-verifiable liability constraints** embedded directly in authorization artifacts and enforced at decision time.
4.  **Authority revocation transparency**, applying append-only public logs to legal authority rather than identity certificates.
5.  **Jurisdiction-bound authorization**, requiring explicit conflict-of-law resolution inputs for autonomous action.

While individual components draw inspiration from existing systems (e.g., SPKI, capability tokens, transparency logs), their composition and legal binding semantics are novel.

### A.2 What This Work Is Not
To avoid misinterpretation, this work explicitly does not claim to:
*   Invent new cryptographic primitives
*   Automate legal judgment or statutory interpretation
*   Prove agent intent or moral responsibility
*   Replace contract law or regulatory enforcement
*   Eliminate the need for human oversight

The contribution lies in making authority explicit, bounded, and auditable for autonomous systems.

### A.3 Relationship to Prior Art
This manuscript should be read as:
*   A systems and protocol design document, not a survey
*   A legal–technical bridge, not a legal treatise
*   A composition of known cryptographic tools under new legal constraints

Any resemblance to existing authorization mechanisms reflects intentional interoperability, not derivation.
