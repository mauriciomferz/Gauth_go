package compliance

import "testing"

func TestPoAValidator_AmountCondition(t *testing.T) {
	v := NewPoAValidator()
	conds := []PoACondition{{Type: "amount", Expr: ">=1000"}}
	ctx := map[string]interface{}{"amount": 1500.0}
	if err := v.Validate(conds, ctx); err != nil {
		t.Fatalf("should pass amount >=1000: %v", err)
	}
	ctx["amount"] = 500.0
	if err := v.Validate(conds, ctx); err == nil {
		t.Fatalf("should fail amount >=1000")
	}
}

func TestPoAValidator_TimeCondition(t *testing.T) {
	v := NewPoAValidator()
	conds := []PoACondition{{Type: "time", Expr: "between(09:00,17:00)"}}
	ctx := map[string]interface{}{"time": "10:30"}
	if err := v.Validate(conds, ctx); err != nil {
		t.Fatalf("should pass time between 09:00-17:00: %v", err)
	}
	ctx["time"] = "18:00"
	if err := v.Validate(conds, ctx); err == nil {
		t.Fatalf("should fail time between 09:00-17:00")
	}
}

func TestPoAValidator_CustomCondition(t *testing.T) {
	v := NewPoAValidator()
	conds := []PoACondition{{Type: "custom", Expr: "custom:approved"}}
	ctx := map[string]interface{}{"custom": "approved"}
	if err := v.Validate(conds, ctx); err != nil {
		t.Fatalf("should pass custom:approved: %v", err)
	}
	ctx["custom"] = "denied"
	if err := v.Validate(conds, ctx); err == nil {
		t.Fatalf("should fail custom:approved")
	}
}
