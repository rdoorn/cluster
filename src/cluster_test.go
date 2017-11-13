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

func TestConfig(t *testing.T) {
	t.Logf("TestConfig...")

	// Manager A
	managerA := NewManager("managerA", "secret")
	err := managerA.ListenAndServe("127.0.0.1:9500")
	if err != nil {
		log.Fatal(err)
	}
	managerA.AddClusterNode(Node{name: "managerB", addr: "127.0.0.1:9501"})

	// Manager B
	time.Sleep(200 * time.Millisecond)
	managerB := NewManager("managerB", "secret")
	err = managerB.ListenAndServe("127.0.0.1:9501")
	if err != nil {
		log.Fatal(err)
	}
	managerB.AddClusterNode(Node{name: "managerA", addr: "127.0.0.1:9500"})

	timeout := time.NewTimer(5 * time.Second)

loop:
	for {
		select {
		case _ = <-timeout.C:
			fmt.Printf("Exiting Loop...\n")
			fmt.Printf("ManagerA has %d Connections on exit\n", managerA.getConnectionCount())
			fmt.Printf("ManagerB has %d Connections on exit\n", managerB.getConnectionCount())
			break loop
		case nodeJoined := <-managerA.nodeJoin:
			fmt.Printf("Node joined managerA: %s\n", nodeJoined)
			managerA.ToCluster <- Message{Message: fmt.Sprintf("Welcome to managerA sir %s", nodeJoined)}
		case nodeJoined := <-managerB.nodeJoin:
			fmt.Printf("Node joined managerB: %s\n", nodeJoined)
			managerB.ToCluster <- Message{Message: fmt.Sprintf("Welcome to managerB sir %s", nodeJoined)}
		case nodeLeft := <-managerA.nodeLeave:
			fmt.Printf("Node left managerA: %s\n", nodeLeft)
		case nodeLeft := <-managerB.nodeLeave:
			fmt.Printf("Node left managerB: %s\n", nodeLeft)
		case message := <-managerA.FromCluster:
			fmt.Printf("ManagerA recieved the following from the cluster: %s\n", message)
		case message := <-managerB.FromCluster:
			fmt.Printf("ManagerB recieved the following from the cluster: %s\n", message)
		}
	}

	fmt.Printf("Exiting Test!\n")
	managerA.Shutdown()
	time.Sleep(5 * time.Second)

	managerB.Shutdown()

}
