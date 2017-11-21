package cluster

import (
	"crypto/tls"
	"net"
	"time"
)

func (m *Manager) handleOutgoingConnections(tlsConfig *tls.Config) {
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
				m.dial(node.name, node.addr, tlsConfig)
			}
		}
		//w ait before we try again
		time.Sleep(m.getDuration("connectinterval"))
	}
}

func (m *Manager) dial(name, addr string, tlsConfig *tls.Config) {
	m.log("Connecting to %s (%s)", name, addr)
	var conn net.Conn
	var err error
	if m.useTLS == false {
		conn, err = net.DialTimeout("tcp", addr, m.getDuration("connecttimeout"))
	} else {
		m.log("DIAL TLS: %+v", time.Now())
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: m.getDuration("connecttimeout")}, "tcp", addr, tlsConfig)
		if err != nil {
			m.log("DIAL TLS END ERROR: %+v %s", time.Now(), err)
		}
		m.log("DIAL TLS END: %+v", time.Now())
	}
	if err == nil {
		// on dialing out, we need to send an auth
		authRequest, _ := m.newPacket(packetAuthRequest{AuthKey: m.authKey})
		m.connectedNodes.writeSocket(conn, authRequest)
		packet, err := m.connectedNodes.readSocket(conn)
		if err != nil {
			// close connection if someone is talking gibrish
			conn.Close()
			return
		}
		authResponse := &packetAuthResponse{}
		err = packet.Message(authResponse)
		if err != nil {
			// auth response unknown
			conn.Close()
			return
		}
		if authResponse.Status != true {
			m.log("auth failed on dial: %s", authResponse.Error)
			conn.Close()
			return
		}

		node := newNode(packet.Name, conn)
		node.joinTime = authResponse.Time

		go m.handleAuthorizedConnection(node)
	}
}
