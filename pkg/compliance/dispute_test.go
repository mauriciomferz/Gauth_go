package compliance

import "testing"

func TestDisputeRegistry_EscalateAndResolve(t *testing.T) {
	dr, err := NewDisputeRegistry("")
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	meta := DisputeMetadata{
		ID:           "dispute-1",
		FlowID:       "flow-1",
		Jurisdiction: JurisdictionUS,
		Entity:       EntityTypeCorporation,
		Reason:       "compliance violation",
		Status:       "pending",
	}
	if err := dr.EscalateDispute(meta); err != nil {
		t.Fatalf("failed to escalate dispute: %v", err)
	}

	d, ok := dr.GetDispute(meta.ID)
	if !ok || d.Status != "escalated" {
		t.Fatalf("dispute should be escalated, got status: %s", d.Status)
	}

	if err := dr.ResolveDispute(meta.ID, "arbitrator-xyz", "resolved by arbitration"); err != nil {
		t.Fatalf("dispute should resolve successfully: %v", err)
	}

	d, _ = dr.GetDispute(meta.ID)
	if d.Status != "resolved" || d.Arbitrator != "arbitrator-xyz" {
		t.Fatalf("dispute should be marked resolved and arbitrator set")
	}
}

func TestDisputeRegistry_ListDisputes(t *testing.T) {
	dr, err := NewDisputeRegistry("")
	if err != nil {
		t.Fatalf("failed to create registry: %v", err)
	}

	for i := 1; i <= 3; i++ {
		meta := DisputeMetadata{
			ID:           "dispute-list-" + string(rune(i)),
			FlowID:       "flow-list-" + string(rune(i)),
			Jurisdiction: JurisdictionEU,
			Entity:       EntityTypeOrganization,
			Reason:       "test reason",
			Status:       "pending",
		}
		if err := dr.EscalateDispute(meta); err != nil {
			t.Fatalf("failed to escalate: %v", err)
		}
	}
	list := dr.ListDisputes()
	if len(list) != 3 {
		t.Fatalf("expected 3 disputes, got %d", len(list))
	}
}
