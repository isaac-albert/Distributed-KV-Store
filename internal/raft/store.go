package raft

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
)

type NodePath struct {
	LogStore       *raftboltdb.BoltStore
	StableStore    *raftboltdb.BoltStore
	SnapsStorePath string
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

func (n *NodePath) CreateStableStore(id string, logger hclog.Logger) (*raftboltdb.BoltStore, error) {

	var dirPath = filepath.Join(storeDirPrefix, id, sStoreSuffix)

	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		logger.Error("error creating stablestore directory", "path", dirPath, "error", err)
		return nil, fmt.Errorf("start stablestore: %w", err)
	}

	logger.Info("created the directory: %s using MkDirAll", "path", dirPath)
	var filePath = filepath.Join(dirPath, sStoreFilePath)
	logger.Info("created a file path")

	sStore, err := raftboltdb.NewBoltStore(filePath)
	if err != nil {
		logger.Error("error creating stable store", "path", filePath, "error", err)
		return nil, fmt.Errorf("start stablestore: %w", err)
	}

	logger.Info("successfully created a stable store")

	n.StableStore = sStore
	return sStore, nil
}

func (n *NodePath) CreateLogStore(id string, logger hclog.Logger) (*raftboltdb.BoltStore, error) {

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
	logger.Info("successfully created a log store")

	return lStore, nil
}

func (n *NodePath) CreateSnapStore(id string, logger hclog.Logger) (*raft.FileSnapshotStore, error) {
	var baseDir = filepath.Join(storeDirPrefix, id, snapStoreSuffix)
	n.SnapsStorePath = baseDir

	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("create snapstore: %w", err)
	}

	store, err := raft.NewFileSnapshotStoreWithLogger(baseDir, snapRC, logger)
	if err != nil {
		return nil, fmt.Errorf("create snapstore: %w", err)
	}

	return store, nil
}
