package flag

import (
	"errors"
	"flag"
	"fmt"
	"strings"
)

/*
	model commands:
		kvstore -id node0 -laddr localhost:11000  -raddr localhost:12000 node
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

type FlagConfig struct {
	Cmd string
	NodeID   string
	HttpAddr string
	RaftAddr string
	JoinAddr string
}

var (
	nodeId     string
	listenAddr string
	raftAddr   string
	joinAddr string
)

var (
	_NodeNum                 = 0
	_defaultFirstHTTPAddress = "127.0.0.1:3000"
	_defaultFirstRaftAddress = "127.0.0.1:4000"
	_defaultJoinAddress = _defaultFirstHTTPAddress
)

var (
	ErrInvalidCommand = errors.New("invalid command")
)

func init() {
	flag.StringVar(&nodeId, "id", fmt.Sprintf("node%d", _NodeNum), "sets the node id")
	flag.StringVar(&listenAddr, "laddr", _defaultFirstHTTPAddress, "node listens on this port")
	flag.StringVar(&raftAddr, "raddr", _defaultFirstRaftAddress, "intra cluster node port")
	flag.StringVar(&joinAddr, "join", _defaultJoinAddress, "to join to the first node in the cluster")
}

func GetArg() string {
	return flag.Args()[0]
}

func StartFlags() (*FlagConfig, error) {
	flag.Parse()
	cmd := GetArg()
	cmdL := strings.ToLower(cmd)
	if cmdL != "node" && cmdL != "shutdown" {
		return nil, ErrInvalidCommand
	}
	return &FlagConfig{
		Cmd: cmd,
		NodeID:   nodeId,
		HttpAddr: listenAddr,
		RaftAddr: raftAddr,
		JoinAddr: joinAddr,
	}, nil
}
