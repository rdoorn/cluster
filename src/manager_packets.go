package cluster

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

func (m *Manager) handlePackets() {
	for {
		select {
		case pm := <-m.ToNode: // incomming from client application
			err := m.writeClusterNode(pm.Node, pm.Message)
			if err != nil {
				m.log("Failed to write message to remote node. error: %s", err)
			}
		case message := <-m.ToCluster: // incomming from client application
			err := m.writeCluster(message)
			if err != nil {
				m.log("Failed to write message to remote node. error: %s", err)
			}
		case message := <-m.internalMessage: // incomming intenal messages (do not leave this library)
			switch message.Type {
			case "nodejoin":
				m.log("Cluster node joined: %s", message.Node)
				select {
				case m.NodeJoin <- message.Node: // send node join to client application
				default:
				}
				m.log("Cluster quorum state: %t", m.quorum())
				select {
				case m.QuorumState <- m.quorum(): // quorum update to client application
				default:
				}
			case "nodeleave":
				m.log("Cluster node left: %s (%s)", message.Node, message.Error)
				select {
				case m.NodeLeave <- message.Node: // send node leave to client application
				default:
				}
				m.log("Cluster quorum state: %t", m.quorum())
				select {
				case m.QuorumState <- m.quorum(): // quorum update to client application
				default:
				}
			}
		case packet := <-m.incommingPackets: // incomming packets from other cluster nodes
			switch packet.DataType {
			case "cluster.Auth": // internal use
			case "cluster.NodeShutdownPacket": // internal use
				m.log("Got exit notice from node %s (shutdown)", packet.Name)
				m.connectedNodes.close(packet.Name)
			case "cluster.PingPacket": // internal use
				m.log("Got ping from node %s (%v)", packet.Name, time.Now().Sub(packet.Time))
				m.connectedNodes.setLag(packet.Name, time.Now().Sub(packet.Time))
			default:
				m.log("Recieved non-cluster packet: %s", packet.DataType)
				m.FromCluster <- packet // outgoing to client application
			}
		}
	}
}

func (m *Manager) newPacket(dataMessage interface{}) ([]byte, error) {
	val := reflect.Indirect(reflect.ValueOf(dataMessage))
	packet := &Packet{
		Name:     m.name,
		DataType: fmt.Sprintf("%s", val.Type()),
		Time:     time.Now(),
	}
	data, err := json.Marshal(dataMessage)
	if err != nil {
		m.log("Unable to jsonfy data: %s", err)
	}
	packet.DataMessage = string(data)

	packetData, err := json.Marshal(packet)
	if err != nil {
		m.log("Unable to create json packet: %s", err)
	}

	packetData = append(packetData, 10) // 10 = newline
	return packetData, err
}
