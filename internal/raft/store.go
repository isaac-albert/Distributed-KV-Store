package raft

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
)

type NodeStore struct {
	LogStore        *raftboltdb.BoltStore
	StableStore     *raftboltdb.BoltStore
	SnapsStore      raft.SnapshotStore
	LogStorePath    string
	StableStorePath string
	SnapStorePath   string
}

const (
	storeDirPrefix  = "assets/stores"
	sStoreSuffix    = "stablestore"
	lStoreSuffix    = "logstore"
	snapStoreSuffix = "snapstore"
)

const (
	sStoreFilePath = "stable.db"
	lStoreFilePath = "log.db"
)

const (
	snapRC = SnapsRetainCount
)

func (n *NodeStore) CreateStableStore(id string, logger hclog.Logger) (*raftboltdb.BoltStore, error) {

	var dirPath = filepath.Join(storeDirPrefix, id, sStoreSuffix)

	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		logger.Error("error creating stablestore directory", "path", dirPath, "error", err)
		return nil, fmt.Errorf("start stablestore: %w", err)
	}

	var filePath = filepath.Join(dirPath, sStoreFilePath)

	sStore, err := raftboltdb.NewBoltStore(filePath)
	if err != nil {
		logger.Error("error creating stable store", "path", filePath, "error", err)
		return nil, fmt.Errorf("start stablestore: %w", err)
	}

	logger.Info("successfully created a stable store")

	n.StableStore = sStore
	n.StableStorePath = filePath
	return sStore, nil
}

func (n *NodeStore) CreateLogStore(id string, logger hclog.Logger) (*raftboltdb.BoltStore, error) {

	var dirPath = filepath.Join(storeDirPrefix, id, lStoreSuffix)

	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		logger.Error("error creating log store directory", "path", dirPath, "error", err)
		return nil, fmt.Errorf("start logstore: %w", err)
	}

	var filePath = filepath.Join(dirPath, lStoreFilePath)

	lStore, err := raftboltdb.NewBoltStore(filePath)
	if err != nil {
		logger.Error("error creating log store", "path", filePath, "error", err)
		return nil, fmt.Errorf("start logstore: %w", err)
	}

	n.LogStore = lStore
	n.LogStorePath = filePath
	logger.Info("successfully created a log store")

	return lStore, nil
}

func (n *NodeStore) CreateSnapStore(id string, logger hclog.Logger) (*raft.FileSnapshotStore, error) {
	var baseDir = filepath.Join(storeDirPrefix, id, snapStoreSuffix)

	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("create snapstore: %w", err)
	}

	store, err := raft.NewFileSnapshotStoreWithLogger(baseDir, snapRC, logger)
	if err != nil {
		return nil, fmt.Errorf("create snapstore: %w", err)
	}

	logger.Info("successfully created a snap store")
	n.SnapsStore = store
	n.SnapStorePath = baseDir
	return store, nil
}
