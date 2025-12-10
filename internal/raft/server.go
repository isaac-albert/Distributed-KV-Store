package raft

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
)

const (
	ResponseTimeout = 500 * time.Millisecond
	MaxBufferSize   = 1024 * 4
)

type Server struct {
	Mux         *http.ServeMux
	TCPListener net.Listener
	Node        *Node
	TCPAddr     string
}

type KV struct {
	Key   []byte `json:"key"`
	Value []byte `json:"value"`
}

func NewServer(addr string) *Server {
	tcpAddr, err := GetTCPAddress(addr)
	if err != nil {
		log.Panic(err)
	}

	listener, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		log.Panic(err)
	}

	server := &Server{
		TCPAddr:     tcpAddr,
		Mux:         http.NewServeMux(),
		TCPListener: listener,
	}

	server.Handlers()
	return server
}

func GetTCPAddress(addr string) (string, error) {
	hp := strings.Split(addr, ":")
	host := hp[0]
	port := hp[1]

	log.Println("the host, port is", "host", host, "port", port)

	portN, err := strconv.Atoi(port)
	if err != nil {
		return "", err
	}

	tcpPort := portN + 12
	hp[1] = strconv.Itoa(tcpPort)

	log.Println("new tcp port", "port", port)

	addr = strings.Join(hp, ":")

	return addr, nil
}

// func SetServers() ([]*Server, error) {

// 	var servers []*Server

// 	for _, node := range rt.ClusterNodes.Nodes {
// 		if node != nil {
// 			s := NewServer(node.Laddr)
// 			s.Node = node
// 			servers = append(servers, s)
// 		} else {
// 			//log.Fatal("node is bootstraped but is nil", "node", node)
// 			return nil, errors.New("node is bootstraped but is nil")
// 		}
// 	}

// 	return servers, nil
// }

// func StartServers() error {
// 	servers, err := SetServers()
// 	if err != nil {
// 		return fmt.Errorf("error getting servers: %w", err)
// 	}

// 	for _, server := range servers {
// 		go server.startServer(server.Node.Logger)
// 	}

// 	return nil
// }

func (s *Server) StartTCPCLient(addr string) {

}

func (s *Server) StartTCPServer(logger hclog.Logger) {
	for {
		conn, err := s.TCPListener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			logger.Error("error acceptign tcp connections", "error", err, "addr", s.TCPAddr)
			continue
		}

		go s.HandleTCPConn(conn)
	}

	logger.Info("shutting down tcp server at addr", "addr", s.TCPAddr)
}

func (s *Server) PushToTCPConn(data []byte, addr string) {
	//start tcp connection to address
	raddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		s.Node.Logger.Error("error resolving tcp address", "addr", addr, "error", err)
		return
	}

	conn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		s.Node.Logger.Error("error dialing to tcp address", "addr", addr, "error", err)
		return
	}
	defer conn.Close()

	//send data length to the tcp connection first
	dataLen := uint32(len(data))
	if err := binary.Write(conn, binary.BigEndian, dataLen); err != nil {
		s.Node.Logger.Error("error sending binary data length to tcp")
		return
	}

	_, err = conn.Write(data)
	if err != nil {
		s.Node.Logger.Error("error writing to tcp connection", "error", err, "addr", addr)
	}
}

func (s *Server) HandleTCPConn(c net.Conn) ([]byte, error) {
	defer c.Close()

	var length uint32
	if err := binary.Read(c, binary.BigEndian, length); err != nil {
		s.Node.Logger.Error("error reading binary legnth from connection")
		return nil, err
	}

	buf := make([]byte, int(length))

	_, err := io.ReadFull(c, buf)
	if err != nil {
		s.Node.Logger.Error("error reading data from tcp connection", "Error", err)
		return nil, err
	}

	return buf, nil
}

func (s *Server) startHTTPServer(logger hclog.Logger) {
	err := http.ListenAndServe(s.Node.Laddr, s.Mux)
	if err != nil {
		logger.Error("server failed", "error", err)
	}
}

func (s *Server) Handlers() {
	s.Mux.HandleFunc("/set", s.SetKV)
	s.Mux.HandleFunc("/get", s.GetKV)
}

func (s *Server) SetKV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "wrong method brotha, use 'POST'", http.StatusBadRequest)
		return
	}
	/*
		step -1: read the data from the body, if no body exists send an error.
		step -2: if leader Apply() if not then forward
		step - 3: apply to the leader
		step - 4: wati for leader's confirmation that it has committed
		step- 5: send message to the user
	*/

	leaderAddr, leaderId := s.Node.LeaderWithID()
	if leaderAddr == "" && leaderId == "" {
		http.Error(w, "error getting leader", http.StatusInternalServerError)
		return
	}

	// nodes := rt.ClusterNodes

	// leaderNode := nodes.GetNodeWithID(leaderId)

	// if leaderNode.Laddr != s.Node.Laddr && leaderNode.Raddr != s.Node.Raddr {
	// 	//redirect
	// 	return
	// }

	kv := &KV{}

	err := json.NewDecoder(r.Body).Decode(kv)
	if err != nil {
		http.Error(w, "error decoding the body", http.StatusBadRequest)
		return
	}

	var kvLog = KVOp{}

	kvLog.Op = "SET"
	kvLog.Key = []byte(kv.Key)
	kvLog.Value = []byte(kv.Value)

	// data, err := json.Marshal(kvLog)
	// if err != nil {
	// 	http.Error(w, "error marshalling the body to kvlog", http.StatusInternalServerError)
	// 	return
	// }

	// future := leaderNode.Apply(data, ResponseTimeout)
	// err = future.Error()
	// if err == raft.ErrAbortedByRestore {
	// 	log.Println("snapshot restore, retrying...")
	// 	time.Sleep(time.Millisecond * 100)
	// 	future := leaderNode.Apply(data, ResponseTimeout)
	// 	err = future.Error()
	// 	if err != nil {
	// 		http.Error(w, "error Applying after retry", http.StatusExpectationFailed)
	// 		return
	// 	}
	// }
	// if err == raft.ErrLeadershipLost {
	// 	http.Error(w, "leadership lost, retry apply", http.StatusInternalServerError)
	// 	return
	// }
	// if err != nil {
	// 	http.Error(w, "raft internal error", http.StatusInternalServerError)
	// 	return
	// }

	// result := future.Response()
	// resp, ok := result.(rt.FSMResponse)
	// if !ok {
	// 	http.Error(w, "error parsing the response", http.StatusInternalServerError)
	// 	return
	// }

	// if resp.Error != nil {
	// 	http.Error(w, "error writing to FSM", http.StatusExpectationFailed)
	// }

	// res := []byte(fmt.Sprintf("[ %v : %v ]", resp.Kv.Key, resp.Kv.Value))
	// w.WriteHeader(http.StatusOK)
	// w.Write(res)
}

func (s *Server) GetKV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "wrong method brotha, use 'GET'", http.StatusBadRequest)
		return
	}

	//URL = "http://localhost:3000/get/?key=junkie"

	_, err := url.Parse(r.URL.RawQuery)
	if err != nil {
		http.Error(w, "error reading key", http.StatusBadRequest)
		return
	}

}

func (s *Server) DeleteKV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "wrong method brotha, use 'DELETE'", http.StatusBadRequest)
		return
	}
}
