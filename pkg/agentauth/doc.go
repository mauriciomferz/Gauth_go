// Package agentauth provides the core AgentAuth 1.0 authorization framework implementation.
//
// AgentAuth (Generic Authorization) is a comprehensive authorization system that supports
// delegated authorization, proof-of-authorization tokens, and multi-signature validation.
// This package implements the core AAP001 and AAP002 specifications.
//
// # Key Features
//
//   - Delegated authorization with proof-of-authorization (POA) tokens
//   - Multi-signature support with threshold validation
//   - Replay attack protection
//   - Key rotation with audit trails
//   - Revocation transparency using Merkle trees
//
// # Basic Usage
//
// Creating and validating a simple authorization token:
//
//	// Initialize the service
//	svc, err := agentauth.NewService(&agentauth.Config{
//	    KeyProvider:   keyProvider,
//	    ReplayStore:   replayStore,
//	    RevocationMgr: revocationMgr,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Create an authorization context
//	ctx := &agentauth.AuthContext{
//	    Subject:  "user:alice",
//	    Resource: "document:123",
//	    Action:   "read",
//	    NotAfter: time.Now().Add(1 * time.Hour),
//	}
//
//	// Generate token
//	token, err := svc.Authorize(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Validate token
//	claims, err := svc.Validate(token)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Delegation Chains
//
// AgentAuth supports delegation chains where authority can be delegated from one
// principal to another:
//
//	// Alice delegates to Bob
//	delegationToken, err := svc.Delegate(&agentauth.DelegationContext{
//	    Delegator: "user:alice",
//	    Delegate:  "user:bob",
//	    Resource:  "document:123",
//	    Actions:   []string{"read"},
//	    NotAfter:  time.Now().Add(24 * time.Hour),
//	})
//
//	// Bob can now act on behalf of Alice
//	authToken, err := svc.AuthorizeWithDelegation(delegationToken, &agentauth.AuthContext{
//	    Subject:  "user:bob",
//	    Resource: "document:123",
//	    Action:   "read",
//	})
//
// # Multi-Signature Validation
//
// For high-security operations, AgentAuth supports multi-signature validation:
//
//	// Create context requiring 2 of 3 signatures
//	ctx := &agentauth.MultiSigContext{
//	    Signers:   []string{"admin:alice", "admin:bob", "admin:charlie"},
//	    Threshold: 2,
//	    Resource:  "critical-system",
//	    Action:    "shutdown",
//	}
//
//	// Each signer signs independently
//	sig1, _ := svc.Sign(ctx, aliceKey)
//	sig2, _ := svc.Sign(ctx, bobKey)
//
//	// Validate with threshold
//	valid, err := svc.ValidateMultiSig(ctx, []Signature{sig1, sig2})
//
// # Revocation
//
// AgentAuth provides cryptographically verifiable revocation:
//
//	// Revoke a token
//	err := svc.Revoke(tokenID, "Security incident")
//
//	// Check revocation status
//	revoked, proof, err := svc.CheckRevocation(tokenID)
//	if revoked {
//	    // Token is revoked, proof contains Merkle tree inclusion proof
//	}
//
// # Thread Safety
//
// All Service methods are safe for concurrent use by multiple goroutines.
// The underlying stores (key, replay, revocation) must also be thread-safe.
//
// # Performance Considerations
//
//   - Token validation is cryptographically intensive; use caching where appropriate
//   - Replay store operations may require database access; consider using memory stores for high throughput
//   - Revocation checks involve Merkle tree operations; batch checks when possible
//
// # Security Considerations
//
//   - Always use cryptographically secure random number generators for token IDs
//   - Implement proper key rotation policies (recommended: 90 days)
//   - Store private keys securely (HSM, KMS, or encrypted storage)
//   - Validate all inputs to prevent injection attacks
//   - Use TLS for all network communication
//   - Implement rate limiting to prevent abuse
//
// For more details, see the AgentAuth RFC specifications:
//   - AAP001: Core Authorization Protocol
//   - AAP002: Proof-of-Authorization (POA) Tokens
package agentauth
