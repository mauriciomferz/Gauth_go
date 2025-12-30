package main

import "testing"

func TestBranchingAccumulator(t *testing.T) {
	// n=15 ensures we hit multiples of 5 (0,5,10), 7, and default cases.
	got := branchingAccumulator(15)
	// Precomputed expected value for n=15.
	// We also do a lightweight structural check by comparing with a second call.
	want := 0
	for i := 0; i < 15; i++ {
		if i%2 == 0 {
			want += i
		} else {
			want += i * 2
		}
		if i%3 == 0 {
			want++
		}
		switch {
		case i%5 == 0:
			want += 5
		case i%7 == 0:
			want += 7
		default:
			want++
		}
	}
	if got != want {
		t.Fatalf("branchingAccumulator(15)=%d want=%d", got, want)
	}
	if branchingAccumulator(0) != 0 { // empty loop path
		t.Fatalf("expected 0 for n=0")
	}
}

func TestTieredClassifier(t *testing.T) {
	cases := map[int]string{
		-1: "neg",
		0:  "zero",
		1:  "small-odd",
		2:  "small-even",
		5:  "boundary",
		7:  "mid",
		10: "boundary",
		12: "large-even",
		13: "large-odd",
	}
	for in, want := range cases {
		if got := tieredClassifier(in); got != want {
			t.Fatalf("tieredClassifier(%d)=%s want=%s", in, got, want)
		}
	}
}
