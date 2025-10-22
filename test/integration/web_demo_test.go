//go:build integration
// +build integration

package integration

import (
	"net/http"
	"os/exec"
	"testing"
	"time"
)

func startWebDemo(t *testing.T) func() {
	cmd := exec.Command("bash", "scripts/start-web-demo.sh")
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start web demo: %v", err)
	}
	// Wait for server to start
	time.Sleep(2 * time.Second)
	return func() {
		exec.Command("bash", "scripts/stop-web-demo.sh").Run()
	}
}

func TestWebDemoHealth(t *testing.T) {
	stop := startWebDemo(t)
	defer stop()
	resp, err := http.Get("http://localhost:8080/api/v1/beta/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
	}
}
