package tokenmanagement

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// ClusterNode simulates a distributed node
type ClusterNode struct {
	id          string
	store       token.Store
	blacklist   *token.Blacklist
	validator   *token.ValidationChain
	jwtManager  *token.JWTManager
	tokenEvents chan TokenEvent
}

type TokenEvent struct {
	Type      string // "created", "revoked", "rotated"
	TokenID   string
	Timestamp time.Time
}

func NewClusterNode(id string, sharedChan chan TokenEvent) *ClusterNode {
	store := token.NewMemoryStore(24 * time.Hour)
	blacklist := token.NewBlacklist()

	node := &ClusterNode{
		id:        id,
		store:     store,
		blacklist: blacklist,
		jwtManager: token.NewJWTManager(token.JWTConfig{
			SigningMethod: token.HS256,
			SigningKey:    []byte("shared-cluster-key"),
			KeyID:         "cluster-key-1",
			MaxAge:        time.Hour,
		}),
		tokenEvents: sharedChan,
	}

	node.validator = token.NewValidationChain(blacklist)
	go node.handleEvents()
	return node
}

func (n *ClusterNode) handleEvents() {
	for event := range n.tokenEvents {
		switch event.Type {
		case "revoked":
			// Simulate receiving revocation from another node
			n.handleRevocation(event.TokenID)
		}
	}
}

func (n *ClusterNode) handleRevocation(tokenID string) {
	// Simulate token revocation sync across cluster
	fmt.Printf("Node %s: Received revocation for token %s\n", n.id, tokenID)
}

func (n *ClusterNode) CreateToken(ctx context.Context, subject string) (*token.Token, string, error) {
	// Create new token
	t := &token.Token{
		ID:        token.NewID(),
		Type:      token.Access,
		Subject:   subject,
		Issuer:    fmt.Sprintf("node-%s", n.id),
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Scopes:    []string{"read", "write"},
		Metadata: map[string]string{
			"node": n.id,
		},
	}

	// Store token
	if err := n.store.Save(ctx, t); err != nil {
		return nil, "", err
	}

	// Sign token
	signed, err := n.jwtManager.SignToken(ctx, t)
	if err != nil {
		return nil, "", err
	}

	// Broadcast creation event
	n.tokenEvents <- TokenEvent{
		Type:      "created",
		TokenID:   t.ID,
		Timestamp: time.Now(),
	}

	return t, signed, nil
}

func (n *ClusterNode) RevokeToken(ctx context.Context, tokenID string) error {
	// Get token
	t, err := n.store.Get(ctx, tokenID)
	if err != nil {
		return err
	}

	// Add to blacklist
	if err := n.blacklist.Add(ctx, t, "cluster revocation"); err != nil {
		return err
	}

	// Broadcast revocation
	n.tokenEvents <- TokenEvent{
		Type:      "revoked",
		TokenID:   tokenID,
		Timestamp: time.Now(),
	}

	return nil
}

func main() {
	ctx := context.Background()

	// Create shared event channel
	events := make(chan TokenEvent, 100)
	defer close(events)

	// Create cluster nodes
	node1 := NewClusterNode("1", events)
	node2 := NewClusterNode("2", events)
	node3 := NewClusterNode("3", events)

	// Simulate distributed token management
	var wg sync.WaitGroup
	wg.Add(3)

	// Node 1: Create and share token
	go func() {
		defer wg.Done()
		token, signed, err := node1.CreateToken(ctx, "user123")
		if err != nil {
			log.Printf("Node 1 failed to create token: %v", err)
			return
		}
		fmt.Printf("Node 1 created token: %s\n", token.ID)

		// Simulate other nodes verifying the token
		if verified, err := node2.jwtManager.VerifyToken(ctx, signed); err != nil {
			log.Printf("Node 2 failed to verify token: %v", err)
		} else {
			fmt.Printf("Node 2 verified token for subject: %s\n", verified.Subject)
		}

		if verified, err := node3.jwtManager.VerifyToken(ctx, signed); err != nil {
			log.Printf("Node 3 failed to verify token: %v", err)
		} else {
			fmt.Printf("Node 3 verified token for subject: %s\n", verified.Subject)
		}
	}()

	// Node 2: Revoke token and propagate
	go func() {
		defer wg.Done()
		time.Sleep(time.Second) // Wait for token creation

		// Get most recent token from events
		event := <-events
		if err := node2.RevokeToken(ctx, event.TokenID); err != nil {
			log.Printf("Node 2 failed to revoke token: %v", err)
			return
		}
		fmt.Printf("Node 2 revoked token: %s\n", event.TokenID)
	}()

	// Node 3: Verify revocation
	go func() {
		defer wg.Done()
		time.Sleep(2 * time.Second) // Wait for revocation

		// Try to verify the revoked token
		event := <-events
		t := &token.Token{ID: event.TokenID}
		if err := node3.validator.Validate(ctx, t); err != nil {
			fmt.Printf("Node 3 confirms token %s is revoked\n", event.TokenID)
		}
	}()

	wg.Wait()
	fmt.Println("Cluster operations completed")
}
