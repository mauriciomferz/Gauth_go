package sync

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicySyncer_EndToEnd(t *testing.T) {
	// Setup temporary file
	tempDir := t.TempDir()
	policyFile := filepath.Join(tempDir, "policies.json")

	// Initial policies
	initialPolicies := []authz.Policy{
		{ID: "p1", Effect: authz.Allow, Actions: []string{"read"}, Resource: "res1"},
	}

	source := NewFilePolicySource(policyFile)
	require.NoError(t, source.UpdateFile(initialPolicies))

	// Setup Authorizer
	authorizer := authz.NewMemoryAuthorizer()

	// Setup Syncer
	syncer := NewPolicySyncer(source, authorizer, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run syncer in background
	go func() {
		_ = syncer.Start(ctx)
	}()

	// Wait for first sync
	time.Sleep(100 * time.Millisecond)

	// Verify initial load
	// We can't inspect policies directly safely without locking, but we can try Authorize
	req := authz.Request{Action: "read", Resource: "res1"}
	dec, err := authorizer.Authorize(ctx, req)
	require.NoError(t, err)
	assert.True(t, dec.Allow, "Initial policy should be loaded and allow access")

	// Update File
	updatedPolicies := []authz.Policy{
		{ID: "p1", Effect: authz.Deny, Actions: []string{"read"}, Resource: "res1"}, // Changed to Deny
		{ID: "p2", Effect: authz.Allow, Actions: []string{"write"}, Resource: "res2"},
	}
	require.NoError(t, source.UpdateFile(updatedPolicies))

	// Wait for sync
	time.Sleep(150 * time.Millisecond)

	// Verify update
	// Check p1 changed to Deny
	dec, err = authorizer.Authorize(ctx, req)
	require.NoError(t, err)
	assert.False(t, dec.Allow, "Updated policy p1 should deny access")

	// Check p2 added
	req2 := authz.Request{Action: "write", Resource: "res2"}
	dec2, err := authorizer.Authorize(ctx, req2)
	require.NoError(t, err)
	assert.True(t, dec2.Allow, "New policy p2 should allow access")
}
