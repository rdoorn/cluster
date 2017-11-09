package signals

import (
	"fmt"
	"net"
)

type Server struct {
	addr     string
	close    chan bool
	listener net.Listener
	timeout  int
}

func (s *Server) Listen() (ln net.Listener, err error) {
	s.listener, err = net.Listen("tcp", s.addr)
	if err != nil {
		return
	}
	return s.listener, nil
}

func (s *Server) Serve(newSocket chan net.Conn) {
	defer s.listener.Close()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		newSocket <- conn
	}
}
