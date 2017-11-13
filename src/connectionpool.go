package signals

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"
)

type connectionPool struct {
	sync.RWMutex
	nodes []*Node
}

// dnsmanager holds all dns records known
//var connected = &connectionPool{
//nodes: make([]*Node),
//}

func (c *connectionPool) nodeAdd(newNode *Node) error {
	c.Lock()
	defer c.Unlock()
	// Check if node does not exist
	for _, node := range c.nodes {
		if node.name == newNode.name {
			return fmt.Errorf("Node %s already exists in connection pool", newNode.name)
		}
	}
	fmt.Printf("Adding node: %+v\n", newNode)
	c.nodes = append(c.nodes, newNode)
	return nil
}

func (c *connectionPool) nodeRemove(newNode *Node) error {
	c.Lock()
	defer c.Unlock()
	// Check if node does not exist
	remove := -1
	for id, node := range c.nodes {
		if node.name == newNode.name {
			remove = id
		}
	}
	if remove >= 0 {
		c.nodes = append(c.nodes[:remove], c.nodes[remove+1:]...)
	}
	return nil
}

func (c *connectionPool) getConnectCount() int {
	c.RLock()
	defer c.RUnlock()
	return len(c.nodes)
}

func (c *connectionPool) nodeExists(name string) bool {
	c.RLock()
	defer c.RUnlock()
	for _, node := range c.nodes {
		if node.name == name {
			return true
		}
	}
	return false
}

func (c *connectionPool) getSocket(name string) (conn net.Conn) {
	c.RLock()
	defer c.RUnlock()
	for _, node := range c.nodes {
		if node.name == name {
			return node.conn
		}
	}
	return
}

func (c *connectionPool) getAllSockets() (conns []net.Conn) {
	c.RLock()
	defer c.RUnlock()
	for _, node := range c.nodes {
		conns = append(conns, node.conn)
	}
	return
}

func (c *connectionPool) writeAll(p []byte) error {
	conns := c.getAllSockets()
	for _, conn := range conns {
		err := c.writeSocket(conn, p)
		if err != nil {
			return fmt.Errorf("writeAll failed: %s", err)
		}

	}
	return nil
}

func (c *connectionPool) write(name string, p []byte) error {
	conn := c.getSocket(name)
	err := c.writeSocket(conn, p)
	if err != nil {
		return fmt.Errorf("write failed: %s", err)
	}
	return nil
}

func (c *connectionPool) writeSocket(conn net.Conn, p []byte) error {
	fmt.Printf("Writing to socket: %+v", string(p))
	_, err := conn.Write(p)
	if err != nil {
		return fmt.Errorf("writeSocket failed: %s", err)
	}
	return nil
}

func (c *connectionPool) readSocket(conn net.Conn) (*Packet, error) {
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	reader := bufio.NewReader(conn)
	bytes, err := reader.ReadBytes('\n')
	packet, err := UnpackPacket(bytes)
	if err != nil {
		return nil, fmt.Errorf("unpack failed: %s", err)
	}
	return packet, nil
}

func (c *connectionPool) closeAll() {
	c.Lock()
	defer c.Unlock()
	for _, node := range c.nodes {
		node.conn.Close()
	}
}

func (c *connectionPool) close(name string) {
	c.Lock()
	defer c.Unlock()
	for _, node := range c.nodes {
		if node.name == name {
			node.conn.Close()
		}
	}
}
