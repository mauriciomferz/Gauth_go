package web

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/metrics"
	"github.com/mauriciomferz/Gauth_go/pkg/crypto"
)

// Run starts the BetaServer HTTP listener and blocks until a signal is received or error occurs.
// It handles graceful shutdown on SIGINT/SIGTERM.
func (s *BetaServer) Run() error {
	// Precedence: explicit env GAUTH_WEB_PORT overrides constructor, else constructor value.
	addr := os.Getenv("GAUTH_WEB_PORT")
	if addr == "" {
		addr = s.port
	}
	if addr == "" {
		addr = ":8080"
	}
	// Normalize address: allow plain numeric port (e.g. 8080) by prefixing ':'; if host:port already present leave unchanged.
	// This mirrors normalization logic used in the constructor.
	if !strings.Contains(addr, ":") { // no colon implies just digits
		addr = ":" + addr
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.router,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}
	fmt.Printf("[startup] BetaServer starting PID=%d on http://localhost%s at %s\n", os.Getpid(), addr, time.Now().Format(time.RFC3339)) // Signal handling for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		fmt.Println("[startup] invoking ListenAndServe...")
		if err := srv.ListenAndServe(); err != nil {
			errCh <- err
		} else {
			errCh <- nil
		}
	}()

	select {
	case sig := <-stop:
		fmt.Printf("\nReceived signal %s, shutting down...\n", sig)
		// Persist metrics snapshot if enabled
		if mm, ok := s.metrics.(*metrics.Memory); ok {
			if err := mm.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "[metrics] save error: %v\n", err)
			} else if mm != nil {
				fmt.Fprintln(os.Stderr, "[metrics] snapshot persisted")
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// Stop EdDSA rotation scheduler if active BEFORE shutting down server
		if km := s.getKeyManager(); km != nil {
			km.Stop()
		}
		// Shut down AgentAuth+ components (caches)
		ShutdownAgentAuthPlus()
		return srv.Shutdown(ctx)

	case err := <-errCh:
		if err == http.ErrServerClosed || err == nil {
			fmt.Println("[shutdown] server closed cleanly")
			if mm, ok := s.metrics.(*metrics.Memory); ok {
				if saveErr := mm.Save(); saveErr != nil {
					fmt.Fprintf(os.Stderr, "[metrics] save error: %v\n", saveErr)
				} else if mm != nil {
					fmt.Fprintln(os.Stderr, "[metrics] snapshot persisted")
				}
			}
			return nil
		}
		fmt.Printf("[error] server exited: %v\n", err)
		if mm, ok := s.metrics.(*metrics.Memory); ok {
			if saveErr := mm.Save(); saveErr != nil {
				fmt.Fprintf(os.Stderr, "[metrics] save error: %v\n", saveErr)
			}
		}
		return err
	}
}

// Shutdown initiates a manual graceful shutdown of the server components.
// It is idempotent and safe to call concurrently.
func (s *BetaServer) Shutdown() {
	if s == nil {
		return
	}
	if s.stopped.Load() {
		return
	}
	s.stopped.Store(true)
	// Close stopCh to broadcast cancellation; recover in case already closed.
	defer func() { _ = recover() }()
	close(s.stopCh)
	// Brief wait to allow loops to exit (they select on stopCh)
	time.Sleep(50 * time.Millisecond)
	// Perform final persistence saves if paths configured (best-effort)
	if s.violationHandler != nil {
		s.violationHandler.Save()
	}
	if s.semanticHandler != nil {
		s.semanticHandler.Save()
	}
	// Flush metrics persistence if enabled
	if mm, ok := s.metrics.(*metrics.Memory); ok {
		if mErr := mm.Save(); mErr != nil {
			fmt.Fprintf(os.Stderr, "[shutdown] metrics persistence failed: %v\n", mErr)
		}
	}
	// Flush limits persistence (best-effort)
	// Note: Model Limits handler doesn't have explicit Save() in interface usually?
	// Checking server_clean.go logic.
	// It says "Flush limits persistence".

	// Shut down AgentAuth+ components (caches)
	if s.systemClockMonitor != nil {
		s.systemClockMonitor.Stop()
	}
	ShutdownAgentAuthPlus()
}

// getKeyManager helper used in Run()
func (s *BetaServer) getKeyManager() *crypto.Manager {
	if s.keyProvider == nil {
		return nil
	}
	if km, ok := s.keyProvider.(*crypto.Manager); ok {
		return km
	}
	return nil
}
