package authz

import (
	"context"
	"testing"
	"time"
)

func TestPolicyRoleMatching(t *testing.T) {
	ma := NewMemoryAuthorizer()
	// define policy using role instead of direct subject
	ma.AddPolicy(Policy{ID: "p1", Roles: []string{"admin"}, Resource: "vault", Actions: []string{"read"}, Effect: Allow})
	// simulate request with roles in context
	req := Request{Subject: "alice", Resource: "vault", Action: "read", Context: map[string]string{"roles": "admin"}}
	dec, err := ma.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("authorize err: %v", err)
	}
	if !dec.Allow {
		t.Fatalf("expected allow via role policy, got deny: %s", dec.Reason)
	}
}

func TestPolicyRequiredScopes(t *testing.T) {
	ma := NewMemoryAuthorizer()
	ma.AddPolicy(Policy{ID: "p2", Subject: "bob", Resource: "data", Actions: []string{"write"}, Effect: Allow, RequiredScopes: []string{"write:data", "admin:users"}})
	// missing one scope
	reqDenied := Request{Subject: "bob", Resource: "data", Action: "write", Context: map[string]string{"scopes": "write:data"}}
	dec, _ := ma.Authorize(context.Background(), reqDenied)
	if dec.Allow {
		t.Fatalf("expected deny due to missing admin:users scope")
	}
	// now include all required scopes
	reqAllowed := Request{Subject: "bob", Resource: "data", Action: "write", Context: map[string]string{"scopes": "write:data admin:users"}}
	dec2, _ := ma.Authorize(context.Background(), reqAllowed)
	if !dec2.Allow {
		t.Fatalf("expected allow when all required scopes present, reason=%s", dec2.Reason)
	}
}

func TestABACAdvancedOperators(t *testing.T) {
	ma := NewMemoryAuthorizer()
	// equals
	ma.AddPolicy(Policy{ID: "p-eq", Subject: "alice", Resource: "doc", Actions: []string{"view"}, Effect: Allow, Conditions: []Condition{{Key: "dept", Operator: "equals", Values: []string{"eng"}}}})
	// not_equals
	ma.AddPolicy(Policy{ID: "p-ne", Subject: "alice", Resource: "doc2", Actions: []string{"view"}, Effect: Allow, Conditions: []Condition{{Key: "dept", Operator: "not_equals", Values: []string{"sales"}}}})
	// contains
	ma.AddPolicy(Policy{ID: "p-contains", Subject: "alice", Resource: "doc3", Actions: []string{"view"}, Effect: Allow, Conditions: []Condition{{Key: "tags", Operator: "contains", Values: []string{"urgent"}}}})
	// prefix
	ma.AddPolicy(Policy{ID: "p-prefix", Subject: "alice", Resource: "doc4", Actions: []string{"view"}, Effect: Allow, Conditions: []Condition{{Key: "path", Operator: "prefix", Values: []string{"/secure"}}}})
	// suffix
	ma.AddPolicy(Policy{ID: "p-suffix", Subject: "alice", Resource: "doc5", Actions: []string{"view"}, Effect: Allow, Conditions: []Condition{{Key: "file", Operator: "suffix", Values: []string{".pdf"}}}})
	// in (alias of equals list membership) already covered by equals logic but test explicit
	ma.AddPolicy(Policy{ID: "p-in", Subject: "alice", Resource: "doc6", Actions: []string{"view"}, Effect: Allow, Conditions: []Condition{{Key: "env", Operator: "in", Values: []string{"prod", "stage"}}}})

	tests := []struct {
		res    string
		ctx    map[string]string
		expect bool
	}{
		{"doc", map[string]string{"dept": "eng"}, true},
		{"doc", map[string]string{"dept": "sales"}, false},
		{"doc2", map[string]string{"dept": "eng"}, true},
		{"doc2", map[string]string{"dept": "sales"}, false},
		{"doc3", map[string]string{"tags": "important,urgent,team"}, true},
		{"doc3", map[string]string{"tags": "important"}, false},
		{"doc4", map[string]string{"path": "/secure/alpha"}, true},
		{"doc4", map[string]string{"path": "/public/secure"}, false},
		{"doc5", map[string]string{"file": "report.final.pdf"}, true},
		{"doc5", map[string]string{"file": "report.txt"}, false},
		{"doc6", map[string]string{"env": "stage"}, true},
		{"doc6", map[string]string{"env": "dev"}, false},
	}
	for i, tc := range tests {
		dec, _ := ma.Authorize(context.Background(), Request{Subject: "alice", Resource: tc.res, Action: "view", Context: tc.ctx})
		if dec.Allow != tc.expect {
			t.Fatalf("case %d expected %v got %v reason=%s", i, tc.expect, dec.Allow, dec.Reason)
		}
	}
}

