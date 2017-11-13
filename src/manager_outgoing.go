package signals

import (
	"fmt"
	"net"
	"time"
)

func (m *Manager) handleOutgoingConnections() {
	// Poll to connect to remote servers every X time
	connectionInterval := time.Duration(1500)
	connectToLostClusters := time.NewTimer(connectionInterval * time.Millisecond)

	for {
		select {
		case <-connectToLostClusters.C:
			connectToLostClusters = time.NewTimer(connectionInterval * time.Millisecond)
			// loop through all configured nodes, and see if we are not connected to one of them
			for _, node := range m.getConfiguredNodes() {
				if !m.connectedNodes.nodeExists(node.name) {
					// Connect to the remote cluster node
					fmt.Printf("%s Connecting to non-connected cluster node: %+v\n", m.name, node)
					go m.dial(node.name, node.addr)
				}
			}
		}
	}
}

func (m *Manager) dial(name, addr string) {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
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
			fmt.Printf("dial auth failed: %s", authResponse.Error)
		}

		node := newNode(packet.Name, conn)

		m.handleAuthorizedConnection(node)
	}
}

func (m *Manager) writeCluster(dataMessage interface{}) error {
	//nodes := connected.getActiveNodes()
	packet, err := m.newPacket(dataMessage)
	if err != nil {
		return err
	}
	err = m.connectedNodes.writeAll(packet)
	return err

}

func (m *Manager) writeClusterNode(node string, dataMessage interface{}) {
	/*if n, err := m.getActiveNode(node); err == nil {
		m.writeSocket(n.conn, dataMessage)
	}*/
}

func (m *Manager) pinger(conn net.Conn, quit chan bool) {
	//if m.name == "managerA" {
	//		return
	//	}
	for {
		select {
		case <-quit:
			fmt.Printf("Exiting pinger for %s\n", conn.RemoteAddr())
			return
		default:
		}
		p, _ := m.newPacket(&PingPacket{Time: time.Now()})
		fmt.Printf("Sending Ping for %s\n", m.name)
		err := m.connectedNodes.writeSocket(conn, p)
		if err != nil {
			fmt.Printf("Failed to send ping to %s\n", conn.RemoteAddr())
			//close(quit)
			conn.Close()
			return
		}
		time.Sleep(5 * time.Second)

	}
}
