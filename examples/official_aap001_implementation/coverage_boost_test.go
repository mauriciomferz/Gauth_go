package main

import "testing"

func TestChooseDelegationScope(t *testing.T) {
	cases := []struct {
		base     []string
		restrict bool
		wantLen  int
	}{
		{nil, false, 1},                              // default
		{[]string{"a", "b", "c"}, true, 2},           // restrict truncate
		{[]string{"a"}, true, 1},                     // restrict copy
		{[]string{"admin:root"}, false, 1},           // sanitized path
		{[]string{"x", "admin:root", "y"}, false, 2}, // filter admin
	}
	for i, tc := range cases {
		got := chooseDelegationScope(tc.base, tc.restrict)
		if len(got) != tc.wantLen {
			t.Fatalf("case %d: want len %d got %d (%v)", i, tc.wantLen, len(got), got)
		}
	}
}

func TestComputeRestrictionFactor(t *testing.T) {
	vals := map[float64]float64{
		-5:     0,
		0:      0,
		50:     0.1,
		150:    0.3,
		500:    0.3,
		5000:   0.6,
		9999.9: 0.6,
		15000:  0.9,
	}
	for in, want := range vals {
		got := computeRestrictionFactor(in)
		if got != want {
			t.Fatalf("amount %f: want %f got %f", in, want, got)
		}
	}
}
