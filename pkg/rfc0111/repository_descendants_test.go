package rfc0111

import (
	"testing"
	"time"
)

func TestMemoryRepositoryListDescendants(t *testing.T) {
	repo := newMemoryRepository()

	// Create a hierarchy: root -> child1 -> grandchild1
	//                         -> child2
	now := time.Now().UTC()
	
	// Root POA
	root := &PowerOfAttorney{
		ID:          "root-poa-1",
		Grantor:     "alice",
		Grantee:     "bob",
		Scope:       []string{"read"},
		ValidFrom:   now,
		ValidUntil:  now.Add(time.Hour),
		Status:      POAStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
		ParentPOAID: "", // No parent - this is a root
		Depth:       0,
	}
	
	// Child POA 1
	child1 := &PowerOfAttorney{
		ID:          "child-poa-1",
		Grantor:     "bob",
		Grantee:     "charlie",
		Scope:       []string{"read"},
		ValidFrom:   now,
		ValidUntil:  now.Add(time.Hour),
		Status:      POAStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
		ParentPOAID: "root-poa-1",
		Depth:       1,
	}
	
	// Child POA 2 (sibling of child1)
	child2 := &PowerOfAttorney{
		ID:          "child-poa-2",
		Grantor:     "bob",
		Grantee:     "dave",
		Scope:       []string{"write"},
		ValidFrom:   now,
		ValidUntil:  now.Add(time.Hour),
		Status:      POAStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
		ParentPOAID: "root-poa-1",
		Depth:       1,
	}
	
	// Grandchild POA
	grandchild1 := &PowerOfAttorney{
		ID:          "grandchild-poa-1",
		Grantor:     "charlie",
		Grantee:     "eve",
		Scope:       []string{"read"},
		ValidFrom:   now,
		ValidUntil:  now.Add(time.Hour),
		Status:      POAStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
		ParentPOAID: "child-poa-1",
		Depth:       2,
	}

	// Store all POAs
	_ = repo.Create(root)
	_ = repo.Create(child1)
	_ = repo.Create(child2)
	_ = repo.Create(grandchild1)

	t.Run("find all descendants unlimited depth", func(t *testing.T) {
		descendants, err := repo.ListDescendants("root-poa-1", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		if len(descendants) != 3 {
			t.Errorf("expected 3 descendants, got %d", len(descendants))
		}
		
		// Check that we found all expected descendants
		ids := make(map[string]bool)
		for _, d := range descendants {
			ids[d.ID] = true
		}
		
		expected := []string{"child-poa-1", "child-poa-2", "grandchild-poa-1"}
		for _, expectedID := range expected {
			if !ids[expectedID] {
				t.Errorf("expected to find descendant %s", expectedID)
			}
		}
	})

	t.Run("find descendants with depth limit", func(t *testing.T) {
		descendants, err := repo.ListDescendants("root-poa-1", 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		if len(descendants) != 2 {
			t.Errorf("expected 2 descendants (depth 1 only), got %d", len(descendants))
		}
		
		// Check that we only found direct children
		ids := make(map[string]bool)
		for _, d := range descendants {
			ids[d.ID] = true
		}
		
		expected := []string{"child-poa-1", "child-poa-2"}
		for _, expectedID := range expected {
			if !ids[expectedID] {
				t.Errorf("expected to find direct child %s", expectedID)
			}
		}
		
		// Should not find grandchild
		if ids["grandchild-poa-1"] {
			t.Errorf("should not find grandchild with depth limit 1")
		}
	})

	t.Run("find descendants of child POA", func(t *testing.T) {
		descendants, err := repo.ListDescendants("child-poa-1", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		if len(descendants) != 1 {
			t.Errorf("expected 1 descendant of child-poa-1, got %d", len(descendants))
		}
		
		if descendants[0].ID != "grandchild-poa-1" {
			t.Errorf("expected grandchild-poa-1, got %s", descendants[0].ID)
		}
	})

	t.Run("no descendants", func(t *testing.T) {
		descendants, err := repo.ListDescendants("grandchild-poa-1", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		if len(descendants) != 0 {
			t.Errorf("expected no descendants of leaf node, got %d", len(descendants))
		}
	})

	t.Run("nonexistent parent", func(t *testing.T) {
		descendants, err := repo.ListDescendants("nonexistent-poa", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		if len(descendants) != 0 {
			t.Errorf("expected no descendants for nonexistent parent, got %d", len(descendants))
		}
	})

	t.Run("empty parent ID", func(t *testing.T) {
		descendants, err := repo.ListDescendants("", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		if len(descendants) != 0 {
			t.Errorf("expected no descendants for empty parent ID, got %d", len(descendants))
		}
	})

	t.Run("cycle detection", func(t *testing.T) {
		// Create a cycle: cycle1 -> cycle2 -> cycle1
		cycle1 := &PowerOfAttorney{
			ID:          "cycle-poa-1",
			Grantor:     "alice",
			Grantee:     "bob",
			Scope:       []string{"read"},
			ValidFrom:   now,
			ValidUntil:  now.Add(time.Hour),
			Status:      POAStatusActive,
			CreatedAt:   now,
			UpdatedAt:   now,
			ParentPOAID: "cycle-poa-2", // Points to cycle2
			Depth:       1,
		}
		
		cycle2 := &PowerOfAttorney{
			ID:          "cycle-poa-2",
			Grantor:     "bob",
			Grantee:     "charlie",
			Scope:       []string{"read"},
			ValidFrom:   now,
			ValidUntil:  now.Add(time.Hour),
			Status:      POAStatusActive,
			CreatedAt:   now,
			UpdatedAt:   now,
			ParentPOAID: "cycle-poa-1", // Points back to cycle1
			Depth:       2,
		}

		_ = repo.Create(cycle1)
		_ = repo.Create(cycle2)

		// This should not infinite loop due to cycle detection
		descendants, err := repo.ListDescendants("cycle-poa-1", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		// Should find at most 1 descendant (cycle2) before hitting the cycle
		if len(descendants) > 1 {
			t.Errorf("cycle detection failed, found %d descendants", len(descendants))
		}
	})
}