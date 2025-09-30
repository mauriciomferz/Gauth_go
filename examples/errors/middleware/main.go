package main

import (
	"log"
	"net/http"
	"time"

	mw "github.com/Gimel-Foundation/gauth/examples/errors/middleware/internal"
	gautherr "github.com/Gimel-Foundation/gauth/pkg/errors"
)

// demoHandler is a sample handler that always returns an error
func demoHandler(w http.ResponseWriter, r *http.Request) {
	err := gautherr.New(gautherr.ErrUnauthorizedClient, "demo: unauthorized access")
	mw.ErrorResponse(w, r, err)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/demo", demoHandler)
	mux.HandleFunc("/audit", mw.AuditHandler)

	errHandler := &mw.ErrorHandler{Next: mux}

	server := &http.Server{
		Addr:         ":8080",
		Handler:      errHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("Starting error middleware demo server on :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