func TestExtendedConditionOperators(t *testing.T) {
	ma := NewMemoryAuthorizer()
	// regex match on email domain
	ma.AddPolicy(Policy{
		ID:       "regex-email",
		Subject:  "alice",
		Resource: "svc",
		Actions:  []string{"access"},
		Effect:   Allow,
		Conditions: []Condition{{
			Key:      "email",
			Operator: "regex",
			Values:   []string{"^alice@.*\\.com$"},
		}},
	})
	// numeric greater than (age > 21)
	ma.AddPolicy(Policy{
		ID:       "age-gt",
		Subject:  "bob",
		Resource: "svc",
		Actions:  []string{"drink"},
		Effect:   Allow,
		Conditions: []Condition{{
			Key:      "age",
			Operator: "numeric_gt",
			Values:   []string{"21"},
		}},
	})
	// numeric less than (risk score < 0.3)
	ma.AddPolicy(Policy{
		ID:       "risk-lt",
		Subject:  "carol",
		Resource: "svc",
		Actions:  []string{"trade"},
		Effect:   Allow,
		Conditions: []Condition{{
			Key:      "risk",
			Operator: "numeric_lt",
			Values:   []string{"0.30"},
		}},
	})
	// time_before (current time string earlier than deadline)
	deadline := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
	ma.AddPolicy(Policy{
		ID:       "before-deadline",
		Subject:  "dan",
		Resource: "svc",
		Actions:  []string{"submit"},
		Effect:   Allow,
		Conditions: []Condition{{
			Key:      "now",
			Operator: "time_before",
			Values:   []string{deadline},
		}},
	})
	// time_after
	past := time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	ma.AddPolicy(Policy{
		ID:       "after-past",
		Subject:  "erin",
		Resource: "svc",
		Actions:  []string{"archive"},
		Effect:   Allow,
		Conditions: []Condition{{
			Key:      "now",
			Operator: "time_after",
			Values:   []string{past},
		}},
	})

	tests := []struct {
		subject string
		action  string
		ctx     map[string]string
		expect  bool
	}{
		{"alice", "access", map[string]string{"email": "alice@example.com"}, true},
		{"alice", "access", map[string]string{"email": "alice@bad"}, false},
		{"bob", "drink", map[string]string{"age": "25"}, true},
		{"bob", "drink", map[string]string{"age": "20"}, false},
		{"carol", "trade", map[string]string{"risk": "0.10"}, true},
		{"carol", "trade", map[string]string{"risk": "0.50"}, false},
		{"dan", "submit", map[string]string{"now": time.Now().UTC().Format(time.RFC3339)}, true},
		{"erin", "archive", map[string]string{"now": time.Now().UTC().Format(time.RFC3339)}, true},
	}

	for i, tt := range tests {
		dec, _ := ma.Authorize(context.Background(), Request{Subject: tt.subject, Resource: "svc", Action: tt.action, Context: tt.ctx})
		if dec.Allow != tt.expect {
			t.Fatalf("test %d: expected %v got %v reason=%s", i, tt.expect, dec.Allow, dec.Reason)
		}
	}
}
