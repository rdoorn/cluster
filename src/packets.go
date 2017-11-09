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

func (packet *Packet) Message(i interface{}) (interface{}, error) {
	message := i
	err := json.Unmarshal(json.RawMessage(packet.DataMessage), &message)
	if err != nil {
		return nil, fmt.Errorf("Failed to decrypt dataMessage:%v", err)
	}
	return message, nil
}

func UnpackPacket(data []byte) (packet *Packet, err error) {
	err = json.Unmarshal(json.RawMessage(data), &packet)
	if err != nil {
		return nil, fmt.Errorf("Failed to decrypt packet header:%v", err)
	}
	return
}
