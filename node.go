package cluster

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

// Node defines a node of the cluster
type Node struct {
	name      string
	addr      string
	conn      net.Conn
	reader    *bufio.Reader
	writer    *bufio.Writer
	quit      chan bool
	joinTime  time.Time
	lag       time.Duration
	packets   int64
	statusStr string
	errorStr  string
}

const (
	// StatusNew is a new node
	StatusNew = "New"
	// StatusAuthenticating is a node doing authentication
	StatusAuthenticating = "Authenticating"
	// StatusShutdown is a node stopping
	StatusShutdown = "Stopping"
	// StatusOnline is a node online
	StatusOnline = "Online"
	// StatusLeaving is a node leaving
	StatusLeaving = "Leaving"
)

func newNode(name string, conn net.Conn) *Node {
	newNode := &Node{
		name:   name,
		conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
		quit:   make(chan bool),
	}
	return newNode
}

func (n *Node) ioReader(packetManager chan Packet, timeoutDuration time.Duration, quit chan bool) error {
	for {
		// Close connection when this function ends
		defer func() {
			n.conn.Close()
		}()

		for {
			// Set a deadline for reading. Read operation will fail if no data is received after deadline.
			n.conn.SetReadDeadline(time.Now().Add(timeoutDuration))

			bytes, err := n.reader.ReadBytes('\n')
			if err != nil {

				select {
				case <-quit:
					return fmt.Errorf("ioreader got quit signal for %s", n.name)
				default:
				}

				return fmt.Errorf("error reading from %s (%s)", n.name, err)
			}
			packet, err := UnpackPacket(bytes)
			if err != nil {
				fmt.Println(err)
				return fmt.Errorf("unable to unpack packet: %s", err) // fail if we do not understand the packet
			}
			packetManager <- *packet
		}

	}
}

func (n *Node) close() {
	// FIXME: nicer close with sync.Once (http://www.tapirgames.com/blog/golang-channel-closing)
	defer func() {
		if recover() != nil {
		}
	}()

	select {
	case <-n.quit:
		return
	default:
	}
	close(n.quit)
	n.conn.Close()
}
