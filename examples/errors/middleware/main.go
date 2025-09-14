package main

import (
	"log"
	"net/http"

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

	errHandler := &mw.ErrorHandler{Next: mux}

	log.Println("Starting error middleware demo server on :8080")
	http.ListenAndServe(":8080", errHandler)
}
