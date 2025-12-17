# Distributed KV Store (Raft)

A production-grade, distributed Key-Value store built in Go, implementing the **Raft Consensus Algorithm** for strong consistency and fault tolerance. This system demonstrates how to build resilient distributed systems that can survive node failures while maintaining data integrity across a network.

## 📁 Table of Contents
- [What is Raft?](#-what-is-raft)
- [Architecture & Communication](#-architecture--communication)
- [The Finite State Machine (FSM)](#-the-finite-state-machine-fsm)
- [Production-Grade Practices](#-production-grade-practices)
- [Installation & Usage](#-installation--usage)
- [Future Roadmap](#-future-roadmap)

---

## 🧠 What is Raft?

Raft is a consensus algorithm designed to be easy to understand while providing the same fault tolerance and performance as Paxos.

### Use Case: Why do we need it?
In a single-node database, if the disk fails, you lose data. To prevent this, we replicate data across multiple servers (nodes). However, this introduces a new problem: **Consistency**.
* If Node A thinks `x=1` and Node B thinks `x=2`, which one is right?
* If the leader crashes, who takes over?

Raft solves these problems by ensuring that a cluster of nodes agrees on a strictly ordered sequence of changes (the **Log**). If a majority of nodes acknowledge a change, it is "committed" and will never be lost.



### How it Works (Simplified)
1.  **Leader Election:** The cluster elects one node as the **Leader**. All writes go to this leader.
2.  **Log Replication:** The leader receives a command (e.g., `SET foo bar`), appends it to its log, and sends it to all Followers.
3.  **Commit:** Once a majority of followers acknowledge the entry, the leader "Commits" it (executes it) and tells followers to do the same.

---

## 📡 Architecture & Communication

This project uses **HashiCorp's Raft implementation**, the industry standard library used by Consul and Nomad.

### Inter-Node Communication (Transport Layer)
Raft nodes must constantly talk to each other to send heartbeats ("I'm alive") and replicate logs. While Raft can theoretically run over any transport (UDP, gRPC, InMemory), we utilize **TCP (Transmission Control Protocol)**.

**Why TCP?**
* **Reliability:** We cannot afford to drop packets containing consensus data. TCP guarantees delivery and ordering.
* **Stream Support:** TCP allows us to maintain persistent connections, which is efficient for the constant stream of heartbeats required by Raft.
* **Production Standard:** It is the default, battle-tested transport mechanism for high-reliability distributed systems.

**Network Topology:**
* **HTTP Layer:** Handles client requests (`GET`, `SET`, `/join`, `/leave`).
* **TCP Layer:** Dedicated exclusively to Raft internal traffic (AppendEntries, RequestVote).

---

## ⚙️ The Finite State Machine (FSM)

The FSM is often the most confusing part of Raft for newcomers. Here is the mental model to visualize it.

### The Mental Model
An FSM is a type of machine, function, or piece of code that can change its state—meaning it manages a specific piece of memory that can be modified.
* **The Memory:** Think of this as a Map, a Slice, an Object, or in our case, a **Database** (BoltDB). Imagine the DB as a single block of memory or a single file.
* **The State Change:** This memory's state changes only when a specific operation is applied (e.g., a new value is added, updated, or deleted).


In Raft, we ensure that every node applies the *exact same operations* to their FSM in the *exact same order*, guaranteeing that the final state of the "Memory" (BoltDB) is identical across the cluster.

**The Interface Implementation:**
1.  **`Apply(log)`**: We parse the incoming log command (e.g., `SET key=val`) and update the BoltDB file.
2.  **`Snapshot()`**: We capture the current state of the BoltDB file to save a backup.
3.  **`Restore(snapshot)`**: We wipe the current memory state and replace it with the backup data from the snapshot.

---

## 🛡️ Production-Grade Practices

This is not just a toy project; it adheres to strict systems programming discipline:

* **Graceful Shutdowns:** Utilizes `signal.NotifyContext` and `waitGroups` to ensure no data is corrupted during a `SIGINT/SIGTERM`. The HTTP server, Raft engine, and DB engine shut down in a specific order.
* **Structured Logging:** Uses `hclog` for level-based, machine-parsable logs, critical for debugging distributed race conditions.
* **IO Streaming:** Snapshots and DB restorations use `io.Reader/Writer` streams (via `io.Copy`) rather than loading entire files into RAM, ensuring the system doesn't crash on large datasets.
* **Safe Concurrency:** HTTP endpoints are non-blocking where appropriate, and the FSM uses BoltDB's ACID transactions to ensure thread safety during concurrent reads/writes.
* **Robust Join Protocol:** Implements a handshake mechanism where new nodes explicitly request to join the cluster via an HTTP API control plane before being allowed into the Raft consensus group.

---

## 🚀 Future Roadmap

We are planning to push the boundaries of this system further:

### 1. Sharded KV Store (Multi-Raft)
**The Problem:** A single Raft group is limited by the disk speed of the leader. It cannot scale writes horizontally.
**The Plan:** Implement **Sharding**.
* Split the key space into ranges (e.g., A-M, N-Z).
* Run **multiple Raft groups** in parallel.
* This achieves **Parallelism in Consensus**, allowing us to scale write throughput linearly while maintaining strong consistency within each shard.

### 2. Observability Stack
* Integrate Prometheus metrics to visualize:
    * Raft state changes (Leader/Follower transitions).
    * Log commit latency.
    * Disk I/O throughput.

---

## 📦 Installation & Usage

**Build:**
```bash
go build -o kvstore
```

**Run Node1 (Leader):**
```./kvstore -id node0 -laddr 127.0.0.1:3000 -raddr 127.0.0.1:4000```

**Run Node 2:**
```./kvstore -id node1 -laddr 127.0.0.1:3001 -raddr 127.0.0.1:4001 -join 127.0.0.1:3000```

**Run Node 3:**
```./kvstore -id node2 -laddr 127.0.0.1:3002 -raddr 127.0.0.1:4002 -join 127.0.0.1:3000```

