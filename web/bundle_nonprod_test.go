package web

import (
	"os"
	"strings"
	"testing"
)

// TestNonProdNoSubstitution ensures placeholders are untouched outside prod environment.
func TestNonProdNoSubstitution(t *testing.T) {
	bs := &BetaServer{}
	orig := os.Getenv("GAUTH_ENV")
	defer os.Setenv("GAUTH_ENV", orig)
	os.Setenv("GAUTH_ENV", "dev")
	page := []byte(`<script src="/static/js/__APP_BUNDLE__" integrity="__APP_SRI__"></script>`)
	out := string(bs.applyBundleSubstitution(page))
	if !strings.Contains(out, "__APP_BUNDLE__") || !strings.Contains(out, "__APP_SRI__") {
		t.Fatalf("expected placeholders unchanged in non-prod: %s", out)
	}
	if strings.Contains(out, "STRICT_ASSET_FAILURE") {
		t.Fatalf("did not expect strict failure marker in non-prod: %s", out)
	}
}
