package rpcs

// JoinArgs is used for proving args for join RPCs
type JoinArgs struct {
	Port   int
	ID     string
	Weight int
}

// ReplicaArgs is used to convey all list of replica
// to nodes to add in their ring
type ReplicaArgs struct {
	Replicas []RepNode
}

// RepNode represents a replication node info
// that is transferred to the node to convey
// replication info
type RepNode struct {
	Key  string
	Port int
}

// Ack is used to provide acknowledgments for RPCs
type Ack struct {
	Success bool
}
