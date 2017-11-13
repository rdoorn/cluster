package signals

import (
	"fmt"
)

func (m *Manager) handleIncommingConnections() {
	for {
		select {
		case conn := <-m.newSocket:
			packet, err := m.connectedNodes.readSocket(conn)
			if err != nil {
				fmt.Printf("handleIncomminConnections readSocket error: %s\n", err)
				conn.Close()
				return
			}
			authRequest := &AuthRequestPacket{}
			err = packet.Message(authRequest)
			if err != nil {
				// invalid message at this time
				authRequest, _ := m.newPacket(AuthResponsePacket{Status: true, Error: err.Error()})
				werr := m.connectedNodes.writeSocket(conn, authRequest)
				if werr != nil {
					fmt.Printf("handleIncomminConnections authrequest write error: %s\n", err)
				}
				fmt.Printf("handleIncomminConnections authrequest error: %s\n", err)
				conn.Close()
				return
			}
			if authRequest.AuthKey != m.authKey {
				// auth failed
				authRequest, _ := m.newPacket(AuthResponsePacket{Status: true, Error: "invalid authentication key"})
				werr := m.connectedNodes.writeSocket(conn, authRequest)
				if werr != nil {
					fmt.Printf("handleIncomminConnections authrequest write2 error: %s\n", err)
				}
				fmt.Printf("handleIncomminConnections authkey error\n")
				conn.Close()
				return
			}
			authResponse, _ := m.newPacket(AuthResponsePacket{Status: true})
			err = m.connectedNodes.writeSocket(conn, authResponse)
			if err != nil {
				fmt.Printf("handleIncomminConnections authresponse error\n")
				conn.Close()
				return
			}

			node := newNode(packet.Name, conn)
			go m.handleAuthorizedConnection(node)
		}
	}
}

func (m *Manager) handleAuthorizedConnection(node *Node) {
	// add authorized node if its uniq
	err := m.connectedNodes.nodeAdd(node)
	if err != nil {
		fmt.Printf("handleAuthorizedConnection nodeAdd error: %s\n", err)
		node.conn.Close()
		return
	}
	// start pinger in the background
	go m.pinger(node.conn, node.quit)

	fmt.Printf("handleAuth: cluster:%s added node:%s\n", m.name, node.name)
	select {
	case m.nodeJoin <- node.name:
	default:
	}
	node.ioReader(m.packetManager, node.quit)
	select {

	case m.nodeLeave <- node.name:
	default:
	}
	m.connectedNodes.nodeRemove(node)
	close(node.quit)
}
