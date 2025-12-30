// Multi-Signature Power of Attorney Demo
//
// This example demonstrates the complete multi-signature PoA workflow implementing
// AAP002 Section B (Authorization Type) joint/collective signature enforcement
// (GAP_MATRIX sec3.item3).
//
// Scenario: A company requires 3-of-5 board member signatures to authorize a
// high-value financial transaction. This demo shows:
// 1. PoA creation with threshold requirements
// 2. Parallel signature collection from multiple signers
// 3. Threshold completion detection
// 4. PoA activation lifecycle
//
// Usage:
//
//	go run examples/multi_signature_poa/main.go
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/multisig"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth_aap_001"
)

// BoardMember represents a signing authority
type BoardMember struct {
	Name       string
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	KeyID      string
}

// mockKeyProvider implements the VerificationProvider interface
type mockKeyProvider struct {
	keys map[string]ed25519.PublicKey
}

func (m *mockKeyProvider) PublicKey(keyID string) ([]byte, string, error) {
	pub, ok := m.keys[keyID]
	if !ok {
		return nil, "", fmt.Errorf("key not found: %s", keyID)
	}
	return []byte(pub), "ed25519", nil
}

func (m *mockKeyProvider) VerifySignature(digest []byte, signature []byte, publicKey []byte) error {
	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key size: %d", len(publicKey))
	}
	if len(signature) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature size: %d", len(signature))
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey), digest, signature) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║  Multi-Signature Power of Attorney (PoA) Demo                ║")
	fmt.Println("║  AAP002 Section B - Joint/Collective Signature Enforcement  ║")
	fmt.Println("║  GAP_MATRIX sec3.item3 Implementation                        ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Step 1: Generate board member keypairs
	fmt.Println("📋 Step 1: Generating Board Member Keypairs")
	fmt.Println("   Creating 5 board members with Ed25519 keys...")

	boardMembers := []BoardMember{
		generateBoardMember("Alice Chen (CEO)"),
		generateBoardMember("Bob Smith (CFO)"),
		generateBoardMember("Carol Johnson (CTO)"),
		generateBoardMember("David Lee (COO)"),
		generateBoardMember("Eve Wilson (General Counsel)"),
	}

	for i, member := range boardMembers {
		fmt.Printf("   [%d] %s\n", i+1, member.Name)
		fmt.Printf("       Key ID: %s\n", member.KeyID)
	}
	fmt.Println()

	// Step 2: Create key provider with all public keys
	fmt.Println("🔑 Step 2: Setting up Key Provider")
	keyProvider := &mockKeyProvider{
		keys: make(map[string]ed25519.PublicKey),
	}
	for _, member := range boardMembers {
		keyProvider.keys[member.KeyID] = member.PublicKey
	}
	fmt.Printf("   Registered %d public keys\n", len(keyProvider.keys))
	fmt.Println()

	// Step 3: Create signature manager
	fmt.Println("⚙️  Step 3: Initializing Signature Manager")
	manager := multisig.NewSignatureManager(keyProvider)
	fmt.Println("   Manager ready for signature collection")
	fmt.Println()

	// Step 4: Create PoA requiring 3-of-5 signatures
	fmt.Println("📝 Step 4: Creating Power of Attorney")
	fmt.Println("   Authorization: High-value financial transaction")
	fmt.Println("   Threshold: 3 of 5 signatures required")

	poa := &gauth_aap_001.PowerOfAttorney{
		ID:        "poa-board-approval-2025-001",
		Grantor:   "Acme Corporation Board of Directors",
		Grantee:   "Chief Financial Officer",
		Scope:     []string{"authorize_wire_transfer", "approve_acquisition"},
		Threshold: 3,
		Signers: []string{
			boardMembers[0].Name,
			boardMembers[1].Name,
			boardMembers[2].Name,
			boardMembers[3].Name,
			boardMembers[4].Name,
		},
	}

	ctx := context.Background()
	expiresIn := 24 * time.Hour

	if err := manager.InitiateCollection(ctx, poa, expiresIn); err != nil {
		log.Fatalf("Failed to initiate collection: %v", err)
	}
	fmt.Println("   ✓ PoA created and signature collection initiated")
	fmt.Printf("   Expires in: %v\n", expiresIn)
	fmt.Println()

	// Step 5: Get canonical digest
	fmt.Println("🔐 Step 5: Computing Canonical Digest")
	state, _ := manager.GetStatus(ctx, poa.ID)
	digest := state.CanonicalDigest
	fmt.Printf("   Canonical Digest (hex): %s...\n", digest[:64])
	fmt.Println("   All signers will sign this digest")
	fmt.Println()

	// Step 6: Collect signatures (simulating parallel signing)
	fmt.Println("✍️  Step 6: Collecting Board Member Signatures")
	fmt.Println("   Simulating asynchronous signature submission...")
	fmt.Println()

	// Alice signs first
	time.Sleep(100 * time.Millisecond)
	submitSignature(ctx, manager, poa.ID, &boardMembers[0], digest, 1)

	// Bob signs
	time.Sleep(200 * time.Millisecond)
	submitSignature(ctx, manager, poa.ID, &boardMembers[1], digest, 2)

	// Carol signs - this should complete the threshold
	time.Sleep(150 * time.Millisecond)
	submitSignature(ctx, manager, poa.ID, &boardMembers[2], digest, 3)

	// Step 7: Check completion status
	fmt.Println()
	fmt.Println("📊 Step 7: Checking Threshold Status")
	state, _ = manager.GetStatus(ctx, poa.ID)

	fmt.Printf("   Status: %s\n", state.Status)
	fmt.Printf("   Collected: %d/%d signatures\n", len(state.Signatures), state.Threshold)

	if state.Status == multisig.StatusCompleted {
		fmt.Println("   🎉 THRESHOLD MET - PoA ready for activation!")
		if state.CompletedAt != nil {
			fmt.Printf("   Completed at: %s\n", state.CompletedAt.Format(time.RFC3339))
		}
	}
	fmt.Println()

	// Step 8: Display collected signatures
	fmt.Println("📜 Step 8: Signature Collection Details")
	for signerID, record := range state.Signatures {
		fmt.Printf("   ✓ %s\n", signerID)
		fmt.Printf("     Signed at: %s\n", record.SignedAt.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("     Key ID: %s\n", record.KeyID)
		fmt.Printf("     Signature: %s...\n", record.Signature[:32])
	}
	fmt.Println()

	// Step 9: Activate PoA
	fmt.Println("🚀 Step 9: Activating Power of Attorney")
	if err := manager.ActivatePoA(ctx, poa.ID); err != nil {
		log.Fatalf("Failed to activate PoA: %v", err)
	}

	state, _ = manager.GetStatus(ctx, poa.ID)
	fmt.Printf("   Status: %s\n", state.Status)
	if state.ActivatedAt != nil {
		fmt.Printf("   Activated at: %s\n", state.ActivatedAt.Format(time.RFC3339))
	}
	fmt.Println("   ✓ PoA is now ACTIVE and authorized for use")
	fmt.Println()

	// Step 10: Retrieve signatures in AAP001 format
	fmt.Println("📤 Step 10: Exporting Signatures (AAP001 Format)")
	signatures, _ := manager.GetSignatures(ctx, poa.ID)
	fmt.Printf("   Retrieved %d signatures in AAP001 POASignature format\n", len(signatures))
	for signerID, sig := range signatures {
		fmt.Printf("   • %s: Algorithm=%s, KeyID=%s\n",
			signerID, sig.Algorithm, sig.KeyID)
	}
	fmt.Println()

	// Summary
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    DEMO COMPLETE                              ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("✅ Successfully demonstrated:")
	fmt.Println("   • M-of-N threshold signature collection (3-of-5)")
	fmt.Println("   • Canonical digest computation and verification")
	fmt.Println("   • Parallel/asynchronous signature submission")
	fmt.Println("   • Automatic threshold completion detection")
	fmt.Println("   • PoA lifecycle (pending → completed → active)")
	fmt.Println("   • AAP001 signature format compliance")
	fmt.Println()
	fmt.Println("🎯 GAP_MATRIX sec3.item3 (Joint/collective signature enforcement)")
	fmt.Println("   Status: IMPLEMENTED ✓")
	fmt.Println()
}

// generateBoardMember creates a board member with Ed25519 keypair
func generateBoardMember(name string) BoardMember {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate key for %s: %v", name, err)
	}

	keyID := fmt.Sprintf("key-%s", hex.EncodeToString(pub[:8]))

	return BoardMember{
		Name:       name,
		PrivateKey: priv,
		PublicKey:  pub,
		KeyID:      keyID,
	}
}

// submitSignature signs the digest and submits to the manager
func submitSignature(
	ctx context.Context,
	manager *multisig.SignatureManager,
	poaID string,
	member *BoardMember,
	digestHex string,
	sequenceNum int,
) {
	// Convert hex digest to bytes
	digestBytes, err := hex.DecodeString(digestHex)
	if err != nil {
		log.Fatalf("Failed to decode digest: %v", err)
	}

	// Sign the digest
	signature := ed25519.Sign(member.PrivateKey, digestBytes)
	signatureB64 := base64.StdEncoding.EncodeToString(signature)

	// Submit signature
	metadata := map[string]string{
		"ip_address": "192.168.1." + fmt.Sprintf("%d", sequenceNum),
		"user_agent": "BoardMemberApp/1.0",
	}

	err = manager.SubmitSignature(ctx, poaID, member.Name, member.KeyID, signatureB64, metadata)
	if err != nil {
		log.Fatalf("Failed to submit signature from %s: %v", member.Name, err)
	}

	fmt.Printf("   [%d/%d] ✓ %s signed successfully\n", sequenceNum, 3, member.Name)
}
