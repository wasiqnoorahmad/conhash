package main

import (
	"conhash/loadbalancer"
	"fmt"
)

func main() {
	lb := loadbalancer.New()
	err := lb.StartLB(8080)

	if err != nil {
		fmt.Println("Unable to start loadbalancer", err)
		return
	}

	for {

	}
}
