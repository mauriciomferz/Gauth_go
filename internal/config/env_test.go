package config

import (
	"os"
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	_ = os.Unsetenv("CFG_TEST_A")
	if v := Get("CFG_TEST_A", "default"); v != "default" {
		t.Fatalf("expected default got %s", v)
	}
	_ = os.Setenv("CFG_TEST_A", "value")
	if v := Get("CFG_TEST_A", "default"); v != "value" {
		t.Fatalf("expected value got %s", v)
	}
}

func TestGetInt(t *testing.T) {
	_ = os.Unsetenv("CFG_INT")
	if v := GetInt("CFG_INT", 42); v != 42 {
		t.Fatalf("expected default 42 got %d", v)
	}
	_ = os.Setenv("CFG_INT", "notnum")
	if v := GetInt("CFG_INT", 42); v != 42 {
		t.Fatalf("invalid parse should fallback 42 got %d", v)
	}
	_ = os.Setenv("CFG_INT", "-1")
	if v := GetInt("CFG_INT", 42); v != 42 {
		t.Fatalf("negative should fallback 42 got %d", v)
	}
	os.Setenv("CFG_INT", "10")
	if v := GetInt("CFG_INT", 42); v != 10 {
		t.Fatalf("expected 10 got %d", v)
	}
}

func TestEphemeralSecretProvided(t *testing.T) {
	os.Setenv("CFG_SECRET", "fixed")
	sec, gen, warn := EphemeralSecret("CFG_SECRET", 16)
	if sec != "fixed" || gen || warn != "" {
		t.Fatalf("expected fixed secret, no generation, got %s gen=%v warn=%s", sec, gen, warn)
	}
}

func TestEphemeralSecretGenerated(t *testing.T) {
	_ = os.Unsetenv("CFG_SECRET")
	sec, gen, warn := EphemeralSecret("CFG_SECRET", 16)
	if !gen {
		t.Fatalf("expected generation")
	}
	if sec == "" {
		t.Fatalf("expected non-empty secret")
	}
	if !strings.Contains(warn, "CFG_SECRET not set") {
		t.Fatalf("unexpected warn: %s", warn)
	}
	// Second call should generate a different secret (high probability)
	sec2, gen2, _ := EphemeralSecret("CFG_SECRET_2", 16)
	if sec2 == sec {
		t.Logf("warning: two generated secrets identical; extremely unlikely but not impossible")
	}
	if !gen2 {
		t.Fatalf("expected generation on second key")
	}
}
