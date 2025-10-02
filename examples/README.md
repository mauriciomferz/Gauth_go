# GAuth Examples

**Copyright (c) 2025 Gimel Foundation gGmbH i.G.**  
Licensed under Apache 2.0

**Gimel Foundation gGmbH i.G.**, www.GimelFoundation.com  
Operated by Gimel Technologies GmbH  
MD: Bjørn Baunbæk, Dr. Götz G. Wehberg – Chairman of the Board: Daniel Hartert  
Hardtweg 31, D-53639 Königswinter, Siegburg HRB 18660, www.GimelID.com

This directory contains examples demonstrating RFC-0115 PoA-Definition implementation and other GAuth features. Each example is self-contained with detailed documentation.

## ⭐ **Featured Example - RFC-0115 PoA-Definition**

**[RFC-0115 PoA-Definition Implementation](rfc_0115_poa_definition/README.md)** ✅ **COMPLETE**
- Complete GiFo-RFC-0115 PoA-Definition structure
- Full type safety with Go type system
- JSON serialization and validation
- Gimel Foundation compliance demonstration
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