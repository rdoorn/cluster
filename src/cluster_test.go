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
	managerB := NewManager("managerB", "secret")
	err = managerB.ListenAndServe("127.0.0.1:9501")
	if err != nil {
		log.Fatal(err)
	}
	managerB.AddClusterNode(Node{name: "managerA", addr: "127.0.0.1:9500"})

	timeout := time.NewTimer(2 * time.Second)

	managerA.ToCluster <- Message{Message: "hello world"}

loop:
	for {
		select {
		case _ = <-timeout.C:
			break loop
		case join := <-managerA.nodeJoin:
			fmt.Printf("TST: Join of %s\n", join)
		}
	}

	managerA.Shutdown()
	managerB.Shutdown()

}
