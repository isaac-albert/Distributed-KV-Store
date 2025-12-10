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

func NewRaftTCPTransport(raddr string, laddr string, logger hclog.Logger) (*raft.NetworkTransport, error) {
	
	netAddr := newTcpNetAddr(raddr)

	logger.Info("transport addresses", "net addr", netAddr, "raddr", raddr, "laddr", laddr)
	return raft.NewTCPTransportWithLogger(laddr, nil, maxTCPConnPool, connTCPTimeout, logger)
}
