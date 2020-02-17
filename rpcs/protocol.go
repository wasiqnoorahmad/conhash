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

// ReplaceArgs is used to replace any replica with new one
type ReplaceArgs struct {
	Old string
	New RepNode
}

// type LookupArgs struct {
// 	Start
// }

// CopyArgs is called when keys of a node (Src) need to
// be copied to their new replicas
type CopyArgs struct {
	Target string
}

// RemoveAll is used to call when all keys of a node
// (ID) needs to be deleted
type RemoveAll struct {
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
	Hash    uint64
}

// // LookupReply ...
// type LookupReply struct {
// 	States map[string]State
// }

// BulkStates ...
type BulkStates struct {
	States map[string]State
}

// LookupInfo defines a lookup value...
type LookupInfo struct {
	Start uint64
	End   uint64
	Key   string
	Dst   string
}

// SyncArgs ...
type SyncArgs struct {
	Key       string
	UserState State
}

// Ack is used to provide acknowledgments for RPCs
type Ack struct {
	Success bool
}
