package signals

import (
	"log"
	"sync"
	"testing"
	"time"
)

type Message struct {
	Message string `json:"message"`
}

func TestOneClusterNode(t *testing.T) {
	t.Parallel()

	managerONE := NewManager("managerONE", "secret")
	err := managerONE.ListenAndServe("127.0.0.1:9501")
	if err != nil {
		log.Fatal(err)
	}

	managerONE.ToCluster <- Message{Message: "Hello World"}
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		if _, timeout := channelReadPacket(managerONE.FromCluster, 1); !timeout {
			t.Errorf("Read from cluster manager.FromCluster should timeout (we don't send to self). but we received data instead")
		}
	}()
	go func() {
		defer wg.Done()
		if timeout := channelWriteTimeout(managerONE.ToCluster, Message{Message: "Hello World"}, 1); timeout {
			t.Errorf("Write to managerTWO.ToCluster should not timeout. but we were unable to send data to it")
		}
	}()
	wg.Wait()

	managerONE.Shutdown()

	if _, timeout := channelReadString(managerONE.nodeLeave, 1); !timeout {
		t.Errorf("Read from cluster manager.nodeLeave should timeout (we don't send to self). but we received data instead")
	}

}

func TestTwoClusterNode(t *testing.T) {
	t.Parallel()
	// Manager A
	managerTWO := NewManager("managerTWO", "secret")
	err := managerTWO.ListenAndServe("127.0.0.1:9502")
	if err != nil {
		log.Fatal(err)
	}
	managerTWO.AddClusterNode(Node{name: "managerTHREE", addr: "127.0.0.1:9503"})

	// Manager B
	//time.Sleep(200 * time.Millisecond)
	managerTHREE := NewManager("managerTHREE", "secret")
	err = managerTHREE.ListenAndServe("127.0.0.1:9503")
	if err != nil {
		log.Fatal(err)
	}
	managerTHREE.AddClusterNode(Node{name: "managerTWO", addr: "127.0.0.1:9502"})

	node, timeout := channelReadString(managerTWO.nodeJoin, 5)
	if timeout {
		t.Errorf("expected Join on managerTWO, but got timeout")
	}
	if node != "managerTHREE" {
		t.Errorf("expected Join on managerTWO to be from managerTHREE, but got:%s", node)
	}

	node, timeout = channelReadString(managerTHREE.nodeJoin, 5)
	if timeout {
		t.Errorf("expected Join on managerTHREE, but got timeout")
	}
	if node != "managerTWO" {
		t.Errorf("expected Join on managerTHREE to be from managerTWO, but got:%s", node)
	}

	if timeout = channelWriteTimeout(managerTWO.ToCluster, Message{Message: "Hello World"}, 2); timeout {
		t.Errorf("expected write to managerTWO.ToCluster to work, but it timedout")
	}

	packet, timeout := channelReadPacket(managerTHREE.FromCluster, 5)
	if timeout {
		t.Errorf("expected data FromCluster on managerTHREE, but got timeout")
	} else {
		msg := &Message{}
		err := packet.Message(msg)
		if err != nil {
			t.Errorf("unable to unpack the message received from managerTHREE.FromCluster error:%s", err)
		} else if msg.Message != "Hello World" {
			t.Errorf("expected managerTHREE.FromCluster to return 'Hello World' but got:%s", msg)
		}
	}

	if timeout = channelWriteTimeout(managerTHREE.ToCluster, Message{Message: "Hello World"}, 2); timeout {
		t.Errorf("expected write to managerTHREE.ToCluster to work, but it timedout")
	}

	logs := channelReadStrings(managerTWO.Log, 1)
	if len(logs) == 0 {
		t.Errorf("expected log output for managerTWO, but got nothing")
	}
	/*
		for _, log := range logs {
			fmt.Println("== LOG: ", log)
		}*/

	managerTWO.Shutdown()

	node, timeout = channelReadString(managerTHREE.nodeLeave, 2)
	if timeout {
		t.Errorf("expected Leave on managerTHREE, but got timeout")
	}
	if node != "managerTWO" {
		t.Errorf("expected Leave on managerTHREE to be from managerTWO, but got:%s", node)
	}

	managerTHREE.Shutdown()

}

