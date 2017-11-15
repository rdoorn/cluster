package cluster

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type connectionPool struct {
	sync.RWMutex
	nodes []*Node
}

func (c *connectionPool) nodeAdd(newNode *Node) (*Node, error) {
	c.Lock()
	defer c.Unlock()
	// Check if node does not exist
	for _, node := range c.nodes {
		if node.name == newNode.name {
			return node, fmt.Errorf("Node %s already exists in connection pool add(%s->%s) existing(%s->%s)", newNode.name, newNode.conn.LocalAddr(), newNode.conn.RemoteAddr(), node.conn.LocalAddr(), node.conn.RemoteAddr())
		}
	}
	c.nodes = append(c.nodes, newNode)
	return nil, nil
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

func (c *connectionPool) getSocket(name string) (net.Conn, error) {
	c.RLock()
	defer c.RUnlock()
	for _, node := range c.nodes {
		if node.name == name {
			return node.conn, nil
		}
	}
	return nil, fmt.Errorf("node not found: %s", name)
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
	var errors []string
	conns := c.getAllSockets()
	for _, conn := range conns {
		err := c.writeSocket(conn, p)
		if err != nil { // collect errors, try to send to the others
			errors = append(errors, err.Error())
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("writeAll failed: %s", strings.Join(errors, ","))
	}

	return nil
}

func (c *connectionPool) write(name string, p []byte) error {
	conn, err := c.getSocket(name)
	if err != nil {
		return fmt.Errorf("write failed: %s", err)
	}
	err = c.writeSocket(conn, p)
	if err != nil {
		return fmt.Errorf("write failed: %s", err)
	}
	return nil
}

func (c *connectionPool) writeSocket(conn net.Conn, p []byte) error {
	//fmt.Printf("Writing to socket: %+v", string(p))
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
		node.close()
	}
}

func (c *connectionPool) close(name string) {
	c.Lock()
	defer c.Unlock()
	for _, node := range c.nodes {
		if node.name == name {
			node.close()
		}
	}
}
