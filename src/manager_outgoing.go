package cluster

import (
	"net"
	"time"
)

func (m *Manager) handleOutgoingConnections() {
	for {
		select {
		case <-m.quit:
			// if manager exists, stop making outgoing connections
			m.log("EXIT of manager for outgoing connections")
			return
		default:
		}

		// Attempt to connect to non-connected nodes
		for _, node := range m.getConfiguredNodes() {
			if !m.connectedNodes.nodeExists(node.name) {
				// Connect to the remote cluster node
				m.log("Connecting to non-connected cluster node: %s", node.name)
				m.dial(node.name, node.addr)
			}
		}
		//w ait before we try again
		time.Sleep(m.getDuration("connectinterval"))
	}
}

func (m *Manager) dial(name, addr string) {
	m.log("Connecting to %s (%s)", name, addr)
	conn, err := net.DialTimeout("tcp", addr, m.getDuration("connecttimeout"))
	if err == nil {
		// on dialing out, we need to send an auth
		authRequest, _ := m.newPacket(AuthRequestPacket{AuthKey: m.authKey})
		m.connectedNodes.writeSocket(conn, authRequest)
		packet, err := m.connectedNodes.readSocket(conn)
		if err != nil {
			// close connection if someone is talking gibrish
			conn.Close()
		}
		authResponse := &AuthResponsePacket{}
		err = packet.Message(authResponse)
		if err != nil {
			// auth response unknown
			conn.Close()
		}
		if authResponse.Status != true {

			m.log("auth failed on dial: %s", authResponse.Error)
		}

		node := newNode(packet.Name, conn)
		node.joinTime = authResponse.Time

		go m.handleAuthorizedConnection(node)
	}
}
