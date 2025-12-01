package raft

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
)

const (
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

	raddr    raft.ServerAddress
	serverId raft.ServerID
	laddr    string
	Path     NodePath
}

type ClusterSeeds struct {
	Nodes []*Node
}

var ClusterNodes = NewClusterSeeds()

func NewClusterSeeds() *ClusterSeeds {
	return &ClusterSeeds{
		Nodes: make([]*Node, 0),
	}
}

func (c *ClusterSeeds) AddNode(node *Node) {
	c.Nodes = append(c.Nodes, node)
}

func (c *ClusterSeeds) GetNode() *Node {
	return c.Nodes[0]
}

func (c *ClusterSeeds) GetLeaderFromAddress(addr raft.ServerAddress, id raft.ServerID) *Node {
	for _, n := range c.Nodes {
		if n.raddr == addr && n.serverId == id {
			return n
		}
		continue
	}

	return nil
}

func (c *ClusterSeeds) GetLeader(ctx context.Context) (*Node, error) {

	var (
		leaderAddr raft.ServerAddress
		leaderID   raft.ServerID
	)

	ticker := time.Tick(GetLeaderTicker)

outer:
	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("leader not found")
		case <-ticker:
			n := c.GetNode()
			if n == nil {
				continue
			}
			leaderAddr = n.raddr
			leaderID = n.serverId
			break outer
		}
	}

	node := c.GetLeaderFromAddress(leaderAddr, leaderID)
	if node == nil {
		return nil, errors.New("error findiing leader node from the address")
	}

	return node, nil
}

func NewRaftNode(laddr string, raddr raft.ServerAddress, id raft.ServerID) *Node {

	return &Node{
		laddr:    laddr,
		raddr:    raddr,
		serverId: id,
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

	var newNode = NewRaftNode(laddr, raft.ServerAddress(raddr), raft.ServerID(id))

	sstore, err := newNode.Path.CreateStableStore(id, logger)
	if err != nil {
		return fmt.Errorf("start node: %w", err)
	}
	lstore, err := newNode.Path.CreateLogStore(id, logger)
	if err != nil {
		return fmt.Errorf("start node: %w", err)
	}

	snapstore, err := newNode.Path.CreateSnapStore(id, logger)
	if err != nil {
		return fmt.Errorf("start node: %w", err)
	}

	var config = raft.DefaultConfig()
	config.LocalID = raft.ServerID(id)

	transport, err := NewRaftTCPTransport(raddr, logger)
	if err != nil {
		return fmt.Errorf("start node: %w", err)
	}

	fsm := NewRaftFSM(id)

	node, err := raft.NewRaft(config, fsm, lstore, sstore, snapstore, transport)
	if err != nil {
		return fmt.Errorf("start node: %w", err)
	}

	newNode.Raft = node

	if newNode.serverId == "node0" {

		isExist, err := raft.HasExistingState(lstore, sstore, snapstore)
		if err != nil {
			logger.Error("has existing state error", "error", err)
			return err
		}

		if !isExist {
			future := newNode.BootstrapCluster(raft.Configuration{
				Servers: []raft.Server{
					{
						ID:      newNode.serverId,
						Address: newNode.raddr,
					},
				},
			})

			err = future.Error()
			if err != nil {
				return fmt.Errorf("start node: %w", err)
			}

			log.Println(newNode.String(), "created")

			ClusterNodes.AddNode(newNode)
		} else {
			logger.Error("state already exists node is trying to get in existing configuration")
			return fmt.Errorf("state already exists for node0")
		}

	} else {
		ctx, cancel := context.WithTimeout(context.Background(), GetLeaderTimeout)
		defer cancel()

		leaderNode, err := ClusterNodes.GetLeader(ctx)
		if err != nil {
			return fmt.Errorf("start node: %w", err)
		}

		index := leaderNode.LastIndex()
		leaderNode.AddVoter(newNode.serverId, newNode.raddr, index, AddVoterTimeout)
		ClusterNodes.AddNode(newNode)
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
