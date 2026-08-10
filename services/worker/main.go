package main

import (
	"net/http"

	"charm.land/log/v2"
)

func main() {
	mux := http.NewServeMux()
	log.Info("Starting server", "port", 8000)
	if err := http.ListenAndServe(":8000", mux); err != nil {
		log.Fatal("failed to start the server!")
	}
}
