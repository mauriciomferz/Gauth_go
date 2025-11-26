package policy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	pkgpolicy "github.com/mauriciomferz/Gauth_go/pkg/policy"
)

func TestBoltPolicyVersionStore_SaveLoadVersion(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_versions.db")

	// Create store
	store, err := NewBoltPolicyVersionStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create test bundle
	bundle := pkgpolicy.Bundle{
		ID:      "test-bundle-v1",
		Version: 1,
		Policies: []pkgpolicy.Policy{
			{
				ID:       "policy1",
				Subjects: []string{"alice"},
				Rules: []pkgpolicy.Rule{
					{
						Actions:   []string{"read"},
						Resources: []string{"/data/*"},
						Effect:    pkgpolicy.Allow,
					},
				},
			},
		},
	}

	// Create metadata
	metadata := &PolicyVersionMetadata{
		SemanticVersion: SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		Name:            "Test Version 1",
		BundleVersion:   1,
		CreatedAt:       time.Now(),
	}

	// Save version
	if err2 := store.SaveVersion(1, bundle, metadata); err2 != nil {
		t.Fatalf("Failed to save version: %v", err)
	}

	// Load version
	loadedBundle, loadedMetadata, err := store.LoadVersion(1)
	if err != nil {
		t.Fatalf("Failed to load version: %v", err)
	}

	// Verify bundle
	if loadedBundle.ID != bundle.ID {
		t.Errorf("Expected bundle ID %s, got %s", bundle.ID, loadedBundle.ID)
	}
	if loadedBundle.Version != bundle.Version {
		t.Errorf("Expected bundle version %d, got %d", bundle.Version, loadedBundle.Version)
	}

	// Verify metadata
	if loadedMetadata.Name != metadata.Name {
		t.Errorf("Expected metadata name %s, got %s", metadata.Name, loadedMetadata.Name)
	}
	if loadedMetadata.SemanticVersion.Major != metadata.SemanticVersion.Major {
		t.Errorf("Expected major version %d, got %d", metadata.SemanticVersion.Major, loadedMetadata.SemanticVersion.Major)
	}
}

func TestBoltPolicyVersionStore_ListVersions(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_versions.db")

	store, err := NewBoltPolicyVersionStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Save 3 versions
	for i := 1; i <= 3; i++ {
		bundle := pkgpolicy.Bundle{
			ID:      fmt.Sprintf("bundle-v%d", i),
			Version: i,
		}
		metadata := &PolicyVersionMetadata{
			SemanticVersion: SemanticVersion{Major: 1, Minor: i - 1, Patch: 0},
			Name:            fmt.Sprintf("Version %d", i),
			BundleVersion:   i,
		}
		if err2 := store.SaveVersion(i, bundle, metadata); err2 != nil {
			t.Fatalf("Failed to save version %d: %v", i, err)
		}
	}

	// List versions
	versions, err := store.ListVersions()
	if err != nil {
		t.Fatalf("Failed to list versions: %v", err)
	}

	if len(versions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(versions))
	}

	// Verify versions are correct
	expectedVersions := map[int]bool{1: true, 2: true, 3: true}
	for _, v := range versions {
		if !expectedVersions[v] {
			t.Errorf("Unexpected version %d", v)
		}
		delete(expectedVersions, v)
	}

	if len(expectedVersions) > 0 {
		t.Errorf("Missing versions: %v", expectedVersions)
	}
}

func TestBoltPolicyVersionStore_ActiveVersion(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_versions.db")

	store, err := NewBoltPolicyVersionStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Save active version
	if err2 := store.SaveActiveVersion(2); err2 != nil {
		t.Fatalf("Failed to save active version: %v", err)
	}

	// Load active version
	activeVersion, err := store.LoadActiveVersion()
	if err != nil {
		t.Fatalf("Failed to load active version: %v", err)
	}

	if activeVersion != 2 {
		t.Errorf("Expected active version 2, got %d", activeVersion)
	}

	// Update active version
	if err2 := store.SaveActiveVersion(3); err2 != nil {
		t.Fatalf("Failed to update active version: %v", err)
	}

	activeVersion, err = store.LoadActiveVersion()
	if err != nil {
		t.Fatalf("Failed to load updated active version: %v", err)
	}

	if activeVersion != 3 {
		t.Errorf("Expected active version 3, got %d", activeVersion)
	}
}

