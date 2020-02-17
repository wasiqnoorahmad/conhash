package loadbalancer

import (
	"conhash/consistent"
	"conhash/rpcs"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
)

// loadBalancer struct maintains the variables
// required for consistent hashing
type loadBalancer struct {
	listener net.Listener // RPC listener of load balancer ...
	ring     consistent.CRing
	joinCh   chan joinEx
	reqCh    chan requestEx
	leaveCh  chan leaveEx
}

// New returns a new instance of loadbalancer but does
// not start it
func New() LoadBalancer {
	return &loadBalancer{
		joinCh:  make(chan joinEx),
		reqCh:   make(chan requestEx),
		leaveCh: make(chan leaveEx),
		ring:    *consistent.NewRing(),
	}
}

// StartLB starts the RPC server for Loadbalancer and
// launches appropriate go routines to serve nodes and UE
func (lb *loadBalancer) StartLB(port int) error {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	lb.listener = listener
	rpcServer := rpc.NewServer()
	rpcServer.Register(rpcs.WrapLoadBalancer(lb))
	rpcServer.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)
	go http.Serve(listener, nil)
	go lb.handleRequests()
	return nil
}

// Close closes all go routines and connections
func (lb *loadBalancer) Close() {
	lb.listener.Close()
}

func (lb *loadBalancer) Join(args *rpcs.JoinArgs, reply *rpcs.Ack) error {
	ex := joinEx{args: args, rep: make(chan rpcs.Ack)}
	lb.joinCh <- ex
	*reply = <-ex.rep
	return nil
}

func (lb *loadBalancer) Forward(args *rpcs.ReqArgs, reply *rpcs.Ack) error {
	ex := requestEx{args: args, rep: make(chan rpcs.Ack)}
	lb.reqCh <- ex
	*reply = <-ex.rep
	return nil
}

func (lb *loadBalancer) Leave(args *rpcs.LeaveArgs, reply *rpcs.Ack) error {
	ex := leaveEx{args: args, rep: make(chan rpcs.Ack)}
	lb.leaveCh <- ex
	*reply = <-ex.rep
	return nil
}

func (lb *loadBalancer) handleRequests() {
	fmt.Println("LB ready to serve...")
	for {
		select {
		case ex := <-lb.joinCh:
			// Joining Node
			rep := lb.joinNode(ex.args)
			lb.assignReplicas(ex.args.ID)
			lb.lookupKeys(ex.args.ID)

			// Previous Node
			lb.assignPrev(ex.args.ID)

			// Next Node
			// Delete Keys
			lb.removeKeys(ex.args.ID)
			lb.ring.Display()
			ex.rep <- rep

		case ex := <-lb.reqCh:
			ex.rep <- lb.forward(ex.args)

		case ex := <-lb.leaveCh:
			fmt.Println("Leave request received for", ex.args.ID)
			lb.leaveNode(ex.args.ID)
			ex.rep <- rpcs.Ack{Success: true}
			lb.ring.Display()
		}
	}
}

func (lb *loadBalancer) replaceReplica(key string) {
	node := lb.ring.GetNext(key)
	walk := 0

	for walk != node.Weight {
		node = lb.ring.GetNext(lb.ring.GetVirKey(key, walk))
		prev := lb.ring.GetPrevParent(node)
		next := lb.ring.GetNextExcept(node, prev.ParentKey)

		args := rpcs.ReplaceArgs{
			Old: node.Key,
			New: rpcs.RepNode{
				Key:       next.Key,
				ParentKey: next.ParentKey,
				Port:      next.Port,
			},
		}
		reply := rpcs.Ack{}

		// Send via RPC
		err := prev.Conn.Call("Node.Replace", &args, &reply)
		if err != nil {
			fmt.Println("Cannot Call RPC")
			return
		}

		fmt.Println("For node", prev.Key, "replace", node.Key, "with", next.Key)
		walk++
	}
}

func (lb *loadBalancer) leaveNode(key string) {
	if lb.ring.Size() <= 2 {
		return
	}

	lb.replaceReplica(key)

	node := lb.ring.GetNext(key)
	walk := 0

	for walk != node.Weight {
		node = lb.ring.GetNext(lb.ring.GetVirKey(key, walk))
		prev := lb.ring.GetPrevParent(node)

		args := rpcs.CopyArgs{
			Target: node.Key,
		}
		reply := rpcs.Ack{}

		// Send via RPC
		err := prev.Conn.Call("Node.Copy", &args, &reply)
		if err != nil {
			fmt.Println("Cannot Call RPC Node.Copy")
			return
		}

		fmt.Println("For node", prev.Key, "transfer all keys with parent key =", node.Key)
		walk++
	}
	// Replace Replica
	lb.ring.RemoveNode(key)
}

