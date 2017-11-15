package signals

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

func (m *Manager) handlePackets() {
	for {
		select {
		case pm := <-m.ToNode:
			err := m.writeClusterNode(pm.Node, pm.Message)
			if err != nil {
				m.log("Failed to write message to remote node. error: %s", err)
			}
		case message := <-m.ToCluster:
			err := m.writeCluster(message)
			if err != nil {
				m.log("Failed to write message to remote node. error: %s", err)
			}
		case packet := <-m.packetManager:
			switch packet.DataType {
			case "signals.Auth":
			case "signals.NodeExitPacket":
				m.log("Got exit notice from node %s", packet.Name)
				m.connectedNodes.close(packet.Name)
			case "signals.PingPacket":
				m.log("Got ping from node %s", packet.Name)
			default:
				m.log("Recieved non-cluster packet: %+v\n", packet)
				m.FromCluster <- packet
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
