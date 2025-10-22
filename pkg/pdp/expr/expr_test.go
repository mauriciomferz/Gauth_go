package expr

import (
	"testing"
	"time"
)

func TestEvalBasic(t *testing.T) {
	attrs := map[string]string{"role": "finance", "amount": "250"}
	ok, err := Eval("role == 'finance' && amount < 300", attrs, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected expression to evaluate true")
	}
}

func TestEvalTimeBetween(t *testing.T) {
	attrs := map[string]string{}
	cur := time.Date(2025, 10, 17, 15, 30, 0, 0, time.UTC)
	ok, err := Eval("time_between(\"09:00\",\"17:00\")", attrs, cur)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected inside time window")
	}
}

func TestEvalInOperator(t *testing.T) {
	attrs := map[string]string{"dept": "risk"}
	ok, err := Eval("dept in ['risk','finance']", attrs, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected membership match")
	}
}

func TestEvalNumericFailure(t *testing.T) {
	attrs := map[string]string{"amount": "notnum"}
	_, err := Eval("amount > 10", attrs, time.Now())
	if err == nil {
		t.Fatalf("expected numeric parse error")
	}
}

func TestEvalParenAndNot(t *testing.T) {
	attrs := map[string]string{"env": "prod", "flag": "on"}
	ok, err := Eval("!(env == 'dev') && flag == 'on'", attrs, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected true after NOT dev")
	}
}

func TestEvalContains(t *testing.T) {
	attrs := map[string]string{"msg": "payment approved"}
	ok, err := Eval("contains('msg','approved')", attrs, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected contains to match")
	}
}

func TestEvalRegexMatch(t *testing.T) {
	attrs := map[string]string{"email": "user@example.com"}
	ok, err := Eval("regex_match('email','^[a-z]+@example\\.com$')", attrs, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected regex to match")
	}
}
