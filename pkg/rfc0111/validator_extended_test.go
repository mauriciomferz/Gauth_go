package rfc0111

import (
	"testing"
	"time"
)

func TestBasicPoAValidatorJointDelegation(t *testing.T) {
	v := BasicPoAValidator{}
	poa := &PowerOfAttorney{ID: "x", Grantor: "a", Grantee: "b", Scope: []string{"joint:payment"}, ValidFrom: time.Now(), ValidUntil: time.Now().Add(time.Hour)}
	// Missing signatures restriction should fail
	if err := v.Validate(poa); err == nil {
		t.Fatalf("expected error for missing signatures restriction")
	}
	poa.Restrictions = map[string]string{"signatures": "1"}
	if err := v.Validate(poa); err == nil {
		t.Fatalf("expected error for signatures count <2")
	}
	poa.Restrictions["signatures"] = "2"
	if err := v.Validate(poa); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBasicPoAValidatorNumericRestrictions(t *testing.T) {
	v := BasicPoAValidator{}
	base := &PowerOfAttorney{ID: "n", Grantor: "a", Grantee: "b", Scope: []string{"transaction:execute"}, ValidFrom: time.Now(), ValidUntil: time.Now().Add(24 * time.Hour), Restrictions: map[string]string{"currency": "USD"}}
	// Invalid max_amount
	bad := *base
	bad.Restrictions["max_amount"] = "abc"
	if err := v.Validate(&bad); err == nil {
		t.Fatalf("expected error for invalid max_amount")
	}
	// Valid numeric
	good := *base
	good.Restrictions["max_amount"] = "100.50"
	good.Restrictions["max_daily_amount"] = "50.00"
	if err := v.Validate(&good); err == nil {
		t.Fatalf("expected error because max_daily_amount < max_amount")
	}
	good.Restrictions["max_daily_amount"] = "150.00"
	if err := v.Validate(&good); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Currency format
	curBad := *base
	curBad.Restrictions["currency"] = "usd"
	if err := v.Validate(&curBad); err == nil {
		t.Fatalf("expected error for lowercase currency")
	}
}
