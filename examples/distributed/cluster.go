package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/gauth"
)

// ResourceNode represents a distributed resource node
type ResourceNode struct {
	ID            string
	Region        string
	Capabilities  []string
	IsHealthy     bool
	LastHeartbeat time.Time
}

// DistributedResourceManager manages a cluster of resource nodes
type DistributedResourceManager struct {
	auth         *gauth.GAuth
	nodes        map[string]*ResourceNode
	nodesMutex   sync.RWMutex
	tokenCache   map[string]time.Time
	cacheMutex   sync.RWMutex
	healthChecks chan string
}

func NewDistributedResourceManager(auth *gauth.GAuth) *DistributedResourceManager {
	drm := &DistributedResourceManager{
		auth:         auth,
		nodes:        make(map[string]*ResourceNode),
		tokenCache:   make(map[string]time.Time),
		healthChecks: make(chan string, 100),
	}

	// Start background workers
	go drm.healthCheckWorker()
	go drm.tokenCleanupWorker()

	return drm
}

// RegisterNode adds a new resource node to the cluster
func (drm *DistributedResourceManager) RegisterNode(node *ResourceNode) error {
	drm.nodesMutex.Lock()
	defer drm.nodesMutex.Unlock()

	// Validate node registration with GAuth
	tx := gauth.TransactionDetails{
		Type:       "node_registration",
		ResourceID: node.ID,
		CustomMetadata: map[string]string{
			"region":       node.Region,
			"capabilities": "," + string(node.Capabilities[0]),
		},
	}

	// Request temporary token for node
	authReq := gauth.AuthorizationRequest{
		ClientID: "cluster-manager",
		Scopes:   []string{"node:register", "transaction:execute"},
	}

	grant, err := drm.auth.InitiateAuthorization(authReq)
	if err != nil {
		return err
	}

	tokenResp, err := drm.auth.RequestToken(gauth.TokenRequest{
		GrantID: grant.GrantID,
		Scope:   append(grant.Scope, "transaction:execute"),
	})
	if err != nil {
		return err
	}
	log.Printf("Granted token scopes: %v", tokenResp.Scope)

	// Process registration
	server := gauth.NewResourceServer("cluster-manager", drm.auth)
	_, err = server.ProcessTransaction(tx, tokenResp.Token)
	if err != nil {
		return err
	}

	node.IsHealthy = true
	node.LastHeartbeat = time.Now()
	drm.nodes[node.ID] = node

	// Schedule health checks
	drm.healthChecks <- node.ID
	return nil
}

// FindAvailableNode finds a healthy node with required capabilities
func (drm *DistributedResourceManager) FindAvailableNode(ctx context.Context, capabilities []string) (*ResourceNode, error) {
	drm.nodesMutex.RLock()
	defer drm.nodesMutex.RUnlock()

	for _, node := range drm.nodes {
		if !node.IsHealthy {
			continue
		}

		// Check if node has all required capabilities
		hasAll := true
		for _, required := range capabilities {
			found := false
			for _, available := range node.Capabilities {
				if required == available {
					found = true
					break
				}
			}
			if !found {
				hasAll = false
				break
			}
		}

		if hasAll {
			return node, nil
		}
	}

	return nil, fmt.Errorf("no available node found with required capabilities")
}

