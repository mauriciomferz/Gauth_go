---
title: "Examples Index"
category: example-index
status: active
lastUpdated: 2025-11-12
owners: architecture-team
refreshCadence: on-change
---
# AgentAuth Examples

> Last Updated: 2025-10-17
> Status: Active

> **⚠️ BETA EXAMPLES NOTICE**
>
> These examples are for **beta demonstration purposes only**.
> **NOT production ready. This is for beta demonstration purposes only. Do NOT use for real security, production, or commercial deployment.**

**Copyright (c) 2025 AgentAuth Community gGmbH i.G.**
Licensed under Apache 2.0

**AgentAuth Community gGmbH i.G.**, www.AgentAuthFoundation.com
Operated by AgentAuth Technologies GmbH
MD: Bjørn Baunbæk, Dr. Götz G. Wehberg – Chairman of the Board: Daniel Hartert
Hardtweg 31, D-53639 Königswinter, Siegburg HRB 18660, www.AgentAuthID.com

This directory contains beta examples demonstrating RFC-0115 PoA-Definition concepts and other AgentAuth learning materials. Each example is designed for learning and understanding AgentAuth principles.

## ⭐ **Featured Example - RFC-0115 PoA-Definition**

**[RFC-0115 PoA-Definition Implementation](rfc_0115_poa_definition/README.md)** ✅ **COMPLETE**
- Complete AAP-RFC-0115 PoA-Definition structure
- Full type safety with Go type system
- JSON serialization and validation
- AgentAuth Community compliance demonstration
- **Status**: Fully functional and tested

```bash
cd examples/rfc_0115_poa_definition
go run main.go
```

## Quick Start Examples

1. [Basic Auth](basic/README.md)
   - Simple authentication setup
   - Token validation
   - Basic rate limiting

2. [Rate Limiting](rate/README.md)
   - Different rate limiting algorithms
   - Distributed rate limiting
   - Custom rate limit policies

3. [Token Management](token_management/README.md)
   - Token generation and validation
   - Token storage options
   - Key rotation and security

## Advanced Examples

1. [Advanced Auth](advanced/README.md)
   - Multiple auth providers
   - Custom auth flows
   - Complex permissions

2. [Distributed Setup](distributed/README.md)
   - Redis-based token store
   - Distributed rate limiting
   - Service mesh integration

3. [Gateway](gateway/README.md)
   - API gateway setup
   - Auth middleware
   - Request routing

4. [Resilience](resilient/README.md)
   - Circuit breakers
   - Retry policies
   - Fallback strategies

## Use Case Examples

1. [Microservices](microservices/README.md)
   - Service-to-service auth
   - Distributed tracing
   - Load balancing

2. [Custom Server](custom_server/README.md)
   - Custom auth server
   - Policy management
   - Audit logging

3. [Patterns](patterns/README.md)
   - Common auth patterns
   - Best practices
   - Security considerations

## Running Examples

Each example can be run independently:

```bash
# Run basic example
cd basic
go run main.go

# Run rate limiting example
cd ../rate
go run main.go

# Run token management example
cd ../token_management
go run main.go
```

## Example Structure

Each example follows this structure:
```
example/
  ├── README.md           # Documentation and setup
  ├── main.go            # Main entry point
  ├── handlers/          # HTTP handlers
  │   └── auth.go        # Auth-related handlers
  ├── middleware/        # Middleware components
  │   └── auth.go        # Auth middleware
  └── config/           # Configuration
      └── auth.go       # Auth configuration
```

## Contributing Examples

When adding new examples:
1. Create a new directory
2. Add comprehensive README.md
3. Include clear documentation
4. Add tests if applicable
5. Update this index

## Cryptographic Transparency Artifacts

The following JSON artifacts demonstrate transparency and verifiability concepts.

| Artifact | Purpose | Notes |
|----------|---------|-------|
| `revocation_inclusion_proof.json` | Example revocation Merkle inclusion proof and signed tree head | Sibling hashes truncated (placeholders) – structural only. |
| `rotation_summary_multisig.json` | Prospective multi-signature rotation summary (threshold 2) | Signatures are placeholders; endpoint currently emits single-sign summary. |

These artifacts complement the runtime endpoints: `/api/v1/token/revocation/proof` and `/api/v1/beta/rotations/summary`.
They are intended for documentation, test scaffolding, and forward-looking design discussions around threshold signing.

---
Need context? See: README.md | docs/ARCHITECTURE.md | docs/GETTING_STARTED.md
