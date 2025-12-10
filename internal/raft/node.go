package raft

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
)

const (
	BOOTSTRAP_NODE = "node0"
	SnapsRetainCount    = 5
	ProcessRPCTimeout   = 100 * time.Millisecond
	AddVoterTimeout     = 200 * time.Millisecond
	GetLeaderTimeout    = 500 * time.Millisecond
	GetValidNodeTimeout = 200 * time.Millisecond
	GetNodeTicker       = 50 * time.Millisecond
	GetLeaderTicker     = 50 * time.Millisecond
)

type Node struct {
	*raft.Raft

	FSM       *RaftFSM
	Raddr     raft.ServerAddress
	ServerId  raft.ServerID
	Logger    hclog.Logger
	Laddr     string
	Stores    NodeStore
	Transport raft.Transport
}

func NewRaftNode(laddr string, Raddr raft.ServerAddress, id raft.ServerID, logger hclog.Logger) *Node {

	return &Node{
		Laddr:    laddr,
		Raddr:    Raddr,
		ServerId: id,
		Logger:   logger,
	}
}

func StartNode(id, laddr, raddr string) error {

	logConfig := hclog.LoggerOptions{
		Name:            id,
		Level:           hclog.Debug,
		IncludeLocation: true,
		DisableTime:     true,
	}

	logger := hclog.New(&logConfig)

	var newNode = NewRaftNode(laddr, raft.ServerAddress(raddr), raft.ServerID(id), logger)

	sstore, err := newNode.Stores.CreateStableStore(id, logger)
	if err != nil {
		logger.Error("error getting stable store", "store", sstore)
		return fmt.Errorf("start node: %w", err)
	}
	lstore, err := newNode.Stores.CreateLogStore(id, logger)
	if err != nil {
		logger.Error("error getting log store", "store", lstore)
		return fmt.Errorf("start node: %w", err)
	}

	snapstore, err := newNode.Stores.CreateSnapStore(id, logger)
	if err != nil {
		logger.Error("error getting snap store", "store", snapstore)
		return fmt.Errorf("start node: %w", err)
	}

	var config = raft.DefaultConfig()
	config.LocalID = raft.ServerID(id)

	logger.Info("configuration of the new node", "config", config)

	//used Raddr for the same net.Addr, in production use laddr
	transport, err := NewRaftTCPTransport(raddr, raddr, logger)
	if err != nil {
		logger.Error("error getting valid transport", "transport", transport)
		return fmt.Errorf("start node transport: %w", err)
	}
	newNode.Transport = transport

	fsm, err := NewRaftFSM(id)
	if err != nil {
		logger.Error("error creating new fsm for the node", "fsm", fsm)
		return fmt.Errorf("error creating new fsm: %w", err)
	}
	newNode.FSM = fsm

	node, err := raft.NewRaft(config, fsm, lstore, sstore, snapstore, transport)
	if err != nil {
		logger.Error("error creating new Raft node with arguments",
			"config", config)
		return fmt.Errorf("start node: %w", err)
	}

	newNode.Raft = node

	if newNode.ServerId == BOOTSTRAP_NODE {

		isExist, err := raft.HasExistingState(lstore, sstore, snapstore)
		if err != nil {
			logger.Error("error getting the state of the server", "error", err)
			return err
		}

		if !isExist {
			future := newNode.BootstrapCluster(raft.Configuration{
				Servers: []raft.Server{
					{
						ID:      newNode.ServerId,
						Address: newNode.Raddr,
					},
				},
			})

			err = future.Error()
			if err != nil {
				return fmt.Errorf("start node: %w", err)
			}

			logger.Info("created new server", "server", newNode.String())

			server := NewServer(laddr)
			go server.StartTCPServer(logger)
			go server.startHTTPServer(logger)

			BootStrapConfig(newNode.Laddr, string(newNode.Raddr), string(id), logger)

			//ClusterNodes.AddNode(newNode)
		} else {
			logger.Error("state already exists, use [ nodeN ] N > 0 for new nodes",
				"Log_Store", newNode.Stores.LogStorePath,
				"Stable_Store", newNode.Stores.StableStorePath,
				"Snap_Store", newNode.Stores.StableStorePath)
			return fmt.Errorf("state already exists for node0")
		}

	} else {
		_, cancel := context.WithTimeout(context.Background(), GetLeaderTimeout)
		defer cancel()

		//leaderNode, err := ClusterNodes.GetLeader(ctx)
		if err == nil {
			return fmt.Errorf("start node: %w", err)
		}

		// index := leaderNode.LastIndex()
		// leaderNode.AddVoter(newNode.ServerId, newNode.Raddr, index, AddVoterTimeout)
		// ClusterNodes.AddNode(newNode)
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