func TestBoltPolicyVersionStore_AuditEvents(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_versions.db")

	store, err := NewBoltPolicyVersionStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Save audit events for different versions
	events := []VersionAuditEvent{
		{
			EventType: "version_created",
			Version:   1,
			Timestamp: time.Now(),
			Success:   true,
		},
		{
			EventType: "version_activated",
			Version:   1,
			Timestamp: time.Now().Add(1 * time.Minute),
			Success:   true,
		},
		{
			EventType: "version_created",
			Version:   2,
			Timestamp: time.Now().Add(2 * time.Minute),
			Success:   true,
		},
	}

	for _, event := range events {
		if err2 := store.SaveAuditEvent(event); err2 != nil {
			t.Fatalf("Failed to save audit event: %v", err)
		}
		// Small delay to ensure unique event IDs (timestamp-based)
		time.Sleep(1 * time.Millisecond)
	}

	// Load all events
	allEvents, err := store.LoadAuditEvents(0)
	if err != nil {
		t.Fatalf("Failed to load all audit events: %v", err)
	}

	if len(allEvents) != 3 {
		t.Errorf("Expected 3 audit events, got %d", len(allEvents))
	}

	// Load events for version 1
	v1Events, err := store.LoadAuditEvents(1)
	if err != nil {
		t.Fatalf("Failed to load version 1 audit events: %v", err)
	}

	if len(v1Events) != 2 {
		t.Errorf("Expected 2 events for version 1, got %d", len(v1Events))
	}

	// Load events for version 2
	v2Events, err := store.LoadAuditEvents(2)
	if err != nil {
		t.Fatalf("Failed to load version 2 audit events: %v", err)
	}

	if len(v2Events) != 1 {
		t.Errorf("Expected 1 event for version 2, got %d", len(v2Events))
	}
}

func TestBoltPolicyVersionStore_Stats(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_versions.db")

	store, err := NewBoltPolicyVersionStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Save 2 versions
	for i := 1; i <= 2; i++ {
		bundle := pkgpolicy.Bundle{
			ID:      fmt.Sprintf("bundle-v%d", i),
			Version: i,
		}
		metadata := &PolicyVersionMetadata{
			Name:          fmt.Sprintf("Version %d", i),
			BundleVersion: i,
		}
		if err2 := store.SaveVersion(i, bundle, metadata); err2 != nil {
			t.Fatalf("Failed to save version %d: %v", i, err)
		}
	}

	// Save active version
	if err2 := store.SaveActiveVersion(2); err2 != nil {
		t.Fatalf("Failed to save active version: %v", err)
	}

	// Save audit event
	event := VersionAuditEvent{
		EventType: "version_created",
		Version:   1,
		Timestamp: time.Now(),
		Success:   true,
	}
	if err2 := store.SaveAuditEvent(event); err2 != nil {
		t.Fatalf("Failed to save audit event: %v", err)
	}

	// Get stats
	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.TotalVersions != 2 {
		t.Errorf("Expected 2 versions, got %d", stats.TotalVersions)
	}
	if stats.TotalBundles != 2 {
		t.Errorf("Expected 2 bundles, got %d", stats.TotalBundles)
	}
	if stats.TotalAuditEvents != 1 {
		t.Errorf("Expected 1 audit event, got %d", stats.TotalAuditEvents)
	}
	if stats.ActiveVersion != 2 {
		t.Errorf("Expected active version 2, got %d", stats.ActiveVersion)
	}
}

