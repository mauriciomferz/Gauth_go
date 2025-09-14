package main

import (
	"context"
	"fmt"
	"log"
	"time"

	token "github.com/Gimel-Foundation/gauth/pkg/token"
)

type ClusterNode struct {
	id        string
	store     token.Store
	blacklist *token.Blacklist
	validator *token.ValidationChain
	// jwtManager removed
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
		// jwtManager removed
		tokenEvents: sharedChan,
	}

	node.validator = token.NewValidationChain(token.ValidationConfig{
		AllowedIssuers: []string{fmt.Sprintf("node-%s", id)},
		ClockSkew:      2 * time.Minute,
	}, blacklist)
	go node.handleEvents()
	return node
}

func (n *ClusterNode) handleEvents() {
	for event := range n.tokenEvents {
		switch event.Type {
		case "revoked":
			n.handleRevocation(event.TokenID)
		}
	}
}

func (n *ClusterNode) handleRevocation(tokenID string) {
	fmt.Printf("Node %s: Received revocation for token %s\n", n.id, tokenID)
}

func (n *ClusterNode) CreateToken(ctx context.Context, subject string) (*token.Token, string, error) {
	t := &token.Token{
		ID:        token.NewID(),
		Type:      token.Access,
		Subject:   subject,
		Issuer:    fmt.Sprintf("node-%s", n.id),
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Scopes:    []string{"read", "write"},
		Metadata: &token.Metadata{
			AppData: map[string]string{
				"node": n.id,
			},
		},
	}

	if err := n.store.Save(ctx, t.ID, t); err != nil {
		return nil, "", err
	}

	n.tokenEvents <- TokenEvent{
		Type:      "created",
		TokenID:   t.ID,
		Timestamp: time.Now(),
	}

	return t, "", nil
}

func main() {
	ctx := context.Background()
	eventChan := make(chan TokenEvent, 10)

	nodeA := NewClusterNode("A", eventChan)
	nodeB := NewClusterNode("B", eventChan)

	// Node A creates a token
	tokenA, signedA, err := nodeA.CreateToken(ctx, "userA")
	if err != nil {
		log.Fatalf("Node A failed to create token: %v", err)
	}
	fmt.Printf("Node A issued token: %s\n", signedA)

	// Node B receives event
	time.Sleep(100 * time.Millisecond)

	// Node B revokes token
	nodeB.tokenEvents <- TokenEvent{Type: "revoked", TokenID: tokenA.ID, Timestamp: time.Now()}

	// Wait for event handling
	time.Sleep(100 * time.Millisecond)
}
