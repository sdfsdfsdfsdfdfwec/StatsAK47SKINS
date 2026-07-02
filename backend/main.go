package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	router := SetupRouter(nil)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	go StartBackgroundScraper()

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Server starting on port %s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
