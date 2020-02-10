package node

import (
	"conhash/consistent"
	"conhash/rpcs"
	"errors"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
)

// loadBalancer struct maintains the variables
// required for consistent hashing
type node struct {
	myPort   int
	id       string
	listener net.Listener // RPC listener of load balancer ...
	ring     consistent.CRing
}

// New returns a new instance of loadbalancer but does
// not start it
func New(port int, id string) Node {
	return &node{
		myPort: port,
		id:     id,
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
	// go lb.handleRequests()
	if err = n.joinLB(dst); err != nil {
		return err
	}
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
		Port: n.myPort,
		ID:   n.id,
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
