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
	unRepl   []string
	stateMap map[string]rpcs.State
	weight   int
	myPort   int
	id       string
	listener net.Listener // RPC listener of node
	ring     consistent.CRing
	repCh    chan replicaEx
	reqCh    chan requestEx
}

// New returns a new instance of loadbalancer but does
// not start it
func New(port int, id string, weight int) Node {
	return &node{
		myPort:   port,
		id:       id,
		ring:     *consistent.NewRing(),
		repCh:    make(chan replicaEx),
		reqCh:    make(chan requestEx),
		weight:   weight,
		stateMap: make(map[string]rpcs.State),
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
		case repEx := <-n.repCh:
			rep := n.updateRing(repEx.args)
			if len(n.unRepl) > 0 {
				n.tryReplicate()
			}
			repEx.rep <- rep

		case reqEx := <-n.reqCh:
			rep := n.updateState(reqEx.args)
			succ := n.replState(reqEx.args.ID)
			if !succ {
				n.unRepl = append(n.unRepl, reqEx.args.ID)
			}
			reqEx.rep <- rep
		}
	}
}

func (n *node) tryReplicate() {
	var failure []string

	for _, key := range n.unRepl {
		res := n.replState(key)
		if !res {
			failure = append(failure, key)
		}
	}
	n.unRepl = failure
}

func (n *node) updateState(args *rpcs.ReqArgs) rpcs.Ack {
	// Check if state already exist
	userSt, exist := n.stateMap[args.ID]
	if !exist {
		// fmt.Println("Size is", n.ring.Size())
		// replica := n.ring.GetNext(args.ID)
		// if replica != nil {
		userSt = rpcs.State{
			Primary: args.NodeID,
		}
		// Replica: replica.Key,
		// }
	}
	// }
	n.stateMap[args.ID] = userSt
	return rpcs.Ack{Success: true}
}

func (n *node) replState(key string) bool {
	// Check if state already exist
	userSt, exist := n.stateMap[key]
	if exist {
		fmt.Println("State Exist")
		replica := n.ring.GetNext(key)
		if replica != nil {
			fmt.Println("Replica is", replica)
			userSt.Replica = replica.Key
			syncArgs := rpcs.SyncArgs{UserState: userSt}
			reply := rpcs.Ack{}
			err := replica.Conn.Call("Node.RecvState", &syncArgs, &reply)
			if err != nil {
				fmt.Println("Cannot call RPC")
			} else if reply.Success {
				fmt.Println("State Replicated")
				n.stateMap[key] = userSt
				return true
			}
		}
	}
	return false
}

func (n *node) updateRing(args *rpcs.ReplicaArgs) rpcs.Ack {
	for i := 0; i < len(args.Replicas); i++ {
		replica := args.Replicas[i]
		res := n.ring.AddSolo(replica.Key, replica.ParentKey, replica.Port)
		if !res {
			fmt.Println("Unable to add", replica)
		}
		fmt.Println("Replica Key", args.Replicas[i].Key)
	}
	return rpcs.Ack{Success: true}
}

func (n *node) GetStatus(args *rpcs.Ack, reply *rpcs.Ack) error {
	return nil
}

func (n *node) RecvState(args *rpcs.SyncArgs, reply *rpcs.Ack) error {
	fmt.Println("State Replicated")
	*reply = rpcs.Ack{Success: true}
	return nil
}

func (n *node) GetRequest(args *rpcs.ReqArgs, reply *rpcs.Ack) error {
	reqEx := requestEx{
		args: args,
		rep:  make(chan rpcs.Ack),
	}
	n.reqCh <- reqEx
	fmt.Println("Request rcvd", args.ID, args.NodeID)
	*reply = <-reqEx.rep
	return nil
}

func (n *node) GetReplicas(args *rpcs.ReplicaArgs, ack *rpcs.Ack) error {
	repEx := replicaEx{
		args: args,
		rep:  make(chan rpcs.Ack),
	}
	n.repCh <- repEx
	*ack = <-repEx.rep
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
