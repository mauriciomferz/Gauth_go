# Cascading Failure Demo

This project demonstrates resilience patterns in microservices architectures, showing how failures can cascade through service dependencies and how different patterns can help mitigate these issues.

## Structure

```
.
├── cmd/                    # Example applications
│   └── main.go            # Demo entry point
├── internal/              # Private implementation details
│   └── resilience/        # Resilience pattern implementations
├── pkg/                   # Public API packages
│   └── mesh/             # Service mesh implementation
├── docs/                 # Documentation
└── examples/             # Additional usage examples
```

## Quick Start

```bash
# Run the basic demo
go run cmd/main.go

# Run with increased load factors
go run cmd/main.go -load=0.8

# Run with custom service configuration
go run cmd/main.go -services=order,payment
```

## Packages

### pkg/mesh

The core service mesh implementation provides:
- Service dependency management
- Health monitoring
- Load simulation
- Metric collection

```go
import "cascade/pkg/mesh"

// Create a new service mesh
serviceMesh := mesh.NewServiceMesh()

// Configure service load
serviceMesh.SetServiceLoad(mesh.PaymentService, 0.5)

// Process requests through the mesh
err := serviceMesh.ProcessRequest(ctx, mesh.OrderService)
```

### internal/resilience

Implementation of core resilience patterns:
- Circuit Breaker
- Rate Limiting
- Bulkhead Pattern
- Retry Mechanism

## Design Principles

1. **Clear Entry Points**: Each package has a focused responsibility and clear entry points
2. **Type Safety**: Strong typing throughout with minimal use of interface{}
3. **Modularity**: Each resilience pattern can be used independently
4. **Testability**: Clear interfaces and separation of concerns

## Examples

See the `examples/` directory for additional scenarios:

1. Basic Service Mesh
2. Custom Resilience Patterns
3. Complex Service Dependencies
4. Load Testing Scenarios

## Contributing

We welcome contributions! Please see our [Contributing Guide](docs/CONTRIBUTING.md) for guidelines.

## License

MIT License - See [LICENSE](LICENSE) for details.