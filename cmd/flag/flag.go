package flag

import (
	"errors"
	"flag"
	"fmt"

	"www.github.com/isaac-albert/Distributed-KV-Store/cmd/shutdown"
	"www.github.com/isaac-albert/Distributed-KV-Store/internal/raft"
)

/*
	model commands:
		kvstore node -id node0 -laddr localhost:11000  -raddr localhost:12000
		// when this above command is executed, it creates a stablestore file path as well as a log store filepath
		// if not already created and stores the files in ./internal/store/stablestore/nodeId
		// if this is the first node to be created, it checks whether the directories ./internal/store/stablestore and ./internal/store/logstore are created
		// and then stores them accordingly.
		kvstore http
		// why are we not giving a http address to listen from? it's cause the http can listen from any one of the nodes.
		// but even if you write to a follower node, it redirects to the leader node and writes from there.
		// when the number of nodes is 0, the http address will be not be able to listen since there are no leader nodes available, or any nodes
		// for that matter. so for this purpose we are required to start at least 1 node before we start the http server otherwise an error will be thrown out.
*/

var (
	nodeId     string
	listenAddr string
	raftAddr   string
)

var (
	_nodeNum              = 0
	_defaultListenAddress = "localhost:3000"
	_defaultRaftAddress   = "localhost:4000"
)

var (
	ErrInvalidCommand = errors.New("invalid command")
)

func init() {
	flag.StringVar(&nodeId, "id", fmt.Sprintf("node%d", _nodeNum), "sets the node id")
	flag.StringVar(&listenAddr, "laddr", _defaultListenAddress, "node listens on this port")
	flag.StringVar(&raftAddr, "raddr", _defaultRaftAddress, "intra cluster node port")
}

func GetArg() string {
	return flag.Args()[0]
}

func printFlags() {
	fmt.Println("node id is: ", nodeId)
	fmt.Println("listenAddr is: ", listenAddr)
	fmt.Println("raftAddr is: ", raftAddr)
}

func StartFlags() {
	flag.Parse()
	//printFlags()
	//fmt.Println("the main command is:", GetArg())
}

func GetCommand() error {
	switch GetArg() {
	case "node":
		err := raft.StartNode(nodeId, listenAddr, raftAddr)
		if err != nil {
			return fmt.Errorf("flag: %w", err)
		}
		return nil
	case "http":
		//StartHTTP()
		return nil
	case "shutdown":
		err := shutdown.ShutDown()
		if err != nil {
			return fmt.Errorf("error shutting down: %w", err)
		}
		return nil
	default:
		return ErrInvalidCommand
	}
}
