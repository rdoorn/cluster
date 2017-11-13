package signals

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Manager is the main cluster manager
type Manager struct {
	sync.RWMutex
	name            string           // name of our cluster node
	configuredNodes map[string]Node  // details of the remote cluster nodes
	connectedNodes  connectionPool   // the list of connected nodes and their sockets
	listener        net.Listener     // our listener
	newSocket       chan net.Conn    // new clients connecting
	nodeJoin        chan string      // returns string of the node joining
	nodeLeave       chan string      // returns string of the node leaving
	packetManager   chan Packet      // packets sent to packet manager
	FromCluster     chan Packet      // data received from cluster
	ToCluster       chan interface{} // data send to cluster
	logger          chan string      // logging messages go here
	authKey         string           // authentication key
	quit            chan bool        // signals exit of listener
}

// NewManager creates a new cluster manager
func NewManager(name, key string) *Manager {
	fmt.Println("Starting new Manager")
	m := &Manager{
		name:            name,
		configuredNodes: make(map[string]Node),
		newSocket:       make(chan net.Conn),
		nodeJoin:        make(chan string),
		nodeLeave:       make(chan string),
		packetManager:   make(chan Packet),
		FromCluster:     make(chan Packet),
		ToCluster:       make(chan interface{}),
		logger:          make(chan string, 100),
		authKey:         key,
		quit:            make(chan bool),
	}
	return m
}

// ListenAndServe starts the listener and serves connections to clients
func (m *Manager) ListenAndServe(addr string) (err error) {

	fmt.Printf("%s Starting listener\n", m.name)
	s := Server{addr: addr}
	m.listener, err = s.Listen()
	if err == nil {
		go m.handleIncommingConnections() // handles incommin socket connections
		go m.handleOutgoingConnections()  // creates connections to remote nodes
		go m.handlePackets()              // handles all incomming packets
		go s.Serve(m.newSocket, m.quit)   // accepts new connections and passes them on to the manager
	}
	return
}

// Shutdown stops the cluster node
func (m *Manager) Shutdown() {
	fmt.Printf("%s Stopping listener\n", m.name)
	// write exit message to remote cluster
	packet, _ := m.newPacket(&NodeExitPacket{})
	m.connectedNodes.writeAll(packet)
	fmt.Printf("Sent to all the exit: %+v", string(packet))
	// close all connected nodes
	time.Sleep(5 * time.Second)
	m.connectedNodes.closeAll()
	close(m.quit)
	m.listener.Close()
}

func (m *Manager) getConnectionCount() int {
	return m.connectedNodes.getConnectCount()
}
