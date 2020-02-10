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
	joinCh   chan (joinEx)
}

// New returns a new instance of loadbalancer but does
// not start it
func New() LoadBalancer {
	return &loadBalancer{}
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
	ex := joinEx{args: args, rep: make(chan *rpcs.Ack)}
	lb.joinCh <- ex
	reply = <-ex.rep
	return nil
}

func (lb *loadBalancer) handleRequests() {
	fmt.Println("LB ready to serve")
	for {
		select {
		case ex := <-lb.joinCh:
			fmt.Println(ex.args.ID)
			ex.rep <- &rpcs.Ack{Success: true}
		}
	}
}

// Forward is called when a request needs to be
// sent to a node in a ring
func (lb *loadBalancer) Forward(key string) {
	node := lb.ring.GetNext(key)
	if node != nil {
		fmt.Println("Key forwarded to", node.Key)
	}
}

// AssignReplicas returns a slice of node keys that are
// assigned as the replica nodes
func (lb *loadBalancer) AssignReplicas(key string) {
	hash := lb.ring.GenHash(key)
	node := lb.ring.GetNext(key)

	var walk uint16 = 0
	for walk < node.Weight {
		parent := lb.ring.GetNextParent(hash)
		if parent != nil {
			fmt.Println("Replica is", hash)
		}
		hash = lb.ring.GenHash(hash)
		walk++
	}
}
