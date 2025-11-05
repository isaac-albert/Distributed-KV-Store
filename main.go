package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api", Hello)

	server := &http.Server{
		Addr:    "localhost:3000",
		Handler: mux,
	}

	closeIdleConnections := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		fmt.Printf("\ngracefully shutting down %v...\n", server.Addr)
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			fmt.Printf("HTTP server shutdown: %v\n", err)
		}
		close(closeIdleConnections)
	}()

	fmt.Println(">> listening on addr", server.Addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("error listen and serve %v", err)
	}

	<-closeIdleConnections
}

func Hello(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		fmt.Fprintf(w, "hello from handler - %s\n", "get method used")
		return
	}
	fmt.Fprintf(w, "hello from handler - %s\n", "other method used")
}
