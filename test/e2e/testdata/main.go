package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Start a HTTP server that listens 8080 and responds "Hello, world!" to all requests.
// It exists gracefully on SIGINT and SIGTERM.
func runHTTPServer() {
	// Create a HTTP server.
	srv := &http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello, world!"))
		}),
	}

	// Create a channel to receive signals.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start the server in a goroutine.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("HTTP server ListenAndServe: %v", err)
		}
	}()

	// Wait for a signal.
	<-sigCh

	// Create a context with 5 second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ask the server to gracefully shutdown.
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server Shutdown: %v", err)
	}
}

func main() {
	runHTTPServer()
}
