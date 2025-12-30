package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStrictMalformedManifest ensures strict mode flags invalid JSON manifest.
func TestStrictMalformedManifest(t *testing.T) {
	bs := &BetaServer{}
	origEnv := os.Getenv("GAUTH_ENV")
	origStrict := os.Getenv("GAUTH_STRICT_ASSETS")
	defer t.Setenv("GAUTH_ENV", origEnv)
	defer t.Setenv("GAUTH_STRICT_ASSETS", origStrict)
	t.Setenv("GAUTH_ENV", "prod")
	t.Setenv("GAUTH_STRICT_ASSETS", "1")
	manifestDir := filepath.Join("web", "static", "js")
	// #nosec G301
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifestPath := filepath.Join(manifestDir, "asset-manifest.json")
	// Invalid JSON content
	if err := os.WriteFile(manifestPath, []byte(`{"app":"file.js",`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	defer func() { _ = os.Remove(manifestPath) }()
	page := []byte(`<script src="/static/js/__APP_BUNDLE__" integrity="__APP_SRI__"></script>`)
	out := string(bs.applyBundleSubstitution(page))
	if !strings.Contains(out, "STRICT_ASSET_FAILURE: invalid manifest JSON") {
		t.Fatalf("expected invalid manifest failure marker, got: %s", out)
	}
	// Placeholders should remain unchanged under failure
	if !strings.Contains(out, "__APP_BUNDLE__") || !strings.Contains(out, "__APP_SRI__") {
		t.Fatalf("expected placeholders retained under failure: %s", out)
	}
}

// TestStrictMissingFieldsManifest ensures strict mode flags missing required fields.
func TestStrictMissingFieldsManifest(t *testing.T) {
	bs := &BetaServer{}
	origEnv := os.Getenv("GAUTH_ENV")
	origStrict := os.Getenv("GAUTH_STRICT_ASSETS")
	defer t.Setenv("GAUTH_ENV", origEnv)
	defer t.Setenv("GAUTH_STRICT_ASSETS", origStrict)
	t.Setenv("GAUTH_ENV", "prod")
	t.Setenv("GAUTH_STRICT_ASSETS", "1")
	manifestDir := filepath.Join("web", "static", "js")
	// #nosec G301
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifestPath := filepath.Join(manifestDir, "asset-manifest.json")
	// Missing SRI field
	if err := os.WriteFile(manifestPath, []byte(`{"app":"file.js"}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	defer func() { _ = os.Remove(manifestPath) }()
	page := []byte(`<script src="/static/js/__APP_BUNDLE__" integrity="__APP_SRI__"></script>`)
	out := string(bs.applyBundleSubstitution(page))
	if !strings.Contains(out, "STRICT_ASSET_FAILURE: missing required fields") {
		t.Fatalf("expected missing required fields failure marker, got: %s", out)
	}
	// Placeholders should remain unchanged under failure
	if !strings.Contains(out, "__APP_BUNDLE__") || !strings.Contains(out, "__APP_SRI__") {
		t.Fatalf("expected placeholders retained under failure: %s", out)
	}
}
