// PASETO Advanced Example with Structured Footer
// This demonstrates PASETO token creation and validation with advanced claims and structured footer
// Usage: go run advanced_demo.go or call runAdvancedDemo() from main
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/gauth"
)

func runAdvancedDemo() {
	fmt.Println("=== PASETO Advanced Claims and Structured Footer Demo ===")

	// Create advanced claims with comprehensive metadata
	advancedClaims := createAdvancedClaimsExample()
	fmt.Printf("Created Advanced Claims:\n%s\n\n", formatJSON(advancedClaims.ToMap()))

	// Create structured PASETO footer
	footer := createStructuredFooter()
	fmt.Printf("Created Structured Footer:\n%s\n\n", formatJSON(footer))

	// Simulate PASETO token creation (v4.public format)
	token := createMockPASETOToken(advancedClaims, footer)
	fmt.Printf("Mock PASETO Token:\n%s\n\n", token)

	// Demonstrate advanced claims validation
	demonstrateValidation(advancedClaims)

	// Demonstrate time window validation
	demonstrateTimeWindowValidation()

	// Demonstrate claims metadata confidence scoring
	demonstrateConfidenceScoring()
}

func createAdvancedClaimsExample() *gauth.AdvancedClaims {
	return &gauth.AdvancedClaims{
		Subject:   "user@example.com",
		Issuer:    "https://auth.gauth.example.com",
		Audience:  []string{"api.gauth.example.com", "mobile.gauth.example.com"},
		ExpiresAt: time.Now().Add(2 * time.Hour).Unix(),
		IssuedAt:  time.Now().Unix(),
		NotBefore: time.Now().Unix(),
		JWTID:     "gauth-advanced-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Scope:     []string{"gauth:read", "gauth:write", "gauth:admin"},
		TokenType: "PASETO",
		ClientID:  "gauth-mobile-app-v2",
		ClaimsMetadata: &gauth.ClaimsMetadata{
			Version:      "2.0",
			Capabilities: []string{"delegation", "revocation", "audit", "multi-tenant"},
			Source:       "internal-identity-provider",
			Confidence:   0.98,
			Restrictions: &gauth.ClaimsRestrictions{
				TimeWindow: &gauth.TimeWindow{
					StartHour: 6,                    // 6 AM
					EndHour:   20,                   // 8 PM
					Weekdays:  []int{1, 2, 3, 4, 5}, // Monday-Friday
				},
				UsageLimit:     500,
				GeofenceRegion: "US-WEST",
				IPWhitelist:    []string{"192.168.1.0/24", "10.0.0.0/8"},
			},
		},
		Custom: map[string]interface{}{
			"tenant_id":          "gauth-enterprise-001",
			"session_id":         "sess_" + fmt.Sprintf("%d", time.Now().UnixNano()),
			"device_fingerprint": "mobile-ios-16.1-iphone14",
			"risk_score":         0.15,
			"delegation_chain":   []string{"admin@example.com", "manager@example.com"},
			"audit_context": map[string]interface{}{
				"action":    "token_issued",
				"timestamp": time.Now().Format(time.RFC3339),
				"source_ip": "192.168.1.100",
			},
		},
	}
}

func createStructuredFooter() *gauth.PASETOFooter {
	footer, err := gauth.CreatePASETOFooter(
		"gauth-ed25519-key-v2-2025",
		"Ed25519",
		"https://auth.gauth.example.com",
		map[string]interface{}{
			"version":        "4.0",
			"purpose":        "public",
			"key_rotation":   24, // hours
			"compliance":     []string{"SOC2", "GDPR", "HIPAA"},
			"jurisdiction":   "US-CA",
			"audit_trail_id": "audit_" + fmt.Sprintf("%d", time.Now().UnixNano()),
			"chain_of_trust": map[string]interface{}{
				"root_ca":      "AgentAuth-Root-CA-2025",
				"intermediate": "AgentAuth-Intermediate-CA-West",
				"leaf":         "auth.gauth.example.com",
			},
		},
	)
	if err != nil {
		log.Fatalf("Failed to create PASETO footer: %v", err)
	}
	return footer
}

func createMockPASETOToken(claims *gauth.AdvancedClaims, footer *gauth.PASETOFooter) string {
	// This creates a mock PASETO v4.public token format
	// In a real implementation, this would use a proper PASETO library

	// Encode claims as payload
	claimsJSON, _ := json.Marshal(claims.ToMap())
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// Encode footer
	footerJSON, _ := footer.ToJSON()
	footerEncoded := base64.RawURLEncoding.EncodeToString([]byte(footerJSON))

	// Mock PASETO format: v4.public.payload.footer
	return fmt.Sprintf("v4.public.%s.%s", payload, footerEncoded)
}

