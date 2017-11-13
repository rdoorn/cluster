package signals

import (
	"encoding/json"
	"fmt"
	"time"
)

// Packet is a cluster communication packet
type Packet struct {
	Name        string    `json:"name"`
	DataType    string    `json:"datatype"`
	DataMessage string    `json:"datamessage"`
	Time        time.Time `json:"time"`
	source      string
}

// Some predefined packets //

// AuthRequestPacket defines an authorization request
type AuthRequestPacket struct {
	AuthKey string `json:"authkey"`
}

// AuthResponsePacket defines an authorization response
type AuthResponsePacket struct {
	Status bool   `json:"status"`
	Error  string `json:"error"`
}

// PingPacket defines a ping
type PingPacket struct {
	Time time.Time `json:"time"`
}

// NodeExitPacket defines a node exiting the cluster
type NodeExitPacket struct{}

// Message returns the message of a packet
func (packet *Packet) Message(message interface{}) error {
	//message := i
	err := json.Unmarshal(json.RawMessage(packet.DataMessage), &message)
	if err != nil {
		return fmt.Errorf("Failed to decrypt dataMessage:%v", err)
	}
	return nil
}

// UnpackPacket unpacks a packet and returns its structure
func UnpackPacket(data []byte) (packet *Packet, err error) {
	err = json.Unmarshal(json.RawMessage(data), &packet)
	if err != nil {
		return nil, fmt.Errorf("Failed to decrypt packet header:%v", err)
	}
	return
}
