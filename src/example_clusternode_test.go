package cluster

import (
	"log"
	"testing"
)

func TestExample(t *testing.T) {
	t.Parallel()
	Example()
}

// Example shows a single node able to send and recieve messages
// You can send any type of data to the cluster
func Example() {
	// start cluster 1
	manager := NewManager("node1", "secret")
	manager.AddClusterNode(Node{name: "node2", addr: "127.0.0.1:9605"})
	err := manager.ListenAndServe("127.0.0.1:9604")
	if err != nil {
		log.Fatal(err)
	}

	// start cluster 2
	manager2 := NewManager("node2", "secret")
	manager2.AddClusterNode(Node{name: "node1", addr: "127.0.0.1:9604"})
	err = manager2.ListenAndServe("127.0.0.1:9605")
	if err != nil {
		log.Fatal(err)
	}

	// wait for cluster join to be complete
	<-manager2.NodeJoin

	// send message to all nodes of manager2
	manager2.ToCluster <- "Hello World!"

	// process all channels
	for {
		select {
		case logentry := <-manager.Log:
			log.Printf("manager.log: %s\n", logentry)
		case logentry := <-manager2.Log:
			log.Printf("manager2.log: %s\n", logentry)
		case p := <-manager.FromCluster:
			var cm string
			err := p.Message(&cm)
			if err != nil {
				log.Printf("Unable to get message from package: %s\n", err)
			}
			log.Printf("we received a custom message: %s\n", cm)
			return
		}
	}
}
