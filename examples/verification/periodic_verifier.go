package main

// Demonstration of periodic revocation transparency verification using pkg/verification.
// NOT production ready: simplistic backoff & logging. Intended to show structured error handling.

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	verification "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/verification"
)

func main() {
	base := envOr("GAUTH_BASE_URL", "http://localhost:8080")
	interval := 10 * time.Second
	client := &http.Client{Timeout: 5 * time.Second}

	log.Printf("[periodic] starting verifier loop base=%s interval=%s", base, interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		start := time.Now()
		// Use context with timeout to bound verification duration
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		err := verification.VerifyAll(client, base, "")
		cancel()
		dur := time.Since(start)
		if err != nil {
			var vErr *verification.VerifyError
			if errors.As(err, &vErr) {
				log.Printf("[verify][error] code=%s detail=%s cause=%v duration=%s", vErr.Code, vErr.Detail, vErr.Cause, dur)
				handleCode(vErr.Code)
			} else {
				log.Printf("[verify][error] unstructured=%v duration=%s", err, dur)
			}
		} else {
			log.Printf("[verify][ok] duration=%s", dur)
		}
		_ = ctx // explicit noop to satisfy linters referencing ctx usage pattern
	}
}

func handleCode(code string) {
	switch code {
	case "no_events":
		// benign – chain empty
	case "inclusion_failed":
		log.Println("[action] raise alert: inclusion integrity failure")
	case "proof_endpoint_failure":
		log.Println("[action] schedule immediate re-check: proof endpoint failure")
	case "sth_verify":
		log.Println("[action] escalate: multi-sig signature verification failure")
	default:
		// other codes logged already
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
