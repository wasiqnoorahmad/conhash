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
	unRepl    []string
	stateMap  map[string]rpcs.State
	weight    int
	myPort    int
	id        string
	listener  net.Listener // RPC listener of node
	ring      consistent.CRing
	repCh     chan replicaEx
	reqCh     chan requestEx
	rmvCh     chan removeEx
	cpyCh     chan copyEx
	lookupCh  chan lookupEx
	bulkCh    chan bulkEx
	stateCh   chan stateEx
	replaceCh chan replaceEx
}

// New returns a new instance of loadbalancer but does
// not start it
func New(port int, id string, weight int) Node {
	return &node{
		myPort:    port,
		id:        id,
		ring:      *consistent.NewRing(),
		repCh:     make(chan replicaEx),
		reqCh:     make(chan requestEx),
		rmvCh:     make(chan removeEx),
		cpyCh:     make(chan copyEx),
		replaceCh: make(chan replaceEx),
		lookupCh:  make(chan lookupEx),
		bulkCh:    make(chan bulkEx),
		stateCh:   make(chan stateEx),
		weight:    weight,
		stateMap:  make(map[string]rpcs.State),
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

		case ex := <-n.stateCh:
			fmt.Println("State replicat at backup")
			n.stateMap[ex.args.Key] = ex.args.UserState
			ex.rep <- rpcs.Ack{Success: true}

		case rmvEx := <-n.rmvCh:
			fmt.Println("Going to remove all keys with replica key =", rmvEx.args.ID)
			n.removeAll(rmvEx.args.ID)
			rmvEx.rep <- rpcs.Ack{Success: true}

		case cpyEx := <-n.cpyCh:
			fmt.Println("Copy Called")
			n.replicateKeys(cpyEx.args.Target)
			cpyEx.rep <- rpcs.Ack{Success: true}

		case repEx := <-n.replaceCh:
			fmt.Println("Replace Called")
			n.replaceNodes(repEx.args)
			repEx.rep <- rpcs.Ack{Success: true}

		case lukupEx := <-n.lookupCh:
			n.lookupKeys(lukupEx.args)
			lukupEx.rep <- rpcs.Ack{Success: true}

		case bulkEx := <-n.bulkCh:
			bulkEx.rep <- n.cpyBulk(bulkEx.args)
		}
	}
}

func (n *node) cpyBulk(args *rpcs.LookupInfo) rpcs.BulkStates {
	bulk := rpcs.BulkStates{
		States: make(map[string]rpcs.State),
	}

	for key, state := range n.stateMap {
		if args.Start < args.End && state.Hash >= args.Start && state.Hash <= args.End {
			bulk.States[key] = state
			state.Primary = args.Key
			state.Replica = args.Dst
		} else if args.Start > args.End && state.Hash >= args.Start || state.Hash <= args.End {
			bulk.States[key] = state
			state.Primary = args.Key
			state.Replica = args.Dst
		}
	}
	return bulk
}

func (n *node) lookupKeys(args *rpcs.LookupInfo) {
	next := n.ring.GetNextParentWithKey(args.Key)
	if next != nil {
		bulkStates := rpcs.BulkStates{}
		args.Dst = next.Key

		err := next.Conn.Call("Node.CopyBulk", args, &bulkStates)
		if err != nil {
			fmt.Println("Cannot call RPC")
			return
		}
		fmt.Println("States are", bulkStates)

		for key, state := range bulkStates.States {
			state.Primary = args.Key
			state.Replica = next.Key
			n.stateMap[key] = state
		}
	}

}

func (n *node) replaceNodes(args *rpcs.ReplaceArgs) {
	n.ring.RemoveSolo(args.Old)
	n.ring.AddSolo(args.New.Key, args.New.ParentKey, args.New.Port)

}

func (n *node) replicateKeys(target string) {
	for key, state := range n.stateMap {
		if state.Replica == target {
			ok := n.replState(key)
			if !ok {
				n.unRepl = append(n.unRepl, key)
			}
		}
	}
}

func (n *node) removeAll(key string) {
	for _, state := range n.stateMap {
		if state.Primary == key {
			delete(n.stateMap, key)
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
		userSt = rpcs.State{
			Hash: n.ring.GenHash(args.ID),
		}
	}
	userSt.Primary = args.NodeID
	n.stateMap[args.ID] = userSt
	return rpcs.Ack{Success: true}
}

func (n *node) replState(key string) bool {
	// Check if state already exist
	userSt, exist := n.stateMap[key]
	if exist {
		replica := n.ring.GetNext(key)

		if replica != nil {
			fmt.Println("Replica is", replica)
			userSt.Replica = replica.Key

			syncArgs := rpcs.SyncArgs{
				UserState: userSt,
			}
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

func (n *node) Lookup(args *rpcs.LookupInfo, reply *rpcs.Ack) error {
	ex := lookupEx{
		args: args,
		rep:  make(chan rpcs.Ack),
	}
	n.lookupCh <- ex
	*reply = <-ex.rep
	return nil
}

func (n *node) Replace(args *rpcs.ReplaceArgs, reply *rpcs.Ack) error {
	repEx := replaceEx{
		args: args,
		rep:  make(chan rpcs.Ack),
	}
	n.replaceCh <- repEx
	*reply = <-repEx.rep
	return nil
}

func (n *node) RecvState(args *rpcs.SyncArgs, reply *rpcs.Ack) error {
	stateEx := stateEx{
		args: args,
		rep:  make(chan rpcs.Ack),
	}
	n.stateCh <- stateEx
	*reply = <-stateEx.rep
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

func (n *node) CopyBulk(args *rpcs.LookupInfo, reply *rpcs.BulkStates) error {
	blkEx := bulkEx{
		args: args,
		rep:  make(chan rpcs.BulkStates),
	}
	n.bulkCh <- blkEx
	*reply = <-blkEx.rep
	return nil
}

func (n *node) RemoveAll(args *rpcs.RemoveAll, reply *rpcs.Ack) error {
	rmvEx := removeEx{
		args: args,
		rep:  make(chan rpcs.Ack),
	}
	n.rmvCh <- rmvEx
	*reply = <-rmvEx.rep
	return nil
}

func (n *node) Copy(args *rpcs.CopyArgs, reply *rpcs.Ack) error {
	cpyEx := copyEx{
		args: args,
		rep:  make(chan rpcs.Ack),
	}
	n.cpyCh <- cpyEx
	*reply = <-cpyEx.rep
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