func TestTreeNodeCluster(t *testing.T) {
	t.Parallel()
	// Manager 4
	managerFOUR := NewManager("managerFOUR", "secret")
	err := managerFOUR.ListenAndServe("127.0.0.1:9504")
	if err != nil {
		log.Fatal(err)
	}
	managerFOUR.AddClusterNode(Node{name: "managerFIVE", addr: "127.0.0.1:9505"})
	managerFOUR.AddClusterNode(Node{name: "managerSIX", addr: "127.0.0.1:9506"})

	// Manager 5
	managerFIVE := NewManager("managerFIVE", "secret")
	err = managerFIVE.ListenAndServe("127.0.0.1:9505")
	if err != nil {
		log.Fatal(err)
	}
	managerFIVE.AddClusterNode(Node{name: "managerFOUR", addr: "127.0.0.1:9504"})
	managerFIVE.AddClusterNode(Node{name: "managerSIX", addr: "127.0.0.1:9506"})

	// Manager 6
	managerSIX := NewManager("managerSIX", "secret")
	err = managerSIX.ListenAndServe("127.0.0.1:9506")
	if err != nil {
		log.Fatal(err)
	}
	managerSIX.AddClusterNode(Node{name: "managerFOUR", addr: "127.0.0.1:9504"})
	managerSIX.AddClusterNode(Node{name: "managerFIVE", addr: "127.0.0.1:9505"})

	// joins on manager4
	for a := 0; a <= 1; a++ {
		node, timeout := channelReadString(managerFOUR.nodeJoin, 5)
		if timeout {
			t.Errorf("expected Join on managerFOUR, but got timeout (loop:%d)", a)
		}
		if node != "managerFIVE" && node != "managerSIX" {
			t.Errorf("expected Join on managerFOUR to be from managerFIVE or managerSIX, but got:%s (loop:%d)", node, a)
		}
	}

	// joins on manager5
	for a := 0; a <= 1; a++ {
		node, timeout := channelReadString(managerFIVE.nodeJoin, 5)
		if timeout {
			t.Errorf("expected Join on managerFIVE, but got timeout (loop:%d)", a)
		}
		if node != "managerFOUR" && node != "managerSIX" {
			t.Errorf("expected Join on managerFIVE to be from managerFOUR or managerSIX, but got:%s (loop:%d)", node, a)
		}
	}

	// joins on manager6
	for a := 0; a <= 1; a++ {
		node, timeout := channelReadString(managerSIX.nodeJoin, 5)
		if timeout {
			t.Errorf("expected Join on managerSIX, but got timeout (loop:%d)", a)
		}
		if node != "managerFIVE" && node != "managerFOUR" {
			t.Errorf("expected Join on managerSIX to be from managerFIVE or managerFOUR, but got:%s (loop:%d)", node, a)
		}
	}

	// send hello to cluster
	if timeout := channelWriteTimeout(managerFOUR.ToCluster, Message{Message: "Hello World"}, 2); timeout {
		t.Errorf("expected write to managerFOUR.ToCluster to work, but it timedout")
	}

	// read hello from cluster node 5
	packet, timeout := channelReadPacket(managerFIVE.FromCluster, 5)
	if timeout {
		t.Errorf("expected data FromCluster on managerFIVE, but got timeout")
	} else {
		msg := &Message{}
		err := packet.Message(msg)
		if err != nil {
			t.Errorf("unable to unpack the message received from managerFIVE.FromCluster error:%s", err)
		} else if msg.Message != "Hello World" {
			t.Errorf("expected managerFIVE.FromCluster to return 'Hello World' but got:%s", msg)
		}
	}

	// read hello from cluster node 6
	packet, timeout = channelReadPacket(managerSIX.FromCluster, 5)
	if timeout {
		t.Errorf("expected data FromCluster on managerSIX, but got timeout")
	} else {
		msg := &Message{}
		err := packet.Message(msg)
		if err != nil {
			t.Errorf("unable to unpack the message received from managerSIX.FromCluster error:%s", err)
		} else if msg.Message != "Hello World" {
			t.Errorf("expected managerSIX.FromCluster to return 'Hello World' but got:%s", msg)
		}
	}

	// write hello to node 4
	if timeout = channelWriteTimeoutPM(managerSIX.ToNode, PM{Node: "managerFOUR", Message: Message{Message: "Hello managerFOUR"}}, 2); timeout {
		t.Errorf("expected write to managerSIX.ToNode to work, but it timedout")
	}

	// read hello from cluster node 4
	packet, timeout = channelReadPacket(managerFOUR.FromCluster, 5)
	if timeout {
		t.Errorf("expected data FromCluster on managerFOUR, but got timeout")
	} else {
		msg := &Message{}
		err := packet.Message(msg)
		if err != nil {
			t.Errorf("unable to unpack the message received from managerFOUR.FromCluster error:%s", err)
		} else if msg.Message != "Hello managerFOUR" {
			t.Errorf("expected managerFOUR.FromCluster to return 'Hello managerFOUR' but got:%s", msg)
		}
	}

	managerSIX.Shutdown()

	node, timeout := channelReadString(managerFOUR.nodeLeave, 2)
	if timeout {
		t.Errorf("expected Leave on managerFOUR, but got timeout")
	}
	if node != "managerSIX" {
		t.Errorf("expected Leave on managerFOUR to be from managerSIX, but got:%s", node)
	}

	node, timeout = channelReadString(managerFIVE.nodeLeave, 2)
	if timeout {
		t.Errorf("expected Leave on managerFIVE, but got timeout")
	}
	if node != "managerSIX" {
		t.Errorf("expected Leave on managerFIVE to be from managerSIX, but got:%s", node)
	}

}

// channelWriteTimeout writes a message to a channel, or will timeout if failed
func channelWriteTimeout(channel chan interface{}, message interface{}, timeout time.Duration) bool {
	select {
	case channel <- message:
		return false // write successfull
	case <-time.After(timeout * time.Second):
		return true // we were blocked
	}
}

// channelWriteTimeoutPM writes a Private Message to a channel, or will timeout if failed
func channelWriteTimeoutPM(channel chan PM, message PM, timeout time.Duration) bool {
	select {
	case channel <- message:
		return false // write successfull
	case <-time.After(timeout * time.Second):
		return true // we were blocked
	}
}

// channelReadPacket reads 1 packet from the channel or times out after timeout
func channelReadPacket(channel chan Packet, timeout time.Duration) (Packet, bool) {
	for {
		select {
		case p := <-channel:
			return p, false // read successfull
		case <-time.After(timeout * time.Second):
			return Packet{}, true // read was blocked
		}
	}
}

// channelReadString reads 1 string from the channel or times out after timeout
func channelReadString(channel chan string, timeout time.Duration) (string, bool) {
	for {
		select {
		case result := <-channel:
			return result, false // read successfull
		case <-time.After(timeout * time.Second):
			return "", true // read was blocked
		}
	}
}

// channelReadStrings reads a array of strings for the duration of timeout
func channelReadStrings(channel chan string, timeout time.Duration) (results []string) {
	for {
		select {
		case result := <-channel:
			results = append(results, result)
		case <-time.After(timeout * time.Second):
			return
		}
	}
}
