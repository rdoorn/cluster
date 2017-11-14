package signals

import (
	"log"
	"time"
)

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

func (m *Manager) updateSettings(settings Settings) {
	m.Lock()
	defer m.Unlock()
	m.settings = settings
}

func (m *Manager) getDuration(setting string) time.Duration {
	m.RLock()
	defer m.RUnlock()
	switch setting {
	case "pinginterval":
		return m.settings.PingInterval
	case "joindelay":
		return m.settings.JoinDelay
	case "connecttimeout":
		return m.settings.ConnectTimeout
	case "connectinterval":
		return m.settings.ConnectInterval
	case "readtimeout":
		return m.settings.ReadTimeout
	default:
		log.Fatalf("Unknown setting: %s", setting)
		return 0
	}
}
