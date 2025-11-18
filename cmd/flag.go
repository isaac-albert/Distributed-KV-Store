package cmd

import (
	"flag"
	"fmt"
)

/*
	Example command:
		kvstore startCluster -nodeId node1 -listenAddr localhost:11000  -raftAddr localhost:12000 -joinCluster=true
*/

var (
	nodeId      string
	listenAddr  string
	raftAddr    string
	joinCluster bool
)

var (
	nodeNum              = 0
	defaultListenAddress = "localhost:3000"
	defaultRaftAddress   = "localhost:4000"
)

func init() {
	flag.StringVar(&nodeId, "nodeId", fmt.Sprintf("node%d", nodeNum), "sets the node id")
	flag.StringVar(&listenAddr, "listenAddr", defaultListenAddress, "node listens on this port")
	flag.StringVar(&raftAddr, "raftAddr", defaultRaftAddress, "intra cluster node port")
	flag.BoolVar(&joinCluster, "joinCluster", true, "sets the node to join the cluster")
}

func Parse() {
	flag.Parse()
}

func GetArgs() []string {
	return flag.Args()
}

func PrintFlags() {
	fmt.Println("node id is: ", nodeId)
	fmt.Println("listenAddr is: ", listenAddr)
	fmt.Println("raftAddr is: ", raftAddr)
	fmt.Println("joinCluster is: ", joinCluster)
}
