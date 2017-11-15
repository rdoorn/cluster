package signals

import (
	"net"
	"sync"
)

// Manager is the main cluster manager
type Manager struct {
	sync.RWMutex
	name            string           // name of our cluster node
	configuredNodes map[string]Node  // details of the remote cluster nodes
	settings        Settings         // adjustable settings
	connectedNodes  connectionPool   // the list of connected nodes and their sockets
	listener        net.Listener     // our listener
	newSocket       chan net.Conn    // new clients connecting
	nodeJoin        chan string      // returns string of the node joining
	nodeLeave       chan string      // returns string of the node leaving
	packetManager   chan Packet      // packets sent to packet manager
	FromCluster     chan Packet      // data received from cluster
	ToCluster       chan interface{} // data send to cluster
	ToNode          chan PM          // data send to specific node
	Log             chan string      // logging messages go here
	authKey         string           // authentication key
	quit            chan bool        // signals exit of listener
}

// PM is used for sending private messages between cluster
type PM struct {
	Node    string      // node to send message to
	Message interface{} // message to send to node
}

// NewManager creates a new cluster manager
func NewManager(name, key string) *Manager {
	m := &Manager{
		name:            name,
		configuredNodes: make(map[string]Node),
		newSocket:       make(chan net.Conn),
		nodeJoin:        make(chan string, 10),
		nodeLeave:       make(chan string, 10),
		packetManager:   make(chan Packet),
		FromCluster:     make(chan Packet),
		ToCluster:       make(chan interface{}),
		ToNode:          make(chan PM),
		Log:             make(chan string, 500),
		authKey:         key,
		quit:            make(chan bool),
		settings:        defaultSetting(),
	}
	return m
}

// ListenAndServe starts the listener and serves connections to clients
func (m *Manager) ListenAndServe(addr string) (err error) {
	m.log("Starting listener on %s", addr)
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
	m.log("Stopping listener on %s", m.listener.Addr())
	// write exit message to remote cluster
	packet, _ := m.newPacket(&NodeExitPacket{})
	m.connectedNodes.writeAll(packet)
	// close all connected nodes
	m.connectedNodes.closeAll()
	close(m.quit)
	m.listener.Close()
}
