package smoketest

import (
	"net/http"
	"os"
	"testing"
	"time"

	webpkg "github.com/mauriciomferz/AgentAuth/web"
)

// TestMain boots a beta server on :8080 so smoketests are self-contained and
// do not rely on an externally launched process in CI/local runs.
func TestMain(m *testing.M) {
	go func() {
		// Start server; ignore error on shutdown.
		bs := webpkg.NewBetaServer(":8080")
		bs.RegisterUIRoutes()
		//nolint:gosec // G114: test server, timeout not critical
		_ = http.ListenAndServe(":8080", bs.Engine())
	}()
	// Allow minimal time for server to bind.
	// Allow adequate time for full route registration (large initialization).
	time.Sleep(1 * time.Second)
	os.Exit(m.Run())
}
