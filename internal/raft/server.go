package raft

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
)

const (
	StartNode           = "node0"
	AddVoterTimeout     = 5 * time.Second
	RemoveServerTimeout = 2 * time.Second
	LeaderApplyTimeout  = 500 * time.Millisecond
	BadRequestMsg       = "failed to decode request"
	VoterFailedToJoin   = "failed to join the cluster"
	ServerFailedToLeave = "failed to leave the server"
	SuccessfullJoin     = "successfully joined the cluster"
	SuccessfullLeave    = "successfully left the cluster"
)

type Server struct {
	HttpServer *http.Server
	Node       *Node
	Logger     hclog.Logger
}

type NodeRequest struct {
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
	mux.HandleFunc("/getleader", s.GetLeader)
	mux.HandleFunc("/removeserver", s.RemoveServer)

	srv := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	s.HttpServer = srv
	return nil
}

func (s *Server) RemoveServer(w http.ResponseWriter, r *http.Request) {

	var Resp = &struct{
		NodeID string `json:"node_id"`
	}{}

	err := json.NewDecoder(r.Body).Decode(Resp)
	if err != nil {
		s.Logger.Error("error decoding address", "error", err)
		http.Error(w, "error decoding body", http.StatusBadRequest)
		return 
	}

	future := s.Node.VerifyLeader()
	if err := future.Error(); err == nil {
		future := s.Node.RemoveServer(raft.ServerID(Resp.NodeID), 0, 0)
		if err := future.Error(); err != nil {
			s.Logger.Error("error failed to remove server", "error", err)
			SendJSONResponse(w, http.StatusInternalServerError, "error failed to remove server")
			return 
		}

		w.WriteHeader(http.StatusOK)
		SendJSONResponse(w, http.StatusOK, "successfully removed server")
		return 
	}

	leaderAddr, leaderID := s.Node.LeaderWithID()
	if leaderAddr == "" || leaderID == "" {
		s.Logger.Error("error leader not exists")
		http.Error(w, "leader not found", http.StatusInternalServerError)
		return
	}
	haddr, err := parseHttpAddr(string(leaderAddr))
	if err != nil {
		http.Error(w, "error parsing http address", http.StatusInternalServerError)
		return 
	}

	url := "http://" + haddr + "/removeserver"
	resp, err := http.Post(url, "application/json", r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return 
	}

	err = ParseNodeResponse(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "successfully removed server")
}

func (s *Server) GetLeader(w http.ResponseWriter, r *http.Request) {

	leaderAddr, leaderID := s.Node.LeaderWithID()
	if leaderAddr == "" || leaderID == "" {
		s.Logger.Error("leader not found")
		http.Error(w, "leader not found", http.StatusInternalServerError)
		return 
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, fmt.Sprintf("leader address: %s | leader id: %s", leaderAddr, leaderID))
}

func (s *Server) SetKV(w http.ResponseWriter, r *http.Request) {

	// s.Logger.Info("raft state", "state", s.Node.State())
	// addr, id :=  s.Node.LeaderWithID()
	// s.Logger.Info("raft leader", "leader","addr", addr, "id", id)
	// s.Logger.Info("raft term", "term", s.Node.CurrentTerm())

	future := s.Node.VerifyLeader()
	if err := future.Error(); err != nil {
		s.Logger.Error("leader changed")
		s.RedirectToLeader(w, r, OpSet)
		return
	}

	s.LeaderKVApply(w, r, OpSet)
}

