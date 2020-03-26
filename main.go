package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pokt-network/pocket-runner/config"
	"github.com/pokt-network/pocket-runner/runner"
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
	cfg, err := config.GetConfigFromEnv()
	if err != nil {
		return err
	}
	_, err = runner.LaunchProcess(cfg, args)
	time.Sleep(time.Minute * 1) // wait for passphrase
	return err
}
