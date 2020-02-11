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
	testcase   = flag.String("--t", "foo", "Test case id")
	lbPort     = 8080
	seed       = lbPort + 1
	lbRunner   = "runner/lb/lb.go"
	nodeRunner = "runner/node/node.go"
)

func cleanup() {
	exec.Command("pkill", "--signal", "SIGKILL", "node").Run()
	exec.Command("pkill", "--signal", "SIGKILL", "lb").Run()

	os.Remove(".lb.lock")
	os.Remove(".node.lock")

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
	cmd := exec.Command("go", "run", lbRunner, "--p=", strconv.Itoa(lbPort))

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

func startNode() error {
	cmd := exec.Command("go", "run", nodeRunner, "--p=", strconv.Itoa(seed))

	err := cmd.Start()
	if err != nil {
		return err
	}
	// waitUntilExist(".node.lock")
	go func() {
		cmd.Wait()
	}()
	seed++
	return nil
}

func setupExp(nodeCount int) error {
	// Start LoadBalancer first
	if err := startLB(); err != nil {
		return err
	}
	fmt.Println("LoadBalancer started")
	for nodeCount > 0 {
		if err := startNode(); err != nil {
			return err
		}
		nodeCount--
	}
	fmt.Println("Nodes started")
	return nil
}

func exeExp() error {
	switch *testcase {
	case "foo":
		setupExp(1)
		fmt.Println("Executing Foo")
	}
	return nil
}

func main() {

	cleanup()
	flag.Parse()

	if err := exeExp(); err != nil {
		fmt.Println("Unable to execute testcase")
	}
	cleanup()

}
