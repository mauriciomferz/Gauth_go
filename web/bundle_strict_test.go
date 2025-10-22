package web

import (
	"os"
	"strings"
	"testing"
)

// TestStrictAssetsPlaceholder ensures that in prod with GAUTH_STRICT_ASSETS=1 and no manifest,
// the placeholder remains and a STRICT_ASSET_FAILURE marker is appended for detection.
func TestStrictAssetsPlaceholder(t *testing.T) {
	bs := &BetaServer{}
	origEnv := os.Getenv("GAUTH_ENV")
	origStrict := os.Getenv("GAUTH_STRICT_ASSETS")
	defer os.Setenv("GAUTH_ENV", origEnv)
	defer os.Setenv("GAUTH_STRICT_ASSETS", origStrict)
	os.Setenv("GAUTH_ENV", "prod")
	os.Setenv("GAUTH_STRICT_ASSETS", "1")
	page := []byte(`<script src="/static/js/__APP_BUNDLE__"></script>`)
	out := string(bs.applyBundleSubstitution(page))
	if !strings.Contains(out, "__APP_BUNDLE__") {
		t.Fatalf("expected unresolved placeholder under strict mode, got %s", out)
	}
	if !strings.Contains(out, "STRICT_ASSET_FAILURE") {
		t.Fatalf("expected STRICT_ASSET_FAILURE marker in output: %s", out)
	}
}
