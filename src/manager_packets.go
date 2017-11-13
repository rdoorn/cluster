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
		case message := <-m.ToCluster:
			//fmt.Printf("received message to send to cluster:%+v\n", message)
			err := m.writeCluster(message)
			if err != nil {
				fmt.Printf("Failed to write message to remote node. message: %+v error: %s\n", message, err)
			}
		case packet := <-m.packetManager:
			switch packet.DataType {
			case "signals.Auth":
			case "signals.NodeExitPacket":
				fmt.Printf("%s Got exit notice from node %s\n", m.name, packet.Name)
				m.connectedNodes.close(packet.Name)
			case "signals.PingPacket":
				fmt.Printf("%s Got ping from node %s\n", m.name, packet.Name)
			default:
				fmt.Printf("%s Recieved non-cluster packet: %+v\n", m.name, packet)
				m.FromCluster <- packet
			}
			//fmt.Printf("packet: %+v\n", packet)
		}
	}
}

func (m *Manager) newPacket(dataMessage interface{}) ([]byte, error) {
	val := reflect.Indirect(reflect.ValueOf(dataMessage))
	packet := &Packet{
		Name:     m.name,
		DataType: fmt.Sprintf("%s", val.Type()),
		//DataMessage: dataMessage,
		Time: time.Now(),
	}
	data, err := json.Marshal(dataMessage)
	if err != nil {
		fmt.Printf("Unable to jsonfy data: %+v", data)
	}
	packet.DataMessage = string(data)

	packetData, err := json.Marshal(packet)
	if err != nil {
		fmt.Println("Unable to create json packet")
	}

	packetData = append(packetData, 10) // 10 = newline
	return packetData, err
}
