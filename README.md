[![Build Status](https://travis-ci.org/rdoorn/cluster.png)](https://travis-ci.org/rdoorn/cluster)

# cluster
Cluster is a cluster service library in Golang
It will allow you to talk to multiple nodes via channels and exchange information based on custom struct's

# Usage
Below is an example of 1 cluster service, with 2 nodes configured (node1 (self) + node2)
```golang
package main

import (
	"github.com/rdoorn/cluster"
	"log"
)

type CustomMessage struct{ Text: string }

func main() {
  manager := NewManager("node1", "secret")
  manager.AddClusterNode("node2", "127.0.0.1:9505"})
  err := manager.ListenAndServe("127.0.0.1:9504")
  if err != nil {
    log.Fatal(err)
  }
  manager.ToCluster <- CustomMessage{ Text: "Hello World!" } // will send data to all nodes except self
  for {
    select {
		  case packet := <- manager.FromCluster:
        cm := &CustomMessage
        err := packet.Message(cm)
        if err != nil {
          log.Printf("Unable to get message from package: %s\n", err)
        }
        log.Printf("we received a custom message: %s\n", cm.Text)
    }
  }
}
```

# Available Interfaces
You can interface with the cluster through channels. available channels are:

Name                | Direction | Type        | Required | Description
------------------- | --------- | ----------- | -------- | -----------
manager.ToCluster   | <-        | interface{} | no       | used to write interface{} data to the cluster
manager.ToNode      | <-        | PM{}        | no       | used to write private messages to a cluster node
manager.FromCluster | ->        | Package{}   | yes      | used to receive cluster packages on from other nodes
manager.QuorumState | ->        | bool        | no       | used to read the current quorum state, will update on node join/leave
manager.NodeJoin    | ->        | string      | no       | name of node joining the cluster
manager.NodeLeave   | ->        | string      | no       | name of node leaving the cluster

## Contributing

1. Clone this repository from GitHub:

        $ git clone git@github.com:rdoorn/cluster.git

2. Create a git branch

        $ git checkout -b my_bug_fix

3. Make your changes/patches/fixes, committing appropriately
4. **Write tests**
5. Run tests

        $ make test

# Authors
        - Author: Ronald Doorn (<rdoorn@schubergphilis.com>)
