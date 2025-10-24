package policy

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/policy"
)

func TestNewPolicyVersionManager(t *testing.T) {
	registry := policy.NewRegistry()
	manager := NewPolicyVersionManager(registry)

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}

	if manager.GetActiveVersion() != 0 {
		t.Errorf("expected active version 0, got %d", manager.GetActiveVersion())
	}
}

func TestCreateVersion_BasicScenario(t *testing.T) {
	registry := policy.NewRegistry()
	manager := NewPolicyVersionManager(registry)
	ctx := context.Background()

	bundle := policy.Bundle{
		ID: "test-bundle-v1",
		Policies: []policy.Policy{
			{
				ID:       "policy1",
				Subjects: []string{"alice"},
				Rules: []policy.Rule{
					{
						Actions:   []string{"read"},
						Resources: []string{"/docs/*"},
						Effect:    policy.Allow,
					},
				},
			},
		},
	}

	metadata := PolicyVersionMetadata{
		SemanticVersion: SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		Name:            "Initial Version",
		Description:     "First policy version",
		Author:          "test-author",
		RollbackAllowed: true,
	}

	createdMeta, err := manager.CreateVersion(ctx, bundle, metadata)
	if err != nil {
		t.Fatalf("CreateVersion failed: %v", err)
	}

	if createdMeta.BundleVersion != 1 {
		t.Errorf("expected bundle version 1, got %d", createdMeta.BundleVersion)
	}

	if createdMeta.SemanticVersion.String() != "1.0.0" {
		t.Errorf("expected semantic version 1.0.0, got %s", createdMeta.SemanticVersion.String())
	}

	if createdMeta.Hash == "" {
		t.Error("expected non-empty hash")
	}

	if manager.GetActiveVersion() != 1 {
		t.Errorf("expected active version 1, got %d", manager.GetActiveVersion())
	}
}

func TestActivateVersion(t *testing.T) {
	registry := policy.NewRegistry()
	manager := NewPolicyVersionManager(registry)
	ctx := context.Background()

	for i := 1; i <= 2; i++ {
		bundle := policy.Bundle{
			ID: fmt.Sprintf("bundle-v%d", i),
			Policies: []policy.Policy{
				{
					ID:       "policy1",
					Subjects: []string{"alice"},
					Rules: []policy.Rule{
						{
							Actions:   []string{"read"},
							Resources: []string{"/data/*"},
							Effect:    policy.Allow,
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

		_, err := manager.CreateVersion(ctx, bundle, metadata)
		if err != nil {
			t.Fatalf("CreateVersion v%d failed: %v", i, err)
		}
	}

	if manager.GetActiveVersion() != 1 {
		t.Errorf("expected active version 1, got %d", manager.GetActiveVersion())
	}

	err := manager.ActivateVersion(ctx, 2, "test-actor")
	if err != nil {
		t.Fatalf("ActivateVersion failed: %v", err)
	}

	if manager.GetActiveVersion() != 2 {
		t.Errorf("expected active version 2, got %d", manager.GetActiveVersion())
	}

	metadata, _ := manager.GetVersionMetadata(2)
	if metadata.ActivatedAt == nil {
		t.Error("expected ActivatedAt to be set")
	}
}

func TestRollbackVersion(t *testing.T) {
	registry := policy.NewRegistry()
	manager := NewPolicyVersionManager(registry)
	ctx := context.Background()

	var auditEvents []VersionAuditEvent
	manager.SetAuditCallback(func(event VersionAuditEvent) {
		auditEvents = append(auditEvents, event)
	})

	for i := 1; i <= 3; i++ {
		bundle := policy.Bundle{
			ID: fmt.Sprintf("bundle-v%d", i),
			Policies: []policy.Policy{
				{
					ID:       "policy1",
					Subjects: []string{"alice"},
					Rules: []policy.Rule{
						{
							Actions:   []string{"read"},
							Resources: []string{"/data/*"},
							Effect:    policy.Allow,
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

		_, err := manager.CreateVersion(ctx, bundle, metadata)
		if err != nil {
			t.Fatalf("CreateVersion v%d failed: %v", i, err)
		}
	}

	manager.ActivateVersion(ctx, 3, "test-actor")

	err := manager.RollbackVersion(ctx, 2, "admin", "Testing rollback")
	if err != nil {
		t.Fatalf("RollbackVersion failed: %v", err)
	}

	// Allow asynchronous audit goroutine to complete
	time.Sleep(100 * time.Millisecond)

	if manager.GetActiveVersion() != 2 {
		t.Errorf("expected active version 2 after rollback, got %d", manager.GetActiveVersion())
	}

	found := false
	for i := 0; i < 10 && !found; i++ { // retry loop in case goroutine audit delays
		for _, event := range auditEvents {
			if event.EventType == "rollback" && event.Version == 2 && event.Success {
				found = true
				break
			}
		}
		if !found {
			time.Sleep(20 * time.Millisecond)
		}
	}
	if !found {
		t.Error("expected rollback audit event")
	}
}

func TestSemanticVersion_Compare(t *testing.T) {
	tests := []struct {
		v1       SemanticVersion
		v2       SemanticVersion
		expected int
	}{
		{SemanticVersion{1, 0, 0}, SemanticVersion{1, 0, 0}, 0},
		{SemanticVersion{1, 0, 0}, SemanticVersion{2, 0, 0}, -1},
		{SemanticVersion{2, 0, 0}, SemanticVersion{1, 0, 0}, 1},
		{SemanticVersion{1, 1, 0}, SemanticVersion{1, 0, 0}, 1},
		{SemanticVersion{1, 0, 1}, SemanticVersion{1, 0, 0}, 1},
		{SemanticVersion{1, 2, 3}, SemanticVersion{1, 2, 4}, -1},
	}

	for _, tt := range tests {
		result := tt.v1.Compare(tt.v2)
		if result != tt.expected {
			t.Errorf("Compare(%v, %v) = %d, expected %d", tt.v1, tt.v2, result, tt.expected)
		}
	}
}

func TestParseSemanticVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected SemanticVersion
		wantErr  bool
	}{
		{"1.0.0", SemanticVersion{1, 0, 0}, false},
		{"2.3.4", SemanticVersion{2, 3, 4}, false},
		{"10.20.30", SemanticVersion{10, 20, 30}, false},
		{"invalid", SemanticVersion{}, true},
		{"1.0", SemanticVersion{}, true},
		{"1.0.0.0", SemanticVersion{}, true},
	}

	for _, tt := range tests {
		result, err := ParseSemanticVersion(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseSemanticVersion(%s) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseSemanticVersion(%s) unexpected error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("ParseSemanticVersion(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		}
	}
}
