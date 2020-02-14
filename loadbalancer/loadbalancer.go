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
}

// New returns a new instance of loadbalancer but does
// not start it
func New() LoadBalancer {
	return &loadBalancer{
		joinCh: make(chan joinEx),
		ring:   *consistent.NewRing(),
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

func (lb *loadBalancer) handleRequests() {
	fmt.Println("LB ready to serve")
	for {
		select {
		case ex := <-lb.joinCh:
			// Joining Node
			rep := lb.joinNode(ex.args)
			lb.assignReplicas(ex.args.ID)
			// lb.lookupKeys(ex.args.ID) ??

			// Previous Node
			lb.assignPrev(ex.args.ID)

			// Next Node
			// Delete Keys
			lb.removeKeys(ex.args.ID)
			lb.ring.Display()
			ex.rep <- rep
		}
	}
}

func (lb *loadBalancer) removeKeys(key string) {
	node := lb.ring.GetNext(key)

	walk := 0
	for walk != node.Weight {
		node = lb.ring.GetNext(lb.ring.GetVirKey(key, walk))
		next := lb.ring.GetNextParent(node)

		if next != nil {
			fmt.Println("Delete Keys from", next.ParentKey, "of node", node.Key)
		}
		walk++
	}

}

func (lb *loadBalancer) lookupKeys(key string) {
	node := lb.ring.GetNext(key)

	walk := 0
	for walk != node.Weight {
		node = lb.ring.GetNext(lb.ring.GetVirKey(key, walk))
		next := lb.ring.GetNextParent(node)
		prev := lb.ring.GetPrevParent(node)

		if prev == nil || next == nil {
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
	success := lb.ring.AddNode(args.ID, args.Weight)
	return rpcs.Ack{Success: success}
}

// forward is called when a request needs to be
// sent to a node in a ring
func (lb *loadBalancer) forward(key string) {
	node := lb.ring.GetNext(key)
	if node != nil {
		fmt.Println("Key forwarded to", node.Key)
	}
}

// assignReplicas returns a slice of node keys that are
// assigned as the replica nodes
func (lb *loadBalancer) assignReplicas(key string) {
	if lb.ring.Size() == 1 {
		return
	}

	var replicas []rpcs.RepNode

	node := lb.ring.GetNext(key)
	steps := node.Weight
	hostPort := ":" + node.Key

	for steps > 0 {
		replica := lb.ring.GetNextParent(node)
		if replica != nil {
			// fmt.Println("Replica of", node.Hash, "is", replica.Key)
			repNode := rpcs.RepNode{
				Key: replica.Key,
			}
			replicas = append(replicas, repNode)
		}
		virKey := lb.ring.GetVirKey(key, steps)
		node = lb.ring.GetNext(virKey)
		steps--
	}

	// Send via RPC
	// Sending join Request to LoadBalancer
	// fmt.Println("Connecting", hostPort)
	conn, err := rpc.DialHTTP("tcp", hostPort)

	if err != nil {
		fmt.Println("Error Connecting", hostPort)
		return
	}
	defer conn.Close()
	args := rpcs.ReplicaArgs{
		Replicas: replicas,
	}
	reply := rpcs.Ack{}
	if err := conn.Call("Node.GetReplicas", &args, &reply); err != nil {
		fmt.Println("Cannot call RPC")
		return
	} else if !reply.Success {
		return
		// return errors.New("LoadBalancer returned failure")
	}
}
