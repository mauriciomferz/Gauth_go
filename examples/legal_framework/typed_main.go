package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
)

// Strong typing for rule parameters
type TransactionParameters struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Limit    float64 `json:"limit"`
}

type KYCParameters struct {
	RequiredLevel string `json:"required_level"`
	Status        string `json:"status"`
}

func main() {
	// Create a new legal framework
	framework := auth.NewLegalFramework(
		"financial-services-framework",
		"Financial Services Regulatory Framework",
		"1.0.0",
	)

	// Configure the policy decision point to use deny-overrides
	framework.PDP.RuleCombiner = auth.DenyOverridesCombiner

	// Add a central authority
	centralAuthority := auth.Authority{
		ID:    "financial-regulator",
		Type:  "government",
		Level: 100,
		Powers: []auth.Power{
			{
				Type:     "regulation",
				Resource: "financial-transactions",
				Actions:  []string{"approve", "deny", "audit"},
			},
		},
	}
	framework.AddAuthority(centralAuthority)

	// Add a delegated authority
	delegatedAuthority := auth.Authority{
		ID:        "bank-compliance",
		Type:      "organization",
		Level:     50,
		Delegator: "financial-regulator",
		Powers: []auth.Power{
			{
				Type:     "compliance",
				Resource: "customer-transactions",
				Actions:  []string{"review", "report"},
			},
		},
	}
	framework.AddAuthority(delegatedAuthority)

	// Add a data source
	customerDataSource := auth.DataSource{
		ID:       "customer-database",
		Type:     "sql",
		Endpoint: "mysql://customer-db:3306/customers",
		Credentials: auth.Credentials{
			Type:       "basic",
			ID:         "service-account",
			Secret:     "password",
			Expiration: time.Now().Add(24 * time.Hour),
		},
	}
	framework.AddDataSource(customerDataSource)

	// Add a transaction history data source
	transactionDataSource := auth.DataSource{
		ID:       "transaction-history",
		Type:     "api",
		Endpoint: "https://api.bank.example/transactions/v1",
		Credentials: auth.Credentials{
			Type:       "oauth2",
			ID:         "service-client",
			Secret:     "client-secret",
			Expiration: time.Now().Add(1 * time.Hour),
		},
	}
	framework.AddDataSource(transactionDataSource)

	// Add a validation rule with strongly typed parameters
	transactionLimitRule := auth.ValidationRule{
		Name:      "transaction-limit",
		Predicate: "amount <= limit",
		Parameters: TransactionParameters{
			Limit: 10000.00,
		},
	}
	framework.ValidationRules = append(framework.ValidationRules, transactionLimitRule)

	// Add a policy with strongly typed conditions
	highValueTransactionPolicy := auth.Policy{
		ID:          "high-value-transaction-policy",
		Name:        "High Value Transaction Policy",
		Description: "Policy for transactions over $10,000",
		Rules: []auth.DecisionRule{
			{
				Condition: auth.Condition{
					Type: "amount",
					Rule: "amount > 10000",
					Parameters: TransactionParameters{
						Currency: "USD",
					},
				},
				Effect:   "deny",
				Priority: 10,
			},
			{
				Condition: auth.Condition{
					Type: "kyc",
					Rule: "kyc_status == 'verified'",
					Parameters: KYCParameters{
						RequiredLevel: "full",
					},
				},
				Effect:   "permit",
				Priority: 5,
			},
		},
		Version: "1.0.0",
		Created: time.Now(),
		Updated: time.Now(),
	}
	framework.AddPolicy(highValueTransactionPolicy)

	// Add a handler for audit logging
	auditHandler := auth.Handler{
		ID:       "audit-handler",
		Type:     "log",
		Priority: 1,
		Callback: func(ctx context.Context, decision auth.Decision) error {
			log.Printf(
				"AUDIT: %s action %s on %s by %s (Effect: %s)",
				decision.Timestamp.Format(time.RFC3339),
				decision.Action,
				decision.Resource,
				decision.Subject,
				decision.Effect,
			)
			return nil
		},
	}
	framework.AddHandler(auditHandler)

	// Add a handler for notifications
	notificationHandler := auth.Handler{
		ID:       "notification-handler",
		Type:     "notify",
		Priority: 2,
		Callback: func(ctx context.Context, decision auth.Decision) error {
			if decision.Action == "transfer" && decision.Effect == "deny" {
				log.Printf(
					"NOTIFICATION: Denied transfer by %s - sending compliance alert",
					decision.Subject,
				)
			}
			return nil
		},
	}
	framework.AddHandler(notificationHandler)

	// Example usage: Make an authorization decision
	ctx := context.Background()
	decision, err := framework.Authorize(ctx, "customer-123", "account-456", "transfer")
	if err != nil {
		log.Fatalf("Authorization failed: %v", err)
	}

	fmt.Printf("Authorization decision: %s\n", decision.Effect)
	fmt.Printf("Subject: %s\n", decision.Subject)
	fmt.Printf("Resource: %s\n", decision.Resource)
	fmt.Printf("Action: %s\n", decision.Action)
	fmt.Printf("Timestamp: %s\n", decision.Timestamp.Format(time.RFC3339))
}
