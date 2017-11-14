package signals

import (
	"fmt"
	"log"
	"testing"
	"time"
)

type Message struct {
	Message string `json:"message"`
}

/*
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
		if timeout := channelReadPacketTimeout(managerONE.FromCluster, 3); !timeout {
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

	if _, timeout := channelReadString(managerONE.nodeLeave, 3); !timeout {
		t.Errorf("Read from cluster manager.nodeLeave should timeout (we don't send to self). but we received data instead")
	}

}
*/

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
	for _, log := range logs {
		fmt.Println("== LOG: ", log)
	}

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

func channelWriteTimeout(channel chan interface{}, message interface{}, timeout time.Duration) bool {
	select {
	case channel <- message:
		return false // write successfull
	case <-time.After(timeout * time.Second):
		return true // we were blocked
	}
}

func channelReadPacketTimeout(channel chan Packet, timeout time.Duration) bool {
	for {
		select {
		case _ = <-channel:
			return false // read successfull
		case <-time.After(timeout * time.Second):
			return true // read was blocked
		}
	}
}

func channelReadPacket(channel chan Packet, timeout time.Duration) (Packet, bool) {
	fmt.Printf("==== reading from channel packet with timeout %s (%v)\n", timeout*time.Second, time.Now())
	for {
		select {
		case p := <-channel:
			fmt.Printf("==== reading from channel packet returned %s\n", p)
			return p, false // read successfull
		case <-time.After(timeout * time.Second):
			fmt.Printf("==== reading from channel packet timedout (%v)\n", time.Now())
			return Packet{}, true // read was blocked
		}
	}
}

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
