// Example: Simple Authorization Server with JWE
// Demonstrates creating and encrypting Extended Tokens
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/agentauth"
	"github.com/mauriciomferz/AgentAuth/pkg/poa"
	"github.com/mauriciomferz/AgentAuth/pkg/poa/taxonomy"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== AgentAuth Authorization Server with JWE ===")

	// Step 1: Generate or load RSA keys
	fmt.Println("Step 1: Setting up RSA keys...")
	privateKey, publicKey, err := setupKeys()
	if err != nil {
		log.Fatalf("Failed to setup keys: %v", err)
	}
	fmt.Println("✅ RSA keys ready (2048-bit)")

	// Step 2: Create JWE service
	fmt.Println("Step 2: Creating JWE service...")
	jweConfig := &agentauth.JWEConfig{
		Enabled:    true,
		Algorithm:  "RSA-OAEP-256",
		Encryption: "A256GCM",
		KeyID:      "agentauth-example-2025-11",
	}

	tmpDir, _ := os.MkdirTemp("", "jwe-example-*")
	defer os.RemoveAll(tmpDir)

	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")

	agentauth.SaveRSAPrivateKey(privateKey, privateKeyPath)
	agentauth.SaveRSAPublicKey(publicKey, publicKeyPath)

	jweConfig.PublicKeyPath = publicKeyPath
	jweConfig.PrivateKeyPath = privateKeyPath

	jweService, err := agentauth.NewJWEService(jweConfig)
	if err != nil {
		log.Fatalf("Failed to create JWE service: %v", err)
	}
	fmt.Println("✅ JWE service created")

	// Step 3: Create Extended Token Service
	fmt.Println("Step 3: Creating Extended Token Service...")
	tokenService := agentauth.NewExtendedTokenService(
		&agentauth.AuthorizationChainValidator{},
		&agentauth.ComplianceValidator{},
		&mockPIP{},
		"https://authserver.example.com",
		"https://authserver.example.com",
		1*time.Hour,
	)
	tokenService = tokenService.WithJWEEncryption(jweService)
	fmt.Println("✅ Token service with JWE enabled")

	// Step 4: Create Extended Token
	fmt.Println("Step 4: Creating Extended Token...")
	request := createSampleTokenRequest()
	token, err := tokenService.CreateExtendedToken(ctx, request)
	if err != nil {
		log.Fatalf("Failed to create token: %v", err)
	}
	fmt.Println("✅ Extended Token created:")
	fmt.Printf("   Access Token ID: %s\n", token.AccessToken[:20]+"...")
	fmt.Printf("   Token Type: %s\n", token.TokenType)
	fmt.Printf("   Expires In: %d seconds\n", token.ExpiresIn)
	fmt.Printf("   Scope: %v\n\n", token.Scope)

	// Step 5: Encode and Encrypt Token
	fmt.Println("Step 5: Encoding and encrypting token...")
	encodedToken, err := tokenService.EncodeExtendedToken(ctx, token)
	if err != nil {
		log.Fatalf("Failed to encode token: %v", err)
	}

	if !agentauth.IsJWE(encodedToken) {
		log.Fatal("ERROR: Token is not JWE format!")
	}

	fmt.Println("✅ Token encrypted to JWE:")
	fmt.Printf("   Format: JWE (5 parts)\n")
	fmt.Printf("   Size: %d bytes\n", len(encodedToken))
	fmt.Printf("   Key ID: %s\n", jweConfig.KeyID)

	// Step 6: Output token
	fmt.Println("=== JWE TOKEN (copy this to resource server) ===")
	fmt.Println(encodedToken)
	fmt.Println("===============================================")

	fmt.Println("✅ Authorization complete!")
	fmt.Println("Next step: Send this token to the resource server")
	fmt.Println("Command: go run resource_server.go '<token>'")
}

func setupKeys() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	// Check for existing keys in environment
	privKeyPath := os.Getenv("AGENTAUTH_JWE_PRIVATE_KEY")
	pubKeyPath := os.Getenv("AGENTAUTH_JWE_PUBLIC_KEY")

	if privKeyPath != "" && pubKeyPath != "" {
		// Load existing keys
		fmt.Println("  Loading keys from environment variables...")
		privateKey, err := agentauth.LoadRSAPrivateKey(privKeyPath)
		if err != nil {
			return nil, nil, err
		}
		return privateKey, &privateKey.PublicKey, nil
	}

	// Generate new keys
	fmt.Println("  Generating new RSA key pair (2048-bit)...")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	return privateKey, &privateKey.PublicKey, nil
}

func createSampleTokenRequest() *agentauth.ExtendedTokenRequest {
	now := time.Now()

	// Sample Power of Attorney
	// Adapt to current poa.PoADefinition struct: Parties + Authorization + Requirements.
	samplePoA := &poa.PoADefinition{
		Parties: poa.Parties{
			Principal:        poa.Principal{Type: "Organization", Identity: "company-abc-123", Organization: &poa.Organization{Name: "Company ABC Ltd.", RegisterEntry: "HRB 12345", ManagingDirector: "Jane Smith", RegisteredAuthority: true}},
			Representative:   &poa.Representative{Identity: "john-doe-456", LegalRelationship: poa.RelationshipOwner},
			AuthorizedClient: poa.AuthorizedClient{Identity: "client-789", Type: "AI"},
		},
		Authorization: poa.AuthorizationScope{
			AuthorizedActions: poa.AuthorizedActions{
				Transactions:       []poa.TransactionType{taxonomy.TransactionLoan},
				NonPhysicalActions: []poa.ActionTypeNonPhysical{taxonomy.ActionNonPhysicalResearching},
			},
			ApplicableRegions: []poa.GeographicScope{{Type: poa.GeoTypeNational, Identifier: "DE"}},
		},
		Requirements: poa.Requirements{
			ValidityPeriod: poa.ValidityPeriod{StartTime: now.Add(-30 * 24 * time.Hour), EndTime: now.Add(365 * 24 * time.Hour)},
		},
	}

	// Sample Authorization Chain
	chain := &agentauth.AuthorizationChain{
		OwnersAuthorizer: &agentauth.AuthorizationLink{
			EntityID:   "company-abc-123",
			EntityType: "organization",
			EntityName: "Company ABC Ltd.",
			Role:       "authorizer",
		},
		ClientOwner: &agentauth.AuthorizationLink{
			EntityID:   "john-doe-456",
			EntityType: "natural_person",
			EntityName: "John Doe",
			Role:       "owner",
		},
	}

	return &agentauth.ExtendedTokenRequest{
		GrantID:            "grant-example-001",
		Scope:              []string{"read:accounts", "write:payments"},
		PowerOfAttorney:    samplePoA,
		AuthorizationChain: chain,
		LegalFramework:     &agentauth.LegalFrameworkInfo{Jurisdiction: "DE"},
	}
}

// Mock PIP for example
type mockPIP struct{}

func (m *mockPIP) GetClientInfo(ctx context.Context, clientID string) (*agentauth.ClientInfo, error) {
	return &agentauth.ClientInfo{
		ClientID:   clientID,
		ClientName: "Example Client",
		Active:     true,
	}, nil
}

func (m *mockPIP) GetAuthorizationServerInfo(ctx context.Context, serverID string) (*agentauth.AuthorizationServerInfo, error) {
	return &agentauth.AuthorizationServerInfo{
		ServerID:   serverID,
		ServerName: "Example Auth Server",
	}, nil
}