func (lb *loadBalancer) removeKeys(key string) {
	if lb.ring.Size() <= 2 {
		return
	}

	node := lb.ring.GetNext(key)

	walk := 0
	for walk != node.Weight {
		node = lb.ring.GetNext(lb.ring.GetVirKey(key, walk))
		prev := lb.ring.GetPrevParent(node)
		next := lb.ring.GetNextExcept(node, prev.ParentKey)

		if next != nil {
			fmt.Println("For", node.Key, "Delete Keys from", next.Key, "of node", prev.Key)
			// Send via RPC
			args := rpcs.RemoveAll{
				ID: prev.Key,
			}
			reply := rpcs.Ack{}

			if err := next.Conn.Call("Node.RemoveAll", &args, &reply); err != nil {
				fmt.Println("Cannot call RPC")
			}
		}
		walk++
	}
}

func (lb *loadBalancer) lookupKeys(key string) {
	// args := rpcs.Lookup{}

	node := lb.ring.GetNext(key)

	walk := 0
	for walk != node.Weight {
		node = lb.ring.GetNext(lb.ring.GetVirKey(key, walk))
		prev := lb.ring.GetPrevParent(node)
		next := lb.ring.GetNextParent(node)

		if prev == nil || next == nil {
			return
		}
		args := rpcs.LookupInfo{
			Start: prev.Hash + 1,
			End:   node.Hash,
			Key:   node.Key,
		}

		// Send via RPC
		reply := rpcs.Ack{}
		err := node.Conn.Call("Node.Lookup", &args, &reply)
		if err != nil {
			return
		}

		fmt.Println("Node", key, "looking up between", prev.Hash+1, "<->", node.Hash, "from", next.Key)
		walk++
	}

}

func (lb *loadBalancer) assignPrev(key string) {
	node := lb.ring.GetNext(key)
	prev := lb.ring.GetPrevParent(node)
	if prev != nil {
		// fmt.Println("Previous Node of", node.Key, "is", prev.ParentKey)
		lb.assignReplicas(prev.ParentKey)
	}
}

func (lb *loadBalancer) joinNode(args *rpcs.JoinArgs) rpcs.Ack {
	success := lb.ring.AddNode(args)
	return rpcs.Ack{Success: success}
}

// forward is called when a request needs to be
// sent to a node in a ring
func (lb *loadBalancer) forward(args *rpcs.ReqArgs) rpcs.Ack {
	node := lb.ring.GetNext(args.ID)

	if node != nil {
		args.NodeID = node.Key
		hash := lb.ring.GenHash(args.ID)
		fmt.Println("User hash is", hash, "<->", node.Hash)

		reply := rpcs.Ack{}
		if err := node.Conn.Call("Node.GetRequest", &args, &reply); err != nil {
			fmt.Println("Cannot call RPC")
			return rpcs.Ack{Success: false}
		}
		return reply
	}
	return rpcs.Ack{Success: false}
}

// assignReplicas returns a slice of node keys that are
// assigned as the replica nodes
func (lb *loadBalancer) assignReplicas(key string) {
	if lb.ring.Size() == 1 {
		return
	}

	var replicas []rpcs.RepNode

	node := lb.ring.GetNext(key)
	walk := 0

	for walk != node.Weight {
		node = lb.ring.GetNext(lb.ring.GetVirKey(key, walk))
		replica := lb.ring.GetNextParent(node)

		if replica != nil {
			fmt.Println("Replica of", node.Key, "is", replica.Key)
			repNode := rpcs.RepNode{
				ParentKey: replica.ParentKey,
				Key:       replica.Key,
				Port:      replica.Port,
			}
			replicas = append(replicas, repNode)
		}
		walk++
	}

	// Send via RPC
	args := rpcs.ReplicaArgs{
		Replicas: replicas,
	}
	reply := rpcs.Ack{}
	if err := node.Conn.Call("Node.GetReplicas", &args, &reply); err != nil {
		fmt.Println("Cannot call RPC")
		return
	} else if !reply.Success {
		return
	}
}
