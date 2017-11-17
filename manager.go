package cluster

import (
	"net"
	"net/http"
	"sync"
)

// InternalMessage is used for internal communication within the cluster
type internalMessage struct {
	Type  string `json:"type"`
	Node  string `json:"node"`
	Error string `json:"error"`
}

// NodeMessage is used for sending private messages between cluster nodes
type NodeMessage struct {
	Node    string      // node to send message to
	Message interface{} // message to send to node
}

// Manager is the main cluster manager
type Manager struct {
	sync.RWMutex
	name             string               // name of our cluster node
	authKey          string               // authentication key
	settings         Settings             // adjustable settings
	listener         net.Listener         // our listener
	connectedNodes   *connectionPool      // the list of connected nodes and their sockets
	configuredNodes  map[string]Node      // details of the remote cluster nodes
	newSocket        chan net.Conn        // new clients connecting
	internalMessage  chan internalMessage // internally sent messages within the cluster
	apiRequest       chan APIRequest      // API sent messages to the cluster from the API
	incommingPackets chan Packet          // packets sent to packet manager
	quit             chan bool            // signals exit of listener
	FromCluster      chan Packet          // data received from cluster
	FromClusterAPI   chan APIRequest      // data received from cluster via API interface
	ToCluster        chan interface{}     // data send to cluster
	ToNode           chan NodeMessage     // data send to specific node
	Log              chan string          // logging messages go here
	NodeJoin         chan string          // returns string of the node joining
	NodeLeave        chan string          // returns string of the node leaving
	QuorumState      chan bool            // returns the current quorum state
}

var managers = struct {
	sync.RWMutex
	manager       []string
	clusterAPISet bool
}{}

// NewManager creates a new cluster manager
func NewManager(name, authKey string) *Manager {
	m := &Manager{
		name:             name,
		authKey:          authKey,
		settings:         defaultSetting(),
		configuredNodes:  make(map[string]Node),
		connectedNodes:   newConnectionPool(),
		newSocket:        make(chan net.Conn),
		internalMessage:  make(chan internalMessage),
		apiRequest:       make(chan APIRequest),
		incommingPackets: make(chan Packet),
		quit:             make(chan bool),
		FromCluster:      make(chan Packet),
		FromClusterAPI:   make(chan APIRequest, 10),
		ToCluster:        make(chan interface{}),
		ToNode:           make(chan NodeMessage),
		Log:              make(chan string, 500),
		NodeJoin:         make(chan string, 10),
		NodeLeave:        make(chan string, 10),
		QuorumState:      make(chan bool, 10),
	}
	addManager(m.name)
	if APIEnabled {
		m.addClusterAPI()
	}
	return m
}

func addManager(name string) {
	managers.Lock()
	defer managers.Unlock()
	managers.manager = append(managers.manager, name)
}

func removeManager(name string) {
	managers.Lock()
	defer managers.Unlock()
	var new []string
	for _, mgr := range managers.manager {
		if mgr != name {
			new = append(new, mgr)
		}
	}
	managers.manager = new
}

func (m *Manager) addClusterAPI() {
	managers.Lock()
	defer managers.Unlock()

	http.Handle("/api/cluster/"+m.name+"/admin/", authenticate(apiClusterAdminHandler{manager: m}, m.authKey))
	http.Handle("/api/cluster/"+m.name, apiClusterPublicHandler{manager: m})
	if managers.clusterAPISet == false {
		http.Handle("/api/cluster", apiClusterHandler{})
		managers.clusterAPISet = true
	}
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
	packet, _ := m.newPacket(&packetNodeShutdown{})
	m.connectedNodes.writeAll(packet)
	// close all connected nodes
	m.connectedNodes.closeAll()
	close(m.quit)
	m.listener.Close()
	removeManager(m.name)
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

func (m *Manager) updateQuorum() {
	m.log("Cluster quorum state: %t", m.quorum())
	select {
	case m.QuorumState <- m.quorum(): // quorum update to client application
	default:
	}
}

// AddClusterNode adds a cluster node to the cluster to be connected to
func (m *Manager) AddClusterNode(n Node) {
	m.Lock()
	defer m.Unlock()
	n.statusStr = StatusNew
	m.configuredNodes[n.name] = n
	select {
	case m.internalMessage <- internalMessage{Type: "nodeadd", Node: n.name}:
	default:
	}
}

// RemoveClusterNode remove a cluster node from the list of servers to connect to, and close its connections
func (m *Manager) RemoveClusterNode(n Node) {
	m.Lock()
	defer m.Unlock()
	m.log("%s is removing node %s", m.name, n.name)
	if _, ok := m.configuredNodes[n.name]; ok {
		delete(m.configuredNodes, n.name)
	}
	select {
	case m.internalMessage <- internalMessage{Type: "noderemove", Node: n.name}:
	default:
	}
	m.connectedNodes.close(n.name)
}

func (m *Manager) getConfiguredNodes() (nodes []Node) {
	m.RLock()
	defer m.RUnlock()
	for _, node := range m.configuredNodes {
		nodes = append(nodes, node)
	}
	return
}
