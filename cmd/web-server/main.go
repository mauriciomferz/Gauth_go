// The web-server command launches the GAuth beta HTTP API including metrics, policy,
// delegation and token endpoints. It is a demonstration binary and NOT production hardened.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/config"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web"
)

// The web-server binary provides the long-running HTTP API for the beta demonstration.
// A lightweight "-healthcheck" flag is supported for container health probes.
func main() {
	healthcheck := flag.Bool("healthcheck", false, "Perform a lightweight health probe against the beta health endpoint and exit")
	flag.Parse()

	if *healthcheck {
		url := config.Get("GAUTH_HEALTH_URL", "http://localhost:8080/api/v1/beta/health")
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			fmt.Println("healthcheck request failed:", err)
			os.Exit(1)
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			fmt.Println("OK")
			_ = resp.Body.Close()
			os.Exit(0)
		}
		_ = resp.Body.Close()
		fmt.Printf("unhealthy status: %d\n", resp.StatusCode)
		os.Exit(1)
	}

	port := os.Getenv("GAUTH_WEB_PORT")
	if port == "" {
		port = "8080"
	}
	if len(flag.Args()) > 0 { // after flag parsing
		port = flag.Args()[0]
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	// Prefer new beta constructor (legacy NewEducationalServer retained as alias)
	srv := web.NewBetaServer(port)
	_ = srv.Run()
}
