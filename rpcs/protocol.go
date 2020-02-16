package rpcs

// JoinArgs is used for proving args for join RPCs
type JoinArgs struct {
	Port   int
	ID     string
	Parent string
	Weight int
}

// LeaveArgs is called when a node is leaving network
type LeaveArgs struct {
	ID string
}

// ReqArgs represents a user request
type ReqArgs struct {
	ID     string
	NodeID string
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
	ParentKey string
	Key       string
	Port      int
}

// State is a user state
type State struct {
	Primary string
	Replica string
}

// SyncArgs ...
type SyncArgs struct {
	UserState State
}

// Ack is used to provide acknowledgments for RPCs
type Ack struct {
	Success bool
}
