package testutil

import "testing"

// TestCapabilityRegistryTable centralizes validation expectations for fixtures.
func TestCapabilityRegistryTable(t *testing.T) {
	cases := []struct {
		name        string
		raw         string
		wantErr     error  // specific error expected (match by equality), nil for success
		wantSub     string // substring match when wantErr is nil but custom error message expected
		shouldParse bool   // whether ParseCapabilityRegistry should succeed
	}{
		{name: "CapTransferV1", raw: CapTransferV1, shouldParse: true},
		{name: "CapTransferIssueV1", raw: CapTransferIssueV1, shouldParse: true},
		{name: "CapTransferIssueDelegationCreateV1", raw: CapTransferIssueDelegationCreateV1, shouldParse: true},
		{name: "CapAlphaV1", raw: CapAlphaV1, shouldParse: true},
		{name: "CapTransferAuditV1", raw: CapTransferAuditV1, shouldParse: true},
		{name: "CapAlphaMissingSchemaVersion", raw: CapAlphaMissingSchemaVersion, wantErr: ErrMissingSchemaVersion},
		{name: "CapAlphaDuplicateIDs", raw: CapAlphaDuplicateIDs, wantErr: ErrDuplicateCapabilityID},
		{name: "CapAlphaUnknownMapping", raw: CapAlphaUnknownMapping, wantSub: "unknown capability id"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg, err := ParseCapabilityRegistry(tc.raw)
			if tc.shouldParse && err != nil {
				t.Fatalf("expected success; got error %v", err)
			}
			if !tc.shouldParse && tc.wantErr != nil {
				if err != tc.wantErr {
					t.Fatalf("expected error %v; got %v", tc.wantErr, err)
				}
				return
			}
			if tc.wantSub != "" {
				if err == nil || index(err.Error(), tc.wantSub) == -1 {
					t.Fatalf("expected error containing %q; got %v", tc.wantSub, err)
				}
				return
			}
			if tc.shouldParse && reg == nil {
				t.Fatalf("expected non-nil registry")
			}
		})
	}
}

func TestCanonicalRegistryHashConsistency(t *testing.T) {
	raw := CapTransferIssueDelegationCreateV1
	h1 := CanonicalRegistryHash(raw)
	h2 := CanonicalRegistryHash(raw)
	if h1 != h2 {
		t.Fatalf("canonical hash not deterministic: %s vs %s", h1, h2)
	}
	if len(h1) != 64 {
		t.Fatalf("expected 64-char hex hash; got %d", len(h1))
	}
}
