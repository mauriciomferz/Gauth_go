# Email Template: Design Partner Recruitment

## Template 1: Executive Outreach

**Subject:** Solving the "AI Liability Gap" at [Company Name]

**Body:**

Hi [Name],

I've been following [Company Name]'s pioneering work in [Sector], specifically your deployment of AI agents for [Specific Use Case].

The industry is hitting a fundamental wall: **OAuth gives agents API access, but not legal authority.**

We've documented cases where this gap led to:
- A $2.3M procurement order to a sanctioned supplier (agent had valid OAuth token)
- Unauthorized contract modifications by agents exceeding their mandate
- Compliance failures because the authorization system couldn't encode business rules

I created **AgentAuth** to solve this. Instead of access tokens, it issues **Proof of Authorization (PoA)** tokens that embed:
- **Liability caps**: "Max spend €50K per transaction"
- **Authorization chains**: Traceable delegation from board resolution to agent
- **Dual control**: "Require human approval above threshold X"
- **Jurisdiction**: "Applicable law: German Civil Code"

The entire authorization chain is cryptographically verifiable—even offline.

We're recruiting **3 Design Partners** before public release. Given your advanced agentic deployments, I thought this might directly address the compliance/liability bottleneck you're likely encountering.

I've attached a technical overview and would welcome a 15-minute call to explore fit.

Best regards,

Mauricio A. Fernandez Fernandez  
Creator, AgentAuth  
Author, "The Agent's Signature: Identity & Law in the Age of AI"

---

## Template 2: Technical Lead Outreach

**Subject:** RFC: Cryptographic mandate tokens for AI agents

**Body:**

Hi [Name],

Quick question: How are you handling authorization limits for your AI agents?

Most teams I talk to are using OAuth + application-layer checks. The problem is that the authorization layer is "dumb"—it can validate scopes but can't enforce:
- Spending limits
- Geographic restrictions
- Time-of-day windows
- Multi-signature requirements

I built **AgentAuth** (open source, MIT) to solve this at the protocol level.

Key architecture:
- PoA tokens carry full authorization metadata + delegation chain
- Cryptographic verification works offline (Degraded Mode)
- Supports EdDSA, ECDSA, BLS aggregate signatures
- Merkle tree-based revocation with inclusion proofs

Implementation is in Go, ~80% test coverage, deployed in production.

Looking for feedback from teams with real agentic workloads. Would you be interested in a technical deep-dive?

Mauricio

GitHub: [Link to Repo]
Technical Book: "The Agent's Signature" (1,360 lines with architecture diagrams)

---

## Template 3: Investor/Strategic Partner

**Subject:** The Trust Layer for the Agentic Economy

**Body:**

[Name],

McKinsey projects $25 trillion in agent-initiated transactions by 2030. Yet the authorization infrastructure is stuck in 2005.

OAuth solved "delegated access" for human users clicking buttons. It never addressed:
- Who is liable when an agent signs a contract?
- How do you verify authority when the server is offline?
- Can an agent's authorization be independently audited in court?

**AgentAuth** answers these questions with a production-ready protocol:

✅ AAP-001: Cryptographic identity for AI agents
✅ AAP-002: Proof-of-Authorization with embedded fiduciary logic
✅ Degraded Mode: Offline verification for industrial/military edge
✅ Legal Framework: Supports German, US, and EU jurisdictions

We're seeking design partners and strategic investors who understand that the Agentic Economy needs more than API keys—it needs legal infrastructure.

I'd welcome 20 minutes to share our roadmap.

Mauricio A. Fernandez Fernandez  
Creator, AgentAuth  
Author, "The Agent's Signature"

---

## Attachments to Include

1. **Technical Overview** (1-pager PDF)
2. **Blog Post**: "Why OAuth Isn't Enough for AI Agents"
3. **Book Chapter 1**: "Beyond OAuth – Why Access Tokens Aren't Enough"
4. **Architecture Diagram**: System flow from PoA issuance to verification

---

*Last Updated: December 31, 2025*
