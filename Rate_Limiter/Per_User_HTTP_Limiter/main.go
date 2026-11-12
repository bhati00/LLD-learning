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

/*
APPROACH
1. create the server
2. use the token bucker or create the one
3. Create the middleware and use that tokenbucket
4. use the middlewares in the handlers
*/
/*
SERVER GRACEFUL SHUTDOWN :
1. create the multiplexer & registers our handlers
2. create the http struct
2. create the channel of OS type and that starts listening the SIGTERM
4. create the context backgroud with the grace period  let the server to pass it
5. listern for the graceful shutdown
*/
func main() {
	limiter := NewRateLimiter()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.Write([]byte("hello " + r.Header.Get("User-ID") + "\n"))
	})
	server := &http.Server{
		Addr:    ":8080",
		Handler: limiter.RateLimit(mux),
	}
	serverErr := make(chan error, 1)
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		log.Fatalf("server failed to start or crashed: %v", err)
	case sig := <-quit:
		log.Printf("received signal %v, shutting down", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v, forcing close", err)
		server.Close()
	}
	log.Println("server stopped")
}
