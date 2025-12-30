package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // #nosec G108 // Enable pprof HTTP handlers, gated by GAUTH_ENABLE_PPROF
	"os"
	"strings"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/config"
	"github.com/mauriciomferz/AgentAuth/internal/security"
	"github.com/mauriciomferz/AgentAuth/web"
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

	// Enable pprof profiling endpoints if GAUTH_ENABLE_PPROF=1
	if os.Getenv("GAUTH_ENABLE_PPROF") == "1" {
		pprofPort := os.Getenv("GAUTH_PPROF_PORT")
		if pprofPort == "" {
			pprofPort = "6060"
		}
		go func() {
			pprofAddr := ":" + pprofPort
			log.Printf("[pprof] Starting profiling server on http://localhost%s/debug/pprof/\n", pprofAddr)
			log.Printf("[pprof] Available endpoints:\n")
			log.Printf("[pprof]   - CPU profile: http://localhost%s/debug/pprof/profile?seconds=30\n", pprofAddr)
			log.Printf("[pprof]   - Heap profile: http://localhost%s/debug/pprof/heap\n", pprofAddr)
			log.Printf("[pprof]   - Goroutines: http://localhost%s/debug/pprof/goroutine\n", pprofAddr)
			log.Printf("[pprof]   - All profiles: http://localhost%s/debug/pprof/\n", pprofAddr)
			server := &http.Server{
				Addr:         pprofAddr,
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
			}
			if err := server.ListenAndServe(); err != nil {
				log.Printf("[pprof] Server failed: %v\n", err)
			}
		}()
	}

	// SECURITY: Validate critical configuration before starting server
	productionMode := security.ProductionModeDetector()
	if productionMode {
		log.Println("[SECURITY] Production mode detected - enforcing security validations")
	} else {
		log.Println("[SECURITY] Development mode detected - reduced security requirements")
	}

	validator := security.NewStartupValidator(productionMode)
	if err := validator.ValidateAll(); err != nil {
		log.Fatalf("[SECURITY] FATAL: %v\n\nSERVER STARTUP BLOCKED. Fix the above security issues and restart.\n", err)
	}

	log.Println("[SECURITY] All security validations passed ✓")

	// Prefer new beta constructor (legacy NewEducationalServer retained as alias)
	srv := web.NewBetaServer(port)
	_ = srv.Run()
}
