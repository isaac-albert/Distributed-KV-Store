package raft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
)

const (
	StartNode = "node0"
)

type Server struct {
	HttpServer *http.Server
	Node       *Node
	Logger     hclog.Logger
}

type JoinRequest struct {
	Id      raft.ServerID      `json:"id"`
	Address raft.ServerAddress `json:"address"`
}

type NodeResp struct {
	Error string `json:"error"`
}

func NewServer(nodeId, httpAddr, raftAddr string) (*Server, error) {

	logConfig := hclog.LoggerOptions{
		Name:            nodeId,
		Level:           hclog.Debug,
		IncludeLocation: true,
		DisableTime:     true,
	}

	logger := hclog.New(&logConfig)

	server := &Server{
		Logger: logger,
	}

	err := server.getRaftNode(nodeId, httpAddr, raftAddr)
	if err != nil {
		return nil, err
	}

	server.getHttpServer(httpAddr)

	return server, nil
}

func (s *Server) getRaftNode(nodeID, httpAddr, raftAddr string) error {

	node, err := NewRaftNode(nodeID, httpAddr, raftAddr, s.Logger)
	if err != nil {
		return err
	}

	s.Node = node
	return nil
}

func (s *Server) getHttpServer(httpAddr string) error {

	mux := http.NewServeMux()

	mux.HandleFunc("/get", s.GetKV)
	mux.HandleFunc("/set", s.SetKV)
	mux.HandleFunc("/delete", s.DeleteKV)
	mux.HandleFunc("/join", s.JoinCluster)
	mux.HandleFunc("/leave", s.LeaveCluster)

	srv := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	s.HttpServer = srv
	return nil
}

func (s *Server) StartHTTPServer() error {
	if err := s.HttpServer.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

func (s *Server) ShutdownHTTP(ctx context.Context) error {

	err := s.HttpServer.Shutdown(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) ShutdownRaft() error {

	future := s.Node.Shutdown()
	if err := future.Error(); err != nil {
		return err
	}

	return nil
}

func (s *Server) ShutdownStores() error {

	err := os.RemoveAll("assets/")
	if err != nil {
		return fmt.Errorf("error removing assets: %w", err)
	}

	return nil
}

func (s *Server) AddNodeToNetwork(joinAddr string) error {
	if s.Node.ServerId == StartNode {
		err := s.Node.BootStrap()
		if err != nil {
			return err
		}
		s.Logger.Info("bootstrapped first node successfully")
	} else {
		err := s.sendJoinReqToNetwork(joinAddr)
		if err != nil {
			s.Logger.Error("error joining to network", "error", err)
			return err
		}
	}

	return nil
}

func (s *Server) sendJoinReqToNetwork(joinAddr string) error {

	payload := &JoinRequest{
		Id:      s.Node.ServerId,
		Address: s.Node.RaftAddr,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(data)

	URL_String := "http://" + joinAddr + "/join"

	resp, err := http.Post(URL_String, "application/json", buf)
	if err != nil {
		return fmt.Errorf("join request failed: %w", err)
	}

	if err := ParseNodeResponse(resp); err != nil {
		return err
	}
	
	return nil
}

func (s *Server) SetKV(w http.ResponseWriter, r *http.Request)        {}
func (s *Server) GetKV(w http.ResponseWriter, r *http.Request)        {}
func (s *Server) DeleteKV(w http.ResponseWriter, r *http.Request)     {}
func (s *Server) JoinCluster(w http.ResponseWriter, r *http.Request)  {}
func (s *Server) LeaveCluster(w http.ResponseWriter, r *http.Request) {}

func ParseNodeResponse(r *http.Response) error {
	defer r.Body.Close()
	
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	var nodeResp = &NodeResp{}

	err = json.Unmarshal(data, nodeResp)
	if err != nil {
		return fmt.Errorf("error unmarshalling data: %w", err)
	}

	if nodeResp.Error != "" {
		return fmt.Errorf("join request denied by leader: %s", nodeResp.Error)
	}

	return nil
}