func (s *Server) RedirectToLeader(w http.ResponseWriter, r *http.Request, cmd string) {

	//gets the leader addr, sends a request to leader using /set/redirect
	//successfull write to db, send ok

	raddr, id := s.Node.LeaderWithID()
	if raddr == "" || id == "" {
		s.Logger.Error("leader not found, retry....")
		http.Error(w, "leader not found, retry...", http.StatusInternalServerError)
		return
	}

	haddr, err := parseHttpAddr(string(raddr))
	if err != nil {
		s.Logger.Error("error parsing http address of leader", "error", err)
		http.Error(w, "error parsing http address of leader", http.StatusInternalServerError)
		return
	}

	url := ""
	switch cmd {
	case OpSet:
		url = "http://" + haddr + "/set"
	case OpDelete:
		url = "http://" + haddr + "/delete"
	default:
		s.Logger.Error("invalid command to leader redirect")
		return
	}

	_, err = http.Post(url, "application/json", r.Body)
	if err != nil {
		s.Logger.Error("error getting a response from leader apply", "error", err)
		http.Error(w, "error getting a resposne from leader apply", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "successfull transaction to leader \n")
}

func (s *Server) LeaderKVApply(w http.ResponseWriter, r *http.Request, cmd string) {

	// s.Logger.Info("raft state", "state", s.Node.State())
	// addr, id :=  s.Node.LeaderWithID()
	// s.Logger.Info("raft leader", "leader","addr", addr, "id", id)
	// s.Logger.Info("raft term", "term", s.Node.CurrentTerm())

	var kvop = &KVOp{
		Op: cmd,
	}

	err := json.NewDecoder(r.Body).Decode(kvop)
	if err != nil {
		s.Logger.Error("error decoding body", "error", err)
		http.Error(w, "error decoding body", http.StatusBadRequest)
		return
	}

	data, err := json.Marshal(kvop)
	if err != nil {
		s.Logger.Error("error marshalling data", "error", err)
		http.Error(w, "error marshalling data", http.StatusInternalServerError)
		return
	}

	future := s.Node.Apply(data, LeaderApplyTimeout)
	if err := future.Error(); err != nil {
		s.Logger.Error("error applying to leader", "error", err)
		http.Error(w, "error applying to leader", http.StatusInternalServerError)
		return
	}

	fsmResp := future.Response()

	rData, err := json.Marshal(fsmResp)
	if err != nil {
		//s.Logger.Error("error parsing response from fsm", "error", err)
		http.Error(w, "error parsing response from fsm", http.StatusInternalServerError)
		return
	}

	resp := &FSMResponse{}

	err = json.Unmarshal(rData, resp)
	if err != nil {
		s.Logger.Error("error parsing response from fsm", "error", err)
		http.Error(w, "error parsing fsm response", http.StatusBadGateway)
		return
	}

	if resp.Error != nil {
		s.Logger.Error("error fsm in setkv", "error", resp.Error)
		http.Error(w, "error setting kv to fsm", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "successfully applied to fsm")
}
func (s *Server) GetKV(w http.ResponseWriter, r *http.Request) {

	// s.Logger.Info("raft state", "state", s.Node.State())
	// addr, id :=  s.Node.LeaderWithID()
	// s.Logger.Info("raft leader", "leader","addr", addr, "id", id)
	// s.Logger.Info("raft term", "term", s.Node.CurrentTerm())

	s.Logger.Info("url string", "url", r.URL.Query())
	key := r.URL.Query()["key"][0]

	kv, err := s.Node.FSM.Get([]byte(key))
	if err != nil {
		s.Logger.Error("error finding the value of key", "error", err)
		http.Error(w, "error finding value of key", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, fmt.Sprintf("%s\n", kv.Value))
}

func (s *Server) DeleteKV(w http.ResponseWriter, r *http.Request) {

	future := s.Node.VerifyLeader()
	if err := future.Error(); err != nil {
		s.Logger.Error("leader changed")
		s.RedirectToLeader(w, r, OpDelete)
		return
	}

	s.LeaderKVApply(w, r, OpDelete)

}

func (s *Server) JoinCluster(w http.ResponseWriter, r *http.Request) {

	var nodeReq = &NodeRequest{}

	if err := json.NewDecoder(r.Body).Decode(nodeReq); err != nil {
		s.Logger.Error("error decoding node request", "path", r.URL.Path)
		SendJSONResponse(w, http.StatusBadRequest, BadRequestMsg)
		return
	}

	indFuture := s.Node.AddVoter(nodeReq.Id, nodeReq.Address, 0, AddVoterTimeout)
	if err := indFuture.Error(); err != nil {
		s.Logger.Error("error adding voter to leader", "error", err)
		SendJSONResponse(w, http.StatusInternalServerError, VoterFailedToJoin)
		return
	}

	SendJSONResponse(w, http.StatusAccepted, string(SuccessfullJoin))
}

func (s *Server) LeaveCluster(w http.ResponseWriter, r *http.Request) {

	var nodeReq = &NodeRequest{}

	err := json.NewDecoder(r.Body).Decode(nodeReq)
	if err != nil {
		s.Logger.Error("error reading leave request", "error", err)
		SendJSONResponse(w, http.StatusBadRequest, BadRequestMsg)
		return
	}
	future := s.Node.RemoveServer(nodeReq.Id, 0, RemoveServerTimeout)
	if err := future.Error(); err != nil {
		s.Logger.Error("error removing server from cluster", "error", err)
		SendJSONResponse(w, http.StatusInternalServerError, ServerFailedToLeave)
		return
	}

	SendJSONResponse(w, http.StatusAccepted, SuccessfullLeave)
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

	// s.Logger.Info("raft state", "state", s.Node.State())
	// addr, id :=  s.Node.LeaderWithID()
	// s.Logger.Info("raft leader", "leader","addr", addr, "id", id)
	// s.Logger.Info("raft term", "term", s.Node.CurrentTerm())

	if s.Node.ServerId == StartNode {
		future := s.Node.Shutdown()
		if err := future.Error(); err != nil {
			return err
		}
	} else {
		addr, id := s.Node.LeaderWithID()
		if addr == "" || id == "" {
			return errors.New("leader not found")
		}
		err := s.sendLeaveReqToNetwork(string(addr))
		if err != nil {
			return err
		}

		future := s.Node.Shutdown()
		if err := future.Error(); err != nil {
			return fmt.Errorf("error failed to shutdown raft node: %w", err)
		}
	}
	return nil
}

func (s *Server) sendLeaveReqToNetwork(leaveAddr string) error {

	// s.Logger.Info("raft state", "state", s.Node.State())
	// addr, id :=  s.Node.LeaderWithID()
	// s.Logger.Info("raft leader", "leader","addr", addr, "id", id)
	// s.Logger.Info("raft term", "term", s.Node.CurrentTerm())

	httpAddr, err := parseHttpAddr(leaveAddr)
	if err != nil {
		return fmt.Errorf("failed to parse http address from leader's address: %w", err)
	}

	payload := &NodeRequest{
		Id:      s.Node.ServerId,
		Address: s.Node.RaftAddr,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshall request: %w", err)
	}

	url := "http://" + httpAddr + "/leave"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("error sending request to leave: %w", err)
	}

	err = ParseNodeResponse(resp)
	if err != nil {
		return fmt.Errorf("failed to parse the node response: %w", err)
	}

	s.Logger.Info("left the server successfully, starting raft shutdown...")
	return nil
}

func (s *Server) ShutdownStores() error {

	storeDirPath := filepath.Join("assets", "stores", string(s.Node.ServerId))
	err := os.RemoveAll(storeDirPath)
	if err != nil {
		return fmt.Errorf("error removing stores: %w", err)
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

	payload := &NodeRequest{
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

	s.Logger.Info("the status code of join request", "code", resp.Status)

	if err := ParseNodeResponse(resp); err != nil {
		return err
	}

	s.Logger.Info("server successfully joined")

	return nil
}

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

	if r.StatusCode != http.StatusAccepted {
		return fmt.Errorf("error: %s", nodeResp.Error)
	}

	return nil
}

func parseHttpAddr(addr string) (string, error) {
	addrSlice := strings.Split(addr, ":")
	port, err := strconv.Atoi(addrSlice[1])
	if err != nil {
		return "", fmt.Errorf("failed to parse port number: %w", err)
	}

	addrSlice[1] = strconv.Itoa(port - 1000)
	httpAddr := strings.Join(addrSlice, ":")
	return httpAddr, nil
}

func SendJSONResponse(w http.ResponseWriter, statusCode int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(NodeResp{Error: msg})
}
