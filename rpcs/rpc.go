package rpcs

// RemoteNode - Students should not use this interface in their code. Use WrapNode() instead.
type RemoteNode interface {
	GetStatus(args *Ack, reply *Ack) error
	GetRequest(args *ReqArgs, reply *Ack) error
	GetReplicas(args *ReplicaArgs, reply *Ack) error
	RecvState(args *SyncArgs, reply *Ack) error
	// OnLeave
}

// RemoteLoadBalancer - Students should not use this interface in their code. Use WrapLB() instead.
type RemoteLoadBalancer interface {
	Join(args *JoinArgs, reply *Ack) error
	Forward(args *ReqArgs, reply *Ack) error
}

// Node ...
type Node struct {
	// Embed all methods into the struct. See the Effective Go section about
	// embedding for more details: golang.org/doc/effective_go.html#embedding
	RemoteNode
}

// LoadBalancer ...
type LoadBalancer struct {
	// Embed all methods into the struct. See the Effective Go section about
	// embedding for more details: golang.org/doc/effective_go.html#embedding
	RemoteLoadBalancer
}

// WrapNode wraps t in a type-safe wrapper struct to ensure that only the desired
// methods are exported to receive RPCs. Any other methods already in the
// input struct are protected from receiving RPCs.
func WrapNode(t RemoteNode) RemoteNode {
	return &Node{t}
}

// WrapLoadBalancer wraps t in a type-safe wrapper struct to ensure that only the desired
// methods are exported to receive RPCs. Any other methods already in the
// input struct are protected from receiving RPCs.
func WrapLoadBalancer(t RemoteLoadBalancer) RemoteLoadBalancer {
	return &LoadBalancer{t}
}
