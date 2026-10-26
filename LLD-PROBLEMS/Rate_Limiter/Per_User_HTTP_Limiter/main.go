package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	limiter := NewLimiter()
	mux := http.NewServeMux()
	helloHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hey i was called :)")
	})

	mux.Handle("/hello", limiter.Middleware(helloHandler))

	// configure the server
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// singals and error
	stopCh := make(chan os.Signal, 1)
	errorCh := make(chan error, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stopCh)

	// running server
	go func() {
		log.Print("Starting HTTP server at http://localhost:8080; press Ctrl+C to shut down")
		errorCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errorCh:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server failed: %v", err)
		}
		return
	case <-stopCh:
		log.Print("Shutdown requested; waiting for active requests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
		if closeErr := server.Close(); closeErr != nil {
			log.Printf("Unable to close HTTP server: %v", closeErr)
		}
	} else {
		log.Print("Graceful shutdown complete")
	}

	if err := <-errorCh; !errors.Is(err, http.ErrServerClosed) {
		log.Printf("HTTP server failed: %v", err)
	}
}
