package main

import (
	"conhash/loadbalancer"
	"flag"
	"os"
)

var (
	port = flag.Int("p", 8080, "Port number of LoadBalancer")
)

func createLock() error {
	f, err := os.Create(".lb.lock")
	if err != nil {
		return err
	}
	f.Close()

	return nil
}

func main() {
	lb := loadbalancer.New()
	err := lb.StartLB(*port)

	if err != nil {
		return
	}

	// if err = createLock(); err != nil {
	// 	return
	// }

	for {

	}
}
