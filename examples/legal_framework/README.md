# Legal Framework Example

This example demonstrates how to use GAuth to implement a legal and regulatory framework for financial services authentication and authorization with strongly typed structures.

## Key Components

1. **Legal Framework**: The main container for all authorization components with strong typing
2. **Authorities**: Central and delegated authorities with specific powers and hierarchies
3. **Data Sources**: Strongly typed external data providers with secure credentials
4. **Policies and Rules**: Type-safe definition of access control rules with conditions and parameters
5. **Handlers**: Strongly typed components that act on authorization decisions
6. **Validation Rules**: Type-safe rules with parameters for transaction validation

## Running the Example

```bash
go run main.go
```

## Understanding the Code

The example illustrates a financial services legal framework with:

- A central financial regulator with high-level powers
- A delegated bank compliance authority with specific powers
- Strongly typed data sources for customer and transaction data
- Validation rules with typed parameters for transaction amounts
- Policies with condition rules for high-value transactions
- Handlers for audit logging and notifications

## Type-Safe Access to Data

The code demonstrates type-safe access to data through strongly typed structures:

```go
// Old approach using map[string]interface{}:
limitValue, ok := condition.Parameters["limit"].(float64)
if !ok {
    // Handle error...
}

// New approach with strongly typed parameters:
transactionLimitRule := auth.ValidationRule{
    Name:      "transaction-limit",
    Predicate: "amount <= limit",
    Parameters: &auth.RuleParameters{
        AmountLimit: 10000.00,
        Currency:    "USD",
    },
}
```

## Adding Your Own Rules

To extend the framework with your own rules:

1. Define a custom condition type with strongly typed fields
2. Create structures for your parameters instead of using maps
3. Create validation rules or decision rules using your types
4. Add the rule to the framework

## Integration with Compliance Systems

This framework can be extended to integrate with:

- Anti-Money Laundering (AML) systems
- Know Your Customer (KYC) verification
- Regulatory reporting
- Fraud detection systems
- Transaction monitoring

## Architecture

This example demonstrates the Policy Decision Point (PDP) architecture with:

- **Policy Administration Point**: Where policies are defined and managed
- **Policy Decision Point**: Where authorization decisions are made
- **Policy Enforcement Point**: Where decisions are enforced
- **Policy Information Point**: Where authorization context is gathered

## Next Steps

- Implement additional strongly typed policy structures
- Create more sophisticated rule engines with type safety
- Add support for more complex authorization scenarios with proper typing
- Integrate with regulatory reporting systems