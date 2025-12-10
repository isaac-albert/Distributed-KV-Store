package shutdown

import (
	"fmt"
	"log"
	"os"

	"www.github.com/isaac-albert/Distributed-KV-Store/internal/raft"
)

func ShutDown() error {

	for _, n := range raft.ClusterNodes.Nodes {
		future := n.Shutdown()
		if err := future.Error(); err != nil {
			return err
		}
	}

	for _, n := range raft.ClusterNodes.Nodes {
		if n.Stores.StableStore != nil {
			if err := n.Stores.StableStore.Close(); err != nil {
				log.Printf("error closing stable store: %v", err)
			}
		}
		if n.Stores.LogStore != nil {
			if err := n.Stores.LogStore.Close(); err != nil {
				log.Printf("error closing log store: %v", err)
			}
		}
	}

	err := os.RemoveAll("assets")
	if err != nil {
		return fmt.Errorf("shutdown path %s: %w", "assets", err)
	}

	raft.ClusterNodes.Nodes = nil

	return nil
}
