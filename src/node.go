package signals

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

// Node defines a node of the cluster
type Node struct {
	name       string
	addr       string
	authorized bool
	conn       net.Conn
	reader     *bufio.Reader
	writer     *bufio.Writer
}

func newNode(conn net.Conn) *Node {
	bufReader := bufio.NewReader(conn)
	bufWriter := bufio.NewWriter(conn)
	node := &Node{
		conn:   conn,
		reader: bufReader,
		writer: bufWriter,
	}
	return node
}

func (n *Node) connect() error {
	conn, err := net.DialTimeout("tcp", n.addr, 10*time.Second)
	if err != nil {
		return err
	}
	n.conn = conn
	n.reader = bufio.NewReader(conn)
	n.writer = bufio.NewWriter(conn)
	return nil
}

func (n *Node) ioReader(packetManager chan Packet) {
	for {
		fmt.Printf("New client connection... %+v\n", n)

		// Close connection when this function ends
		defer func() {
			fmt.Println("Closing connection...")
			n.conn.Close()
		}()

		timeoutDuration := 5 * time.Second

		for {
			// Set a deadline for reading. Read operation will fail if no data
			// is received after deadline.
			n.conn.SetReadDeadline(time.Now().Add(timeoutDuration))

			// Read tokens delimited by newline
			bytes, err := n.reader.ReadBytes('\n')
			if err != nil {
				fmt.Println(err)
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

func (n *Node) authClient(authKey string) bool {
	fmt.Println("Authenticating client")

	n.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	fmt.Println("Reading bytes...")
	bytes, err := n.reader.ReadBytes('\n')

	//header, auth, err := UnpackPacket(bytes, new(Auth))
	packet, err := UnpackPacket(bytes)
	if err != nil {
		return false
	}

	auth, err := packet.Message(new(Auth))
	if err != nil {
		return false
	}

	n.name = packet.Name
	fmt.Printf("New Node Auth: %+v", auth)
	if auth.(*Auth).AuthKey != authKey {
		return false
	}
	fmt.Println("Client Authenticated")
	return true
}
