package main

import (
	"conhash/rpcs"
	"flag"
	"fmt"
	"net/rpc"
)

var (
	id  = flag.String("i", "user", "ID of the User")
	dst = flag.String("d", ":8080", "HostPort of the loadbalancer")
)

func main() {
	flag.Parse()

	conn, err := rpc.DialHTTP("tcp", *dst)

	if err != nil {
		fmt.Println("Unable to connect to LoadBalancer", err)
		return
	}
	defer conn.Close()

	args := rpcs.LeaveArgs{
		ID: *id,
	}
	reply := rpcs.Ack{}

	if err := conn.Call("LoadBalancer.Leave", &args, &reply); err != nil {
		fmt.Println("Unable to call LB RPC", err)
	} else if reply.Success {
		fmt.Println("Leave Success")
		return
	} else {
		fmt.Println("Leave Failed")
		return
	}
}
