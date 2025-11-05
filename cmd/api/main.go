package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"www.github.com/isaac-albert/Distributed-KV-Store/cmd/handlers"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api", handlers.MethodRouter)

	server := &http.Server{
		Addr:    "localhost:3000",
		Handler: mux,
	}

	closeIdleConnections := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		fmt.Printf("\n>> gracefully shutting down %v...\n", server.Addr)
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			fmt.Printf(">> HTTP server shutdown: %v\n", err)
		}
		close(closeIdleConnections)
	}()

	fmt.Println(">> listening on addr", server.Addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf(">> error listen and serve %v", err)
	}

	<-closeIdleConnections
}
