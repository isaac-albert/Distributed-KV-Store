package raft

import (
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
)

const (
	maxTCPConnPool = 8
	connTCPTimeout = 100 * time.Millisecond
)

type TCPNetAddr struct {
	Addr string
}

func newTcpNetAddr(a string) *TCPNetAddr { return &TCPNetAddr{Addr: a} }

func (t *TCPNetAddr) Network() string {
	return "tcp"
}
func (t *TCPNetAddr) String() string {
	return t.Addr
}

func NewRaftTCPTransport(raddr string, logger hclog.Logger) (*raft.NetworkTransport, error) {
	netAddr := newTcpNetAddr(raddr)

	return raft.NewTCPTransportWithLogger(raddr, netAddr, maxTCPConnPool, connTCPTimeout, logger)
}
