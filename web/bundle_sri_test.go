package web

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyBundleSubstitutionSRI(t *testing.T) {
	bs := &BetaServer{}
	origEnv := os.Getenv("AGENTAUTH_ENV")
	defer t.Setenv("AGENTAUTH_ENV", origEnv)
	t.Setenv("AGENTAUTH_ENV", "prod")
	manifestDir := filepath.Join("web", "static", "js")
	// #nosec G301
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifestPath := filepath.Join(manifestDir, "asset-manifest.json")
	manifest := `{"app":"app-deadbeef.js","sha256":"deadbeefdeadbeef","sri":"sha256-ABCDEFG=="}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	defer func() { _ = os.Remove(manifestPath) }()

	page := []byte(`<script src="/static/js/__APP_BUNDLE__" integrity="__APP_SRI__"></script>`)
	out := string(bs.applyBundleSubstitution(page))
	if !strings.Contains(out, "app-deadbeef.js") {
		t.Fatalf("expected app filename substituted: %s", out)
	}
	if !strings.Contains(out, "sha256-ABCDEFG==") {
		t.Fatalf("expected SRI substituted: %s", out)
	}
}
