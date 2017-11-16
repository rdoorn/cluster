package cluster

import (
	"net/http"
	"time"
)

type apiClusterPublicHandler struct {
	manager *Manager
}

// APIClusterNode contains details of a node we might connect to
type APIClusterNode struct {
	Name     string        `json:"name"`
	Addr     string        `json:"addr"`
	Status   string        `json:"status"`
	JoinTime time.Time     `json:"jointime"`
	Lag      time.Duration `json:"lag"`
	Packets  int64         `json:"packets"`
}

// APIClusterNodeList contains a list of configured/connected nodes
type APIClusterNodeList struct {
	Nodes map[string]APIClusterNode `json:"nodes"`
}

func (h apiClusterPublicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.manager.RLock()
	defer h.manager.RLock()
	var message = &APIClusterNodeList{
		Nodes: make(map[string]APIClusterNode),
	}
	for _, configured := range h.manager.configuredNodes {

		n := APIClusterNode{
			Name: configured.name,
			Addr: configured.addr,
		}

		active := findActiveNode(h.manager.connectedNodes.nodes, configured.name)
		if active != nil {
			n.JoinTime = active.joinTime
			n.Lag = active.lag
			n.Packets = active.packets
		}
		message.Nodes[configured.name] = n
	}
	apiWriteData(w, http.StatusOK, apiMessage{Success: true, Data: message})
}

func findActiveNode(n []*Node, name string) *Node {
	for _, node := range n {
		if node.name == name {
			return node
		}
	}
	return nil

}
