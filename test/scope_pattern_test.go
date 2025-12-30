package test

import (
	"os"
	"testing"

	gauth_aap_001 "github.com/mauriciomferz/AgentAuth/pkg/gauth_aap_001"
)

func TestScopePatternsExactAndWildcard(t *testing.T) {
	scope := []string{"read.user", "*"}
	if !gauth_aap_001.ScopeContains(scope, "any.action") {
		// Global wildcard should match everything
		t.Fatalf("expected global wildcard to match arbitrary action")
	}
	if !gauth_aap_001.ScopeContains(scope, "read.user") {
		t.Fatalf("expected exact match for read.user")
	}
	if gauth_aap_001.ScopeContains([]string{"read.user"}, "write.user") {
		t.Fatalf("did not expect read.user scope to match write.user")
	}
}

func TestScopePatternsPrefixWildcard(t *testing.T) {
	scope := []string{"audit.*"}
	if !gauth_aap_001.ScopeContains(scope, "audit.read.log") {
		t.Fatalf("expected prefix wildcard audit.* to match audit.read.log")
	}
	if gauth_aap_001.ScopeContains(scope, "auditz.read.log") { // prefix mismatch
		t.Fatalf("did not expect audit.* to match auditz.read.log")
	}
}

func TestScopePatternsRegexGate(t *testing.T) {
	scope := []string{"regex:/^payment\\.(create|update)$/"}
	// Regex disabled by default
	if gauth_aap_001.ScopeContains(scope, "payment.create") {
		t.Fatalf("regex matched while gate disabled")
	}
	os.Setenv("GAUTH_SCOPE_ALLOW_REGEX", "1")
	defer os.Unsetenv("GAUTH_SCOPE_ALLOW_REGEX")
	if !gauth_aap_001.ScopeContains(scope, "payment.create") {
		t.Fatalf("expected regex to match payment.create when enabled")
	}
	if gauth_aap_001.ScopeContains(scope, "payment.delete") {
		t.Fatalf("did not expect regex to match payment.delete")
	}
}

func TestScopePatternsNumericRange(t *testing.T) {
	scope := []string{"rate[5-10]"}
	if !gauth_aap_001.ScopeContains(scope, "rate7") {
		t.Fatalf("expected rate7 within range [5-10]")
	}
	if gauth_aap_001.ScopeContains(scope, "rate4") {
		t.Fatalf("did not expect rate4 below range [5-10]")
	}
	if gauth_aap_001.ScopeContains(scope, "rate11") {
		t.Fatalf("did not expect rate11 above range [5-10]")
	}
}