// SimulateDistributedAuthorization simulates distributed authorization requests routed to nodes
func SimulateDistributedAuthorization(manager *DistributedResourceManager, numRequests int) {
	nodeIDs := make([]string, 0)
	manager.nodesMutex.RLock()
	for id, node := range manager.nodes {
		if node.IsHealthy {
			nodeIDs = append(nodeIDs, id)
		}
	}
	manager.nodesMutex.RUnlock()

	if len(nodeIDs) == 0 {
		log.Println("No healthy nodes available for simulation.")
		return
	}

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < numRequests; i++ {
		nodeID := nodeIDs[rnd.Intn(len(nodeIDs))]
		// No need to fetch node object here; nodeID is sufficient for simulation

		// Simulate a resource access request
		resourceID := fmt.Sprintf("resource-%d", i+1)
		userID := fmt.Sprintf("user-%d", rnd.Intn(3)+1) // user-1, user-2, user-3

		tx := gauth.TransactionDetails{
			Type:       "resource_access",
			ResourceID: resourceID,
			CustomMetadata: map[string]string{
				"requested_by": userID,
				"node":         nodeID,
			},
		}

		authReq := gauth.AuthorizationRequest{
			ClientID: userID,
			Scopes:   []string{"resource:access"},
		}

		grant, err := manager.auth.InitiateAuthorization(authReq)
		if err != nil {
			log.Printf("[Sim] Node %s failed to initiate auth for %s: %v", nodeID, userID, err)
			continue
		}

		tokenResp, err := manager.auth.RequestToken(gauth.TokenRequest{
			GrantID: grant.GrantID,
			Scope:   grant.Scope,
		})
		if err != nil {
			log.Printf("[Sim] Node %s failed to get token for %s: %v", nodeID, userID, err)
			continue
		}

		server := gauth.NewResourceServer(nodeID, manager.auth)
		result, err := server.ProcessTransaction(tx, tokenResp.Token)
		if err != nil {
			log.Printf("[Sim] Node %s denied access to %s for %s: %v", nodeID, userID, resourceID, err)
		} else {
			log.Printf("[Sim] Node %s ALLOWED access to %s for %s: %v", nodeID, userID, resourceID, result)
		}
	}
}

// healthCheckWorker performs periodic health checks on nodes
func (drm *DistributedResourceManager) healthCheckWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case nodeID := <-drm.healthChecks:
			drm.checkNodeHealth(nodeID)
		case <-ticker.C:
			drm.nodesMutex.RLock()
			for nodeID := range drm.nodes {
				drm.healthChecks <- nodeID
			}
			drm.nodesMutex.RUnlock()
		}
	}
}

// tokenCleanupWorker removes expired tokens from cache
func (drm *DistributedResourceManager) tokenCleanupWorker() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		drm.cacheMutex.Lock()
		now := time.Now()
		for token, expiry := range drm.tokenCache {
			if now.After(expiry) {
				delete(drm.tokenCache, token)
			}
		}
		drm.cacheMutex.Unlock()
	}
}

func (drm *DistributedResourceManager) checkNodeHealth(nodeID string) {
	drm.nodesMutex.Lock()
	defer drm.nodesMutex.Unlock()

	node, exists := drm.nodes[nodeID]
	if !exists {
		return
	}

	// Check if node has missed too many heartbeats
	if time.Since(node.LastHeartbeat) > 2*time.Minute {
		node.IsHealthy = false
		log.Printf("Node %s marked unhealthy: missed heartbeats", nodeID)
	}
}

func main() {
	// Initialize GAuth
	config := gauth.Config{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "cluster-manager",
		ClientSecret:      "cluster-secret",
		Scopes:            []string{"node:register", "node:manage", "transaction:execute"},
		AccessTokenExpiry: 24 * time.Hour,
	}

	auth, err := gauth.New(config)
	if err != nil {
		log.Fatalf("Failed to initialize GAuth: %v", err)
	}

	// Create distributed resource manager
	manager := NewDistributedResourceManager(auth)

	// Register some example nodes
	nodes := []*ResourceNode{
		{
			ID:           "node-1",
			Region:       "EU-west",
			Capabilities: []string{"compute", "storage"},
		},
		{
			ID:           "node-2",
			Region:       "EU-east",
			Capabilities: []string{"compute", "memory"},
		},
	}

	for _, node := range nodes {
		if err := manager.RegisterNode(node); err != nil {
			log.Printf("Failed to register node %s: %v", node.ID, err)
			continue
		}
		log.Printf("Registered node: %s", node.ID)
	}

	// Example: Find a node with specific capabilities
	ctx := context.Background()
	node, err := manager.FindAvailableNode(ctx, []string{"compute", "storage"})
	if err != nil {
		log.Printf("Failed to find node: %v", err)
	} else {
		log.Printf("Found suitable node: %s in region %s", node.ID, node.Region)
	}

	// Simulate distributed authorization requests
	log.Println("\n--- Simulating distributed authorization requests ---")
	SimulateDistributedAuthorization(manager, 5)

	// Keep the program running
	select {}
}
