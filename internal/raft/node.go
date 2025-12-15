package raft

import (
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
)

const (
	BOOTSTRAP_NODE              = "node0"
	SnapsRetainCount            = 5
	ProcessRPCTimeout           = 100 * time.Millisecond
	// AddVoterTimeout             = 200 * time.Millisecond
	GetLeaderTimeout            = 500 * time.Millisecond
	GetValidNodeTimeout         = 200 * time.Millisecond
	GetNodeTicker               = 50 * time.Millisecond
	GetLeaderTicker             = 50 * time.Millisecond
	StagingVoterDurationTimeout = 100 * time.Millisecond
)

type Node struct {
	*raft.Raft

	FSM      *RaftFSM
	ServerId raft.ServerID
	RaftAddr raft.ServerAddress
	HTTPAddr string
	Logger   hclog.Logger
}

func NewRaftNode(nodeId, httpAddr, raftAddr string, logger hclog.Logger) (*Node, error) {

	node := &Node{
		ServerId: raft.ServerID(nodeId),
		RaftAddr: raft.ServerAddress(raftAddr),
		HTTPAddr: httpAddr,
		Logger:   logger,
	}

	err := node.StartNode()
	if err != nil {
		return nil, err
	}


	return node, nil
}

func (n *Node) StartNode() error {

	sstore, err := CreateStableStore(string(n.ServerId), n.Logger)
	if err != nil {
		n.Logger.Error("error getting stable store", "error", err)
		return err
	}
	lstore, err := CreateLogStore(string(n.ServerId), n.Logger)
	if err != nil {
		n.Logger.Error("error getting log store", "error", err)
		return err
	}

	snapstore, err := CreateSnapStore(string(n.ServerId), n.Logger)
	if err != nil {
		n.Logger.Error("error getting snap store", "error", err)
		return err
	}

	var config = raft.DefaultConfig()
	config.LocalID = raft.ServerID(n.ServerId)

	n.Logger.Info("Config of the new node", "config", config)

	//used Raddr for the same net.Addr, in production use HTTPAddr
	transport, err := NewRaftTCPTransport(string(n.RaftAddr), n.Logger)
	if err != nil {
		n.Logger.Error("error getting valid transport", "error", err)
		return err
	}

	fsm, err := NewRaftFSM(string(n.ServerId))
	if err != nil {
		n.Logger.Error("error creating new fsm for the node", "error", err)
		return err
	}

	n.FSM = fsm

	ok, err := raft.HasExistingState(lstore, sstore, snapstore)
	if err != nil {
		n.Logger.Error("error getting the state of the node", "error", err)
		return err
	}

	if ok {
		n.Logger.Warn("node of server id has an existing state", "serverID", n.ServerId)
		return errors.New("error: node already exists")
	}

	ne, err := raft.NewRaft(config, fsm, lstore, sstore, snapstore, transport)
	if err != nil {
		n.Logger.Error("error creating new Raft node with argument", "error", err)
		return err 
	}

	n.Raft = ne
	return nil
}

func (n *Node) BootStrap() error {
	future := n.BootstrapCluster(
		raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      n.ServerId,
					Address: n.RaftAddr,
				},
			},
		},
	)

	err := future.Error()
	if err != nil {
		return fmt.Errorf("bootstrap node: %w", err)
	}

	return nil
}

/*
	type Transport: is an interface that allows network transport to allow raft to communicate with other nodes.
	type NetworkTransport:
	general network transport: stream layer is absracted (gRPC is used here)
	NewTCPTransport: the stream layer is TCP:
		has bind addr and listen addr
	if we want a gRPC, we have to implement the StreamLayer interface for the gRPC, then use it in NewNetworkTransport.

*/
