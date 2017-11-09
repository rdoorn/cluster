package signals

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"reflect"
	"sync"
	"time"
)

// Manager is the main cluster manager
type Manager struct {
	sync.RWMutex
	name            string
	configuredNodes map[string]Node
	activeNodes     map[string]*Node
	listener        net.Listener
	newSocket       chan net.Conn    // new clients connecting
	nodeJoin        chan string      // returns string of the node joining
	nodeLeave       chan string      // returns string of the node leaving
	packetManager   chan Packet      // packets sent to packet manager
	FromCluster     chan Packet      // data received from cluster
	ToCluster       chan interface{} // data send to cluster
	authKey         string
}

// NewManager creates a new cluster manager
func NewManager(name, key string) *Manager {
	fmt.Println("Starting new Manager")
	m := &Manager{
		name:            name,
		configuredNodes: make(map[string]Node),
		activeNodes:     make(map[string]*Node),
		newSocket:       make(chan net.Conn),
		nodeJoin:        make(chan string),
		nodeLeave:       make(chan string),
		packetManager:   make(chan Packet),
		FromCluster:     make(chan Packet),
		ToCluster:       make(chan interface{}),
		authKey:         key,
	}
	return m
}

// AddClusterNode adds a cluster node to the cluster to be connected to
func (m *Manager) AddClusterNode(n Node) {
	m.Lock()
	defer m.Unlock()
	m.configuredNodes[n.name] = n
}

// AddActiveNode adds a cluster node to the cluster to the list of authenticated connections
func (m *Manager) AddActiveNode(n *Node) {
	m.Lock()
	defer m.Unlock()
	//m.activeNodes = append(m.activeNodes, n)
	m.activeNodes[n.name] = n
}

// RemoveActiveNode removes a cluster node from the cluster to the list of authenticated connections
func (m *Manager) RemoveActiveNode(n *Node) {
	m.Lock()
	defer m.Unlock()
	delete(m.activeNodes, n.name)
}

// ListenAndServe starts the listener and serves connections to clients
func (m *Manager) ListenAndServe(addr string) (err error) {
	fmt.Println("Starting listener")
	s := Server{addr: addr}
	m.listener, err = s.Listen()
	if err == nil {
		go s.Serve(m.newSocket)           // accepts new connections and passes them on to the manager
		go m.handleIncommingConnections() // handles new socket connections
		go m.handleOutgoingConnections()  // creates connections to remote nodes
		go m.handlePackets()              // handles all incomming packets
	}
	return
}

// Shutdown stops the cluster node
func (m *Manager) Shutdown() {
	fmt.Println("Stopping listener")
	m.Lock()
	defer m.Unlock()
	m.listener.Close()
}

func (m *Manager) getConfiguredNodes() (nodes []string) {
	m.RLock()
	defer m.RUnlock()
	for nodeName := range m.configuredNodes {
		nodes = append(nodes, nodeName)
	}
	return
}

func (m *Manager) getConfiguredNode(node string) (Node, error) {
	m.RLock()
	defer m.RUnlock()
	if node, ok := m.configuredNodes[node]; ok {
		return node, nil
	}
	return Node{}, fmt.Errorf("Node %s is not configured", node)
}

func (m *Manager) getActiveNodes() (nodes []string) {
	m.RLock()
	defer m.RUnlock()
	for nodeName := range m.activeNodes {
		nodes = append(nodes, nodeName)
	}
	return
}

func (m *Manager) getActiveNode(node string) (n *Node, err error) {
	m.RLock()
	defer m.RUnlock()
	if node, ok := m.activeNodes[node]; ok {
		return node, nil
	}
	return n, fmt.Errorf("Node %s is not active", node)
}

func (m *Manager) handleOutgoingConnections() {
	// Poll to connect to remote servers every X time
	connectionInterval := time.Duration(500)
	connectToLostClusters := time.NewTimer(connectionInterval * time.Millisecond)

	for {
		select {
		case <-connectToLostClusters.C:
			connectToLostClusters = time.NewTimer(connectionInterval * time.Millisecond)
			for _, node := range m.getConfiguredNodes() {
				if _, err := m.getActiveNode(node); err != nil {
					fmt.Printf("Connecting to non-connected cluster node: %+v\n", node)
					new, err := m.getConfiguredNode(node)
					go m.connectToNode(new)
					log.Printf("%s NewNode: %+v %s", m.name, new, err)

				}
			}
		}
	}
}

func (m *Manager) connectToNode(n Node) {
	if err := n.connect(); err == nil {
		m.writeSocket(n.conn, Auth{AuthKey: m.authKey}) // if we fail we get disconnected, so assume we're good
		n.authorized = true
		go m.handleAuthorizedConnection(&n)
		//m.AddActiveNode(&n)
	}
}

func (m *Manager) handleIncommingConnections() {
	for {
		select {
		case socket := <-m.newSocket:
			node := newNode(socket)
			if node.authClient(m.authKey) {
				node.authorized = true
				go m.handleAuthorizedConnection(node)
			} else {
				socket.Close()
			}
		}
	}
}

func (m *Manager) handleAuthorizedConnection(node *Node) {
	m.nodeJoin <- node.name
	m.AddActiveNode(node)
	node.ioReader(m.packetManager)
	m.RemoveActiveNode(node)
	m.nodeLeave <- node.name
}

func (m *Manager) handlePackets() {
	for {
		select {
		case message := <-m.ToCluster:
			fmt.Printf("received message to send to cluster:%+v\n", message)
			m.writeCluster(message)
		case packet := <-m.packetManager:
			switch packet.DataType {
			case "signals.Auth":
			default:
				fmt.Printf("Recieved non-cluster packet: %+v\n", packet)
				m.FromCluster <- packet
			}
			fmt.Printf("packet: %+v\n", packet)
		}
	}
}

func (m *Manager) writeCluster(dataMessage interface{}) {
	nodes := m.getActiveNodes()
	for _, node := range nodes {
		m.writeClusterNode(node, dataMessage)
	}
}

func (m *Manager) writeClusterNode(node string, dataMessage interface{}) {
	if n, err := m.getActiveNode(node); err == nil {
		m.writeSocket(n.conn, dataMessage)
	}
}

func (m *Manager) writeSocket(conn net.Conn, dataMessage interface{}) {
	val := reflect.Indirect(reflect.ValueOf(dataMessage))
	packet := &Packet{
		Name:     m.name,
		DataType: fmt.Sprintf("%s", val.Type()),
		//DataMessage: dataMessage,
		Time: time.Now(),
	}
	data, err := json.Marshal(dataMessage)
	if err != nil {
		fmt.Printf("Unable to jsonfy data: %+v", data)
	}
	packet.DataMessage = string(data)

	fmt.Printf("Packet to Write: %+v\n", packet)

	packetData, err := json.Marshal(packet)
	if err != nil {
		fmt.Println("Unable to create json packet")
	}

	packetData = append(packetData, 10) // 10 = newline
	_, err = conn.Write(packetData)
	if err != nil {
		fmt.Println("Failed to send data to client")
		/*
			client.Err = err
			cluster.Err <- client
			return err
		*/
	}
}
