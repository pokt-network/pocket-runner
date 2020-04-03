package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pokt-network/pocket-runner/internal/types"
	"github.com/pokt-network/pocket-runner/x/runner"
)

func main() {
	args := os.Args[1:]
	err := Run(args)
	if err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
}

func Run(args []string) error {
	// cfg, err := types.GetConfigFromEnv()
	cfg := &types.Config{Home: "/Users/edjroz/Repos/pocket-runner/x/runner/testdata/validate", Name: "custom-core"}
	// if err != nil {
	// 	return err
	// }
	_, err := runner.LaunchProcess(cfg, args, os.Stdout, os.Stderr, os.Stdin)
	time.Sleep(time.Second * 10)
	return err
}