func demonstrateValidation(claims *gauth.AdvancedClaims) {
	fmt.Println("=== Advanced Claims Semantic Validation ===")

	// Test valid claims
	err := claims.ValidateSemantics()
	if err != nil {
		fmt.Printf("❌ Validation failed: %v\n", err)
	} else {
		fmt.Printf("✅ Claims validation passed\n")
	}

	// Test claims metadata validation
	if claims.ClaimsMetadata != nil {
		err = claims.ClaimsMetadata.Validate()
		if err != nil {
			fmt.Printf("❌ Metadata validation failed: %v\n", err)
		} else {
			fmt.Printf("✅ Metadata validation passed\n")
		}
	}

	// Test restrictions validation
	if claims.ClaimsMetadata != nil && claims.ClaimsMetadata.Restrictions != nil {
		restrictions := claims.ClaimsMetadata.Restrictions
		err = restrictions.Validate()
		if err != nil {
			fmt.Printf("❌ Restrictions validation failed: %v\n", err)
		} else {
			fmt.Printf("✅ Restrictions validation passed\n")
		}

		// Test time window check
		inWindow := restrictions.IsInTimeWindow()
		fmt.Printf("📅 Current time in allowed window: %v\n", inWindow)
	}

	fmt.Println()
}

func demonstrateTimeWindowValidation() {
	fmt.Println("=== Time Window Validation Examples ===")

	// Business hours restriction
	businessHours := &gauth.ClaimsRestrictions{
		TimeWindow: &gauth.TimeWindow{
			StartHour: 9,                    // 9 AM
			EndHour:   17,                   // 5 PM
			Weekdays:  []int{1, 2, 3, 4, 5}, // Monday-Friday
		},
	}

	fmt.Printf("Business hours allowed: %v\n", businessHours.IsInTimeWindow())

	// 24/7 access
	alwaysAllowed := &gauth.ClaimsRestrictions{
		TimeWindow: &gauth.TimeWindow{
			StartHour: 0,
			EndHour:   23,
			Weekdays:  []int{0, 1, 2, 3, 4, 5, 6}, // All days
		},
	}

	fmt.Printf("24/7 access allowed: %v\n", alwaysAllowed.IsInTimeWindow())

	// Weekend only
	weekendOnly := &gauth.ClaimsRestrictions{
		TimeWindow: &gauth.TimeWindow{
			StartHour: 0,
			EndHour:   23,
			Weekdays:  []int{0, 6}, // Saturday and Sunday
		},
	}

	fmt.Printf("Weekend only allowed: %v\n", weekendOnly.IsInTimeWindow())
	fmt.Println()
}

func demonstrateConfidenceScoring() {
	fmt.Println("=== Claims Confidence Scoring Examples ===")

	// High confidence claims with metadata
	highConfidenceClaims := &gauth.AdvancedClaims{
		Subject:   "admin@example.com",
		Audience:  []string{"api.example.com"},
		TokenType: "JWT",
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(), // Long-lived
		IssuedAt:  time.Now().Unix(),
		ClaimsMetadata: &gauth.ClaimsMetadata{
			Version:    "2.0",
			Confidence: 0.95,
			Restrictions: &gauth.ClaimsRestrictions{
				UsageLimit: 1000,
			},
		},
	}

	// Basic claims with minimal metadata
	basicClaims := &gauth.AdvancedClaims{
		Subject:   "user@example.com",
		Audience:  []string{"api.example.com"},
		ExpiresAt: time.Now().Add(time.Hour).Unix(), // Short-lived
		IssuedAt:  time.Now().Unix(),
	}

	fmt.Printf("High confidence claims score: %.2f\n", calculateMockConfidence(highConfidenceClaims))
	fmt.Printf("Basic claims score: %.2f\n", calculateMockConfidence(basicClaims))
	fmt.Println()
}

// Mock confidence calculation for demonstration
func calculateMockConfidence(claims *gauth.AdvancedClaims) float64 {
	score := 0.5 // Base score

	if claims.ClaimsMetadata != nil {
		score += 0.2
		if claims.ClaimsMetadata.Confidence > 0 {
			score = (score + claims.ClaimsMetadata.Confidence) / 2
		}
	}

	if claims.TokenType != "" {
		score += 0.1
	}

	if claims.ClaimsMetadata != nil && claims.ClaimsMetadata.Restrictions != nil {
		score += 0.1
	}

	if claims.ExpiresAt > 0 && claims.IssuedAt > 0 {
		duration := claims.ExpiresAt - claims.IssuedAt
		if duration > 3600 { // More than 1 hour
			score += 0.1
		}
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

func formatJSON(data interface{}) string {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting JSON: %v", err)
	}
	return string(jsonBytes)
}
