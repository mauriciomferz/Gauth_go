package main

import "testing"

func TestRunDemoSuccess(t *testing.T) {
	res, err := RunDemo()
	if err != nil {
		t.Fatalf("RunDemo error: %v", err)
	}
	if res.DelegationID == "" || !res.InvalidAction || !res.PostRevokeFail {
		t.Fatalf("unexpected demo result: %+v", res)
	}
}
