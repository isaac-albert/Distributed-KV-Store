/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"

	"www.github.com/isaac-albert/Distributed-KV-Store/cmd"
)

func main() {
	/*
		func BootstrapCluster(
		conf *Config,
		logs LogStore,
		stable StableStore,
		snaps SnapshotStore,
		trans Transport,
		configuration Configuration,
		) error

		type Config:
			protocolversion : maxprotocolversion
			HeartbeatTimeout: time.Duration
			ElectionTimeout time.Duration
			CommitTimeout time.Duration
			MaxAppendEntries int - 5-10
			BatchApplyCh bool -> this is a variable which tells whether an applychannnel is buffered or not, since if it is buffered, possible failure is getting a timeout before processing the apply
			ShutdownOnRemove bool -> when RemovePeer is invoked and this is true then this wil shutdown the raft
			TrailingLogs uint64 -> the amount of trailing logs, which are the previous logs that have been snapshotted to keep in history so as to check faster follower inconsistency.
			SnapshotInterval time.Duration -> the leader must send a snapshot at an interval so to maintain inconsistencies across slow or new followers.
			SnapshotThreshold uint64 -> the amount of logs that can be at a time in storage before we take a snapshot.
			LeaderLeaseTimeout time.Duration -> is the time when a leader doesn't hear a majority of servers responding postively to it's appendEntriesRPC
			LocalID ServerID -> the name of the node which can be a string like "node-1", "server-1", an uuid, for protocolversion >= 3, for PV < 3 the serverID has to match the network address
			NotifyCh chan<- bool -> this is a channel which is either buffered or un-buffered, it tells whether a leader is present or not, for external use to write or send requests to the leader
			LogOutput io.Writer -> required if we are giving it a file or a os.Stderr or byte write
			LogLevel string -> determines the level of logs "TRACE", "DEBUG", "ERROR", etc
			Logger hclog.Logger -> when initiated with our own logger, it overrides log output and log level
			NoSnapshotRestoreOnStart bool -> setting this to true makes us to implement or own FSM reconstruction on startup or rebooting, "false" for current usage
			PreVoteDisabled bool -> when enabled a follower will check if it can get majority votes before starting an election (advised to be enabled)
			NoLegacyTelemetry bool -> uses old terms, set it to true

		type LogStore (interface):
			FirstIndex() (uint64, error) -> returns the first index (0), returns 0 for no entries
			LastIndex() (uint64, error) -> LastIndex returns the last index written. 0 for no entries.
			GetLog(index uint64, log *Log) error -> GetLog gets a log entry at a given index.
			StoreLog(log *Log) error ->  StoreLog stores a log entry.
				type Log:
					type LogType uint8:
						LogType describes various types of log entries.
			StoreLogs(logs []*Log) error -> // StoreLogs stores multiple log entries. By default the logs stored may not be contiguous with previous logs (i.e. may have a gap in Index since the last log written). If an implementation can't tolerate this it may optionally implement `MonotonicLogStore` to indicate that this is not allowed. This changes Raft's behaviour after restoring a user snapshot to remove all previous logs instead of relying on a "gap" to signal the discontinuity between logs before the snapshot and logs after.
			DeleteRange(min, max uint64) error -> DeleteRange deletes a range of log entries. The range is inclusive. : this is for followers whose log entries are inconsistent with the leader.

		type StableStore (interface): i have to use a bolt store which implements both logstore and stable store.
			Set(key []byte, val []byte) error -> sets the key configurations of a raft node.
			Get(key []byte) ([]byte, error) -> gets the key configurations of a raft node.
			SetUint64(key []byte, val uint64) error -> sets the key configurations of type uint64 of a raft node.
			GetUint64(key []byte) (uint64, error) -> gets the key configurations of type uint64 of a raft node.

		type SnapshotStore (interface):
			Create(
			version SnapshotVersion,
			index, term uint64,
			configuration Configuration,
			configurationIndex uint64,
			trans Transport
			) (SnapshotSink, error)
				SnapshotVersion : maxsnapshotversion
				index, term -> self explanatory
				Configuration -> is a list of servers in the cluster
				configuration Index -> is the index at which the server resided in the configuration
				trans -> transport (i have no idea)

	*/

	/*
		step - 1: create a new raft node using raft.NewRaft()
		step - 2: for config we have DefaultConfig()
		step - 3: bootstrap of a single node at the start and then call AddVoter() later
	*/

	fmt.Println("build executing perfectly")
	cmd.Parse()
	cmd.PrintFlags()
}
