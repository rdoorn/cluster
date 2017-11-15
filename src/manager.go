package cluster

import (
	"net"
	"sync"
)

// InternalMessage is used for internal communication within the cluster
type InternalMessage struct {
	Type  string `json:"type"`
	Node  string `json:"node"`
	Error string `json:"error"`
}

// Manager is the main cluster manager
type Manager struct {
	sync.RWMutex
	name             string               // name of our cluster node
	authKey          string               // authentication key
	settings         Settings             // adjustable settings
	listener         net.Listener         // our listener
	connectedNodes   connectionPool       // the list of connected nodes and their sockets
	configuredNodes  map[string]Node      // details of the remote cluster nodes
	newSocket        chan net.Conn        // new clients connecting
	internalMessage  chan InternalMessage // internally sent messages within the cluster
	incommingPackets chan Packet          // packets sent to packet manager
	quit             chan bool            // signals exit of listener
	FromCluster      chan Packet          // data received from cluster
	ToCluster        chan interface{}     // data send to cluster
	ToNode           chan PM              // data send to specific node
	Log              chan string          // logging messages go here
	NodeJoin         chan string          // returns string of the node joining
	NodeLeave        chan string          // returns string of the node leaving
	QuorumState      chan bool            // returns the current quorum state
}

// PM is used for sending private messages between cluster
type PM struct {
	Node    string      // node to send message to
	Message interface{} // message to send to node
}

// NewManager creates a new cluster manager
func NewManager(name, key string) *Manager {
	m := &Manager{
		name:             name,
		authKey:          key,
		settings:         defaultSetting(),
		configuredNodes:  make(map[string]Node),
		newSocket:        make(chan net.Conn),
		internalMessage:  make(chan InternalMessage),
		incommingPackets: make(chan Packet),
		quit:             make(chan bool),
		FromCluster:      make(chan Packet),
		ToCluster:        make(chan interface{}),
		ToNode:           make(chan PM),
		Log:              make(chan string, 500),
		NodeJoin:         make(chan string, 10),
		NodeLeave:        make(chan string, 10),
		QuorumState:      make(chan bool, 10),
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
	m.log("Cluster quorum state: %t", m.quorum())
	select {
	case m.QuorumState <- m.quorum(): // quorum update to client application
	default:
	}
	return
}

// Shutdown stops the cluster node
func (m *Manager) Shutdown() {
	m.log("Stopping listener on %s", m.listener.Addr())
	// write exit message to remote cluster
	packet, _ := m.newPacket(&NodeShutdownPacket{})
	m.connectedNodes.writeAll(packet)
	// close all connected nodes
	m.connectedNodes.closeAll()
	close(m.quit)
	m.listener.Close()
}

// quorum returns quorum state based on configured vs connected nodes
func (m *Manager) quorum() bool {
	m.RLock()
	defer m.RUnlock()
	switch len(m.configuredNodes) {
	case 0:
		return true // single node
	case 1:
		return true // 2 cluster node, we don't send quorum loss, as that would nullify the additional node
	default:
		return float64(len(m.configuredNodes)+1)/2 < float64(len(m.connectedNodes.nodes)+1) // +1 to add our selves
	}
}
