package main

import (
	"conhash/consistent"
	"strconv"
)

func main() {
	ring := consistent.NewRing()

	for i := 0; i < 5; i++ {
		node := consistent.CNode{
			Key:    strconv.Itoa(i),
			Weight: 2,
		}
		hash := ring.GenHash(strconv.Itoa(i))
		ring.AddNode(hash, &node)
	}
	ring.Display()
}
