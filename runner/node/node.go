package main

import (
	"conhash/node"
	"flag"
	"fmt"
	"strconv"
)

var (
	port = flag.Int("--p", 5555, "Port number of node")
	id   = flag.String("--i", strconv.Itoa(*port), "ID of the node")
	dst  = flag.String("--d", ":8080", "HostPort of the loadbalancer")
)

func main() {
	flag.Parse()
	node := node.New(*port, *id)
	err := node.StartNode(*dst)

	if err != nil {
		fmt.Println("Unable to start Node", err)
		return
	}

	for {

	}
}
