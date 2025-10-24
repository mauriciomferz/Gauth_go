package compliance

import "testing"

func TestDisputeRegistry_EscalateAndResolve(t *testing.T) {
	dr := NewDisputeRegistry()
	meta := DisputeMetadata{
		ID:           "dispute-1",
		FlowID:       "flow-1",
		Jurisdiction: JurisdictionUS,
		Entity:       EntityTypeCorporation,
		Reason:       "compliance violation",
		Status:       "pending",
	}
	dr.EscalateDispute(meta)
	d, ok := dr.GetDispute(meta.ID)
	if !ok || d.Status != "escalated" {
		t.Fatalf("dispute should be escalated, got status: %s", d.Status)
	}

	resolved := dr.ResolveDispute(meta.ID, "arbitrator-xyz", "resolved by arbitration")
	if !resolved {
		t.Fatalf("dispute should resolve successfully")
	}
	d, _ = dr.GetDispute(meta.ID)
	if d.Status != "resolved" || d.Arbitrator != "arbitrator-xyz" {
		t.Fatalf("dispute should be marked resolved and arbitrator set")
	}
}

func TestDisputeRegistry_ListDisputes(t *testing.T) {
	dr := NewDisputeRegistry()
	for i := 1; i <= 3; i++ {
		meta := DisputeMetadata{
			ID:           "dispute-list-" + string(rune(i)),
			FlowID:       "flow-list-" + string(rune(i)),
			Jurisdiction: JurisdictionEU,
			Entity:       EntityTypeOrganization,
			Reason:       "test reason",
			Status:       "pending",
		}
		dr.EscalateDispute(meta)
	}
	list := dr.ListDisputes()
	if len(list) != 3 {
		t.Fatalf("expected 3 disputes, got %d", len(list))
	}
}