func TestPolicyVersionManagerWithStore_CreateAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_versions.db")

	// Create store
	store, err := NewBoltPolicyVersionStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create registry and version manager
	registry := pkgpolicy.NewRegistry()
	manager, err := NewPolicyVersionManagerWithStore(registry, store)
	if err != nil {
		t.Fatalf("Failed to create version manager with store: %v", err)
	}

	ctx := context.Background()

	// Create version
	bundle := pkgpolicy.Bundle{
		ID: "test-bundle-v1",
		Policies: []pkgpolicy.Policy{
			{
				ID:       "policy1",
				Subjects: []string{"alice"},
				Rules: []pkgpolicy.Rule{
					{
						Actions:   []string{"read"},
						Resources: []string{"/data/*"},
						Effect:    pkgpolicy.Allow,
					},
				},
			},
		},
	}

	metadata := PolicyVersionMetadata{
		SemanticVersion: SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		Name:            "Test Version 1",
	}

	createdMetadata, err := manager.CreateVersion(ctx, bundle, metadata)
	if err != nil {
		t.Fatalf("Failed to create version: %v", err)
	}

	if createdMetadata.BundleVersion != 1 {
		t.Errorf("Expected bundle version 1, got %d", createdMetadata.BundleVersion)
	}

	// Close store
	store.Close()

	// Reopen store and create new manager (simulates restart)
	store2, err := NewBoltPolicyVersionStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen store: %v", err)
	}
	defer store2.Close()

	registry2 := pkgpolicy.NewRegistry()
	manager2, err := NewPolicyVersionManagerWithStore(registry2, store2)
	if err != nil {
		t.Fatalf("Failed to create second version manager: %v", err)
	}

	// Verify version was loaded
	loadedMetadata, err := manager2.GetVersionMetadata(1)
	if err != nil {
		t.Fatalf("Failed to get version metadata: %v", err)
	}

	if loadedMetadata.Name != "Test Version 1" {
		t.Errorf("Expected name 'Test Version 1', got '%s'", loadedMetadata.Name)
	}

	// Verify active version was loaded
	activeVersion := manager2.GetActiveVersion()
	if activeVersion != 1 {
		t.Errorf("Expected active version 1, got %d", activeVersion)
	}
}

func TestPolicyVersionManagerWithStore_RollbackPersistence(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_versions.db")

	store, err := NewBoltPolicyVersionStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	registry := pkgpolicy.NewRegistry()
	manager, err := NewPolicyVersionManagerWithStore(registry, store)
	if err != nil {
		t.Fatalf("Failed to create version manager: %v", err)
	}

	ctx := context.Background()

	// Create 3 versions
	for i := 1; i <= 3; i++ {
		bundle := pkgpolicy.Bundle{
			ID: fmt.Sprintf("bundle-v%d", i),
			Policies: []pkgpolicy.Policy{
				{
					ID:       "policy1",
					Subjects: []string{"alice"},
					Rules: []pkgpolicy.Rule{
						{
							Actions:   []string{"read"},
							Resources: []string{"/data/*"},
							Effect:    pkgpolicy.Allow,
						},
					},
				},
			},
		}

		metadata := PolicyVersionMetadata{
			SemanticVersion: SemanticVersion{Major: 1, Minor: i - 1, Patch: 0},
			Name:            fmt.Sprintf("Version %d", i),
			RollbackAllowed: true,
		}

		if _, err2 := manager.CreateVersion(ctx, bundle, metadata); err2 != nil {
			t.Fatalf("Failed to create version %d: %v", i, err)
		}

		// Activate each version
		if err2 := manager.ActivateVersion(ctx, i, "system"); err2 != nil {
			t.Fatalf("Failed to activate version %d: %v", i, err)
		}
	}

	// Rollback to version 2
	if err2 := manager.RollbackVersion(ctx, 2, "admin", "Testing rollback"); err2 != nil {
		t.Fatalf("Failed to rollback: %v", err)
	}

	// Verify active version is 2
	activeVersion := manager.GetActiveVersion()
	if activeVersion != 2 {
		t.Errorf("Expected active version 2 after rollback, got %d", activeVersion)
	}

	// Close and reopen (simulates restart)
	store.Close()

	store2, err := NewBoltPolicyVersionStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen store: %v", err)
	}
	defer store2.Close()

	registry2 := pkgpolicy.NewRegistry()
	manager2, err := NewPolicyVersionManagerWithStore(registry2, store2)
	if err != nil {
		t.Fatalf("Failed to create second manager: %v", err)
	}

	// Verify active version persisted
	activeVersion2 := manager2.GetActiveVersion()
	if activeVersion2 != 2 {
		t.Errorf("Expected active version 2 after restart, got %d", activeVersion2)
	}

	// Verify rollback audit event was persisted
	events, err := store2.LoadAuditEvents(2)
	if err != nil {
		t.Fatalf("Failed to load audit events: %v", err)
	}

	found := false
	for _, event := range events {
		if event.EventType == "rollback" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Rollback audit event not found after restart")
	}
}

