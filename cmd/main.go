package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"www.github.com/isaac-albert/Distributed-KV-Store/cmd/flag"
	"www.github.com/isaac-albert/Distributed-KV-Store/internal/raft"
)

func StartProgram() error {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	flagConfig, err := flag.StartFlags()
	if err != nil {
		return err
	}

	server, err := raft.NewServer(flagConfig.NodeID, flagConfig.HttpAddr, flagConfig.RaftAddr)
	if err != nil {
		return fmt.Errorf("error starting a server: %w", err)
	}

	httpErrChan := make(chan error, 1)

	go func() {
		err := server.StartHTTPServer()
		if err != nil && err != http.ErrServerClosed {
			httpErrChan <- err
		}
	}()

	select {
	case <-httpErrChan:
		return fmt.Errorf("http server failed to start: %w", err)
	case <-time.After(100 * time.Millisecond):
		break
	}

	err = server.AddNodeToNetwork(flagConfig.JoinAddr)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Node running, press CTRL+C to stop server")

	select {
	case <-ctx.Done():
		log.Println("Shutting Down Server gracefully...")
	case <-httpErrChan:
		return fmt.Errorf("http server crashed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.ShutdownHTTP(ctx); err != nil {
		return fmt.Errorf("http shutdown error: %w", err)
	}

	if err := server.ShutdownRaft(); err != nil {
		return fmt.Errorf("raft shutdown error: %w", err)
	}

	if err := server.ShutdownStores(); err != nil {
		return fmt.Errorf("store shutdown error: %w", err)
	}

	log.Println("Server exited successfully")
	return nil
}
