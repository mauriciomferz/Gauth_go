package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	version := os.Getenv("VERSION")
	if version == "" {
		version = "unknown"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "GAuth %s environment - OK\n", version)
	})

	http.HandleFunc("/api/v1/beta/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","version":"%s"}`, version)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting GAuth %s on port %s\n", version, port)
	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Server failed: %v\n", err)
		os.Exit(1)
	}
}