func TestPolicyVersionManagerWithStore_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_versions.db")

	store, err := NewBoltPolicyVersionStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	registry := pkgpolicy.NewRegistry()
	manager, err := NewPolicyVersionManagerWithStore(registry, store)
	if err != nil {
		t.Fatalf("Failed to create version manager: %v", err)
	}

	ctx := context.Background()

	// Create initial version
	bundle := pkgpolicy.Bundle{
		ID: "bundle-v1",
		Policies: []pkgpolicy.Policy{
			{
				ID:       "policy1",
				Subjects: []string{"alice"},
				Rules: []pkgpolicy.Rule{
					{
						Actions:   []string{"read"},
						Resources: []string{"/data/*"},
						Effect:    pkgpolicy.Allow,
					},
				},
			},
		},
	}

	metadata := PolicyVersionMetadata{
		SemanticVersion: SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		Name:            "Version 1",
	}

	if _, err := manager.CreateVersion(ctx, bundle, metadata); err != nil {
		t.Fatalf("Failed to create initial version: %v", err)
	}

	// Concurrently read version metadata
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				_, err := manager.GetVersionMetadata(1)
				if err != nil {
					t.Errorf("Failed to get version metadata: %v", err)
				}
				time.Sleep(1 * time.Millisecond)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestPolicyVersionManagerWithStore_NilStore(t *testing.T) {
	// Verify that manager works without store (backward compatibility)
	registry := pkgpolicy.NewRegistry()
	manager := NewPolicyVersionManager(registry) // No store

	ctx := context.Background()

	bundle := pkgpolicy.Bundle{
		ID: "bundle-v1",
		Policies: []pkgpolicy.Policy{
			{
				ID:       "policy1",
				Subjects: []string{"alice"},
				Rules: []pkgpolicy.Rule{
					{
						Actions:   []string{"read"},
						Resources: []string{"/data/*"},
						Effect:    pkgpolicy.Allow,
					},
				},
			},
		},
	}

	metadata := PolicyVersionMetadata{
		SemanticVersion: SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		Name:            "Test Version",
	}

	// Should work without store
	createdMetadata, err := manager.CreateVersion(ctx, bundle, metadata)
	if err != nil {
		t.Fatalf("Failed to create version without store: %v", err)
	}

	if createdMetadata.BundleVersion != 1 {
		t.Errorf("Expected bundle version 1, got %d", createdMetadata.BundleVersion)
	}
}

func TestBoltPolicyVersionStore_NonExistentVersion(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_versions.db")

	store, err := NewBoltPolicyVersionStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Try to load non-existent version
	_, _, err = store.LoadVersion(999)
	if err == nil {
		t.Error("Expected error when loading non-existent version")
	}
}

func TestBoltPolicyVersionStore_EmptyDatabase(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_versions.db")

	store, err := NewBoltPolicyVersionStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// List versions should return empty
	versions, err := store.ListVersions()
	if err != nil {
		t.Fatalf("Failed to list versions: %v", err)
	}

	if len(versions) != 0 {
		t.Errorf("Expected 0 versions in empty database, got %d", len(versions))
	}

	// Load active version should return error
	_, err = store.LoadActiveVersion()
	if err == nil {
		t.Error("Expected error when loading active version from empty database")
	}
}

func TestBoltPolicyVersionStore_CorruptedDatabase(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_versions.db")

	// Create a corrupted database file
	if err := os.WriteFile(dbPath, []byte("corrupted data"), 0600); err != nil {
		t.Fatalf("Failed to create corrupted file: %v", err)
	}

	// Attempt to open should fail
	_, err := NewBoltPolicyVersionStore(dbPath)
	if err == nil {
		t.Error("Expected error when opening corrupted database")
	}
}
