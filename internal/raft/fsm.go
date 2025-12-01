package raft

import (
	"io"

	"github.com/boltdb/bolt"
	"github.com/hashicorp/raft"
)

type RaftFSM struct {
	DB       bolt.DB
	Id       string
	key      string
	value    string
	snapshot raft.FSMSnapshot
}

func NewRaftFSM(id string) *RaftFSM {
	return &RaftFSM{Id: id}
}

func (r *RaftFSM) Apply(log *raft.Log) interface{} {
	return nil
}

func (r *RaftFSM) Snapshot() (raft.FSMSnapshot, error) {
	return r.snapshot, nil
}

func (r *RaftFSM) Restore(reader io.ReadCloser) error {
	return nil
}
