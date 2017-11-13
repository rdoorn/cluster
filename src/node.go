package signals

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

// Node defines a node of the cluster
type Node struct {
	name   string
	addr   string
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	quit   chan bool
}

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

func (n *Node) ioReader(packetManager chan Packet, quit chan bool) {
	for {
		// Close connection when this function ends
		defer func() {
			fmt.Printf("Closing connection of %s\n", n.name)
			n.conn.Close()
		}()

		timeoutDuration := 11 * time.Second

		for {
			// Set a deadline for reading. Read operation will fail if no data
			// is received after deadline.
			n.conn.SetReadDeadline(time.Now().Add(timeoutDuration))

			// Read tokens delimited by newline
			bytes, err := n.reader.ReadBytes('\n')
			if err != nil {

				select {
				case <-quit:
					fmt.Printf("ioreader got quit signal for %s\n", n.name)
					return
				default:
				}

				fmt.Printf("Error reading from %s (%s)\n", n.name, err)
				return
			}
			packet, err := UnpackPacket(bytes)
			if err != nil {
				fmt.Println(err)
				return // fail if we do not understand the packet
			}
			packet.source = fmt.Sprintf("%s", n.conn)
			packetManager <- *packet
		}

	}
}
