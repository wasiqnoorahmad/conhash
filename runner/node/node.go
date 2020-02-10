package main

import (
	"conhash/node"
	"fmt"
)

func main() {
	node := node.New()
	err := node.StartNode(5555)

	if err != nil {
		fmt.Println("Unable to start Node", err)
		return
	}

	for {

	}
}
