package node

import (
	"conhash/consistent"
	"conhash/rpcs"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
)

// loadBalancer struct maintains the variables
// required for consistent hashing
type node struct {
	weight   int
	myPort   int
	id       string
	listener net.Listener // RPC listener of node
	ring     consistent.CRing
	repCh    chan replicaEx
}

// New returns a new instance of loadbalancer but does
// not start it
func New(port int, id string, weight int) Node {
	return &node{
		myPort: port,
		id:     id,
		weight: weight,
		ring:   *consistent.NewRing(),
		repCh:  make(chan replicaEx),
	}
}

func (n *node) StartNode(dst string) error {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(n.myPort))
	if err != nil {
		return err
	}
	n.listener = listener
	rpcServer := rpc.NewServer()
	rpcServer.Register(rpcs.WrapNode(n))
	rpcServer.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)
	go http.Serve(listener, nil)
	go n.handleRequests()
	if err = n.joinLB(dst); err != nil {
		return err
	}
	return nil
}

func (n *node) handleRequests() {
	for {
		select {
			repEx 
		}
	}
}

func (n *node) GetStatus(args *rpcs.Ack, ack *rpcs.Ack) error {
	return nil
}

func (n *node) GetReplicas(args *rpcs.ReplicaArgs, ack *rpcs.Ack) error {
	for i := 0; i < len(args.Replicas); i++ {
		fmt.Println("Replica Key", args.Replicas[i].Key)
	}
	*ack = rpcs.Ack{Success: true}
	return nil
}

func (n *node) joinLB(dst string) error {

	reply := rpcs.Ack{}

	// Sending join Request to LoadBalancer
	conn, err := rpc.DialHTTP("tcp", dst)

	if err != nil {
		return err
	}
	defer conn.Close()
	args := rpcs.JoinArgs{
		Port:   n.myPort,
		Weight: n.weight,
		ID:     n.id,
	}
	if err := conn.Call("LoadBalancer.Join", &args, &reply); err != nil {
		return err
	} else if !reply.Success {
		return errors.New("LoadBalancer returned failure")
	}
	return nil
}

func (n *node) Close() {
	n.listener.Close()
}
