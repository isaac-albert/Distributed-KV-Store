package raft

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/hashicorp/go-hclog"
)

const (
	configFile = "cluster.json"
)

var (
	ErrInconsistentWriteToFile = errors.New("incomplete write to file")
	ErrEmptyConfigFile         = errors.New("cluster is empty, no available servers in config.json")
)

type NodeConfig struct {
	Id         string `json:"id"`
	ListenAddr string `json:"laddr"`
	RaftAddr   string `json:"raddr"`
}

func BootStrapConfig(laddr, raddr, id string, logger hclog.Logger) error {

	nodes, err := GetData(logger)
	if err != nil {
		return err
	}

	jsonify := NodeConfig{
		Id:         id,
		ListenAddr: laddr,
		RaftAddr:   raddr,
	}

	err = SetData(nodes, jsonify, logger)
	if err != nil {
		return err
	}

	return nil
}

func GetData(logger hclog.Logger) ([]NodeConfig, error) {
	data, err := os.ReadFile(configFile)
	var nodes = []NodeConfig{}

	if err == nil {
		if err := json.Unmarshal(data, &nodes); err != nil {
			logger.Error("error unmarshalling file content", "error", err)
		}
	} else if !os.IsNotExist(err) {
		logger.Error("error reading cluster.json", "error", err)
		return nil, err
	}

	return nodes, nil
}

func SetData(nodes []NodeConfig, jsonify NodeConfig, logger hclog.Logger) error {

	nodes = append(nodes, jsonify)

	data, err := json.Marshal(nodes)
	if err != nil {
		logger.Error("error marshalling node to cluster.json", "nodes", nodes)
		return err
	}

	err = os.WriteFile(configFile, data, 0644)
	if err != nil {
		logger.Error("error writing to file", "file", configFile, "data", data)
		return err
	}

	logger.Info("Successfull write to config.json of node")
	return nil
}
