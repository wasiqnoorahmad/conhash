package node

import (
	"conhash/rpcs"
)

// Node ...
type Node interface {
	StartNode(dst string) error

	// Closes the Node
	Close()
}

type replicaEx struct {
	args *rpcs.ReplicaArgs
	rep  chan rpcs.Ack
}

type requestEx struct {
	args *rpcs.ReqArgs
	rep  chan rpcs.Ack
}
