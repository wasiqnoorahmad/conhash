package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

// === Cmd Line Args ===
var (
	testcase = flag.String("--t", "foo", "Test case id")
	seed     = 8080
)

func cleanup() {
	exec.Command("pkill", "--signal", "SIGKILL", "node").Run()
	exec.Command("pkill", "--signal", "SIGKILL", "lb").Run()

	// os.Remove(".lb.lock")
}

func waitUntilExist(fpath string) {
	isExist := false
	for !isExist {
		if _, err := os.Stat(fpath); err == nil {
			isExist = true
		}
	}
}

func startLB() error {
	cmd := exec.Command("go", "run", "runner/loadbalancer/lb.go", "--p=", strconv.Itoa(seed))

	err := cmd.Start()
	if err != nil {
		return err
	}
	waitUntilExist(".lb.lock")
	go func() {
		cmd.Wait()
	}()
	return nil
}

func main() {

	cleanup()
	flag.Parse()

	if err := startLB(); err != nil {
		fmt.Println("Unable to start LoadBalancer")
	} else {
		fmt.Println("Load Balancer started")
	}

	// time.Sleep(20 * time.Second)
	cleanup()

}
