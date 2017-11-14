package signals

import (
	"net"
)

// Server defines the server part of the cluster service
type Server struct {
	addr     string
	close    chan bool
	listener net.Listener
	timeout  int
}

// Listen creates the listener for the cluster server
func (s *Server) Listen() (ln net.Listener, err error) {
	s.listener, err = net.Listen("tcp", s.addr)
	if err != nil {
		return
	}
	return s.listener, nil
}

// Serve accepts connections and forwards these to the cluster server
func (s *Server) Serve(newSocket chan net.Conn, quit chan bool) {
	defer s.listener.Close()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-quit:
				//fmt.Printf("%s Quit initiated on Serve\n", time.Now())
				return
			default:
			}

			continue
		}
		newSocket <- conn
	}
}
