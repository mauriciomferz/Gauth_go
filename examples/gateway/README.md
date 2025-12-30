# API Gateway Resilience Example

> Last Updated: 2025-10-17
> Status: Active

This example demonstrates an API Gateway with resilience patterns using the AgentAuth framework. It covers global and service-level rate limiting, circuit breaker, retry, bulkhead isolation, and simulated backend failures.

## Key Concepts
- **APIGateway**: Manages routes, services, and resilience patterns.
- **BackendService**: Simulates downstream services with latency and error rates.
- **RouteConfig**: Configures resilience for each route.
- **Resilience Patterns**: Rate limiting, circuit breaker, retry, bulkhead.

## How to Run
```bash
cd examples/gateway
go run main.go
```

## Code Review Summary
 - Beta comments clarify major blocks and usage patterns.

- Extend to add more routes, services, or custom patterns as needed.

---

For more, see the [Architecture Guide](../../docs/ARCHITECTURE.md).
