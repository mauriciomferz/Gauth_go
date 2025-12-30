package main

import (
	"fmt"
	"time"

	a "github.com/mauriciomferz/AgentAuth/pkg/audit"
	c "github.com/mauriciomferz/AgentAuth/pkg/compliance"
	p "github.com/mauriciomferz/AgentAuth/pkg/policy"
	v "github.com/mauriciomferz/AgentAuth/pkg/validation"
)

// integration_demo shows how the new beta scaffolding components could be wired together.
func main() {
	policyEng := p.NewStubEngine()
	validator := v.NewBasicValidator()
	ledger := a.NewLedger()
	registry := c.NewRegistry()

	// Register a sample data flow
	registry.Register(c.Flow{ID: "flow1", Source: "auth", Destination: "audit", DataTypes: []c.DataClass{c.DataClassOperational}, Purpose: "demo-event", Retention: "7d"})

	// Validate sample claims
	claimsErr := validator.ValidateClaims(v.Claims{Subject: "user123", Issuer: "issuer", Audience: []string{"svc"}, ExpiresAt: time.Now().Add(10 * time.Minute)})
	if claimsErr != nil {
		fmt.Println("claims validation failed:", claimsErr)
		return
	}

	// Policy evaluation (authorization)
	dec, _ := policyEng.EvaluateAuthorization(p.AuthzInput{Subject: "user123", Action: "read", Resource: "resource:alpha"})
	fmt.Println("policy decision allow=", dec.Allow, "reason=", dec.ReasonCode)

	// Append audit entry
	if _, err := ledger.Append("user123", "read", "resource:alpha", "demo access"); err != nil {
		fmt.Println("audit append failed:", err)
	}

	// Verify ledger integrity
	if err := ledger.Verify(); err != nil {
		fmt.Println("ledger verify failed:", err)
	} else {
		fmt.Println("ledger verify ok")
	}

	// List flows
	for _, f := range registry.List() {
		fmt.Printf("flow: %s %s -> %s\n", f.ID, f.Source, f.Destination)
	}
}
