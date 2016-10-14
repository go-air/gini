// Copyright 2016 The Gini Authors. All rights reserved.  Use of this source
// code is governed by a license that can be found in the License file.

package crisp

import (
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"sync"
)

// Type Server houses state for a server.
type Server struct {
	addr       Addr
	mu         sync.Mutex
	listener   net.Listener
	lastSolve  ProtoPoint
	maxClients int
	connChan   chan net.Conn
	trace      bool
	// The thing which generates the solvers for each connection.
}

// NewServer creates a new Gini protocol server
// n may be one of "tcp" or "unix".
func NewServer(addr string) (*Server, error) {
	a := ParseAddr(addr)
	if a.Network == "unix" {
		st, sterr := os.Stat(a.NetAddr)
		if sterr == nil {
			if st.Mode()&os.ModeSocket != 0 {
				os.Remove(a.NetAddr)
				// if failed, error on bind...
			} else {
				return nil, fmt.Errorf("will not remove %s to start a socket there.", a.NetAddr)
			}
		}
	}
	listener, e := net.Listen(a.Network, a.NetAddr)
	if e != nil {
		return nil, e
	}
	svr := &Server{
		addr:       *a,
		listener:   listener,
		lastSolve:  Unknown,
		maxClients: runtime.NumCPU(),
		connChan:   make(chan net.Conn)}
	return svr, nil
}

// Serve runs the server in a loop, accepting
// connections, generating a solver for each connection and binding them
func (s *Server) Serve() error {
	for i := 0; i < s.maxClients; i++ {
		h := newHandler(i+1, &s.trace)
		go h.serve(s.connChan)
	}
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("error accept: %s\n", err)
			s.Shutdown()
			return err
		}

		log.Printf("accepted from %s\n", conn.RemoteAddr().String())
		// wait for a free handler before accepting again.
		s.connChan <- conn
	}
}

// Trace tells the server whether to trace
func (s *Server) Trace(b bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.trace = b
}

// Shutdown stops listening for connections
func (s *Server) Shutdown() error {
	var e error
	e = s.listener.Close()
	if s.addr.Network == "unix" {
		if rme := os.Remove(s.addr.NetAddr); rme != nil {
			if e != nil {
				e = fmt.Errorf("shutdown errors: %s, %s", e, rme)
			} else {
				e = fmt.Errorf("shutdown error: %s", rme)
			}
		}
	}
	return e
}

// ListenAndServer starts a CRISP server
func ListenAndServe(addr string) error {
	log.SetPrefix("crispd: ")
	s, e := NewServer(addr)
	if e != nil {
		log.Printf("error new server: %s\n", e)
		return e
	}
	log.Printf("created server\n")
	defer func() {
		s.Shutdown()
	}()
	if e := s.Serve(); e != nil {
		log.Printf("error serve: %s\n", e)
		s.Shutdown()
		return e
	}
	log.Printf("serve returned without error\n")
	// unreachable
	return nil
}
