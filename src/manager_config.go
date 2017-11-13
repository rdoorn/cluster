package signals

// AddClusterNode adds a cluster node to the cluster to be connected to
func (m *Manager) AddClusterNode(n Node) {
	m.Lock()
	defer m.Unlock()
	m.configuredNodes[n.name] = n
}

func (m *Manager) getConfiguredNodes() (nodes []Node) {
	m.RLock()
	defer m.RUnlock()
	for _, node := range m.configuredNodes {
		nodes = append(nodes, node)
	}
	return
}
