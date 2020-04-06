package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pokt-network/pocket-runner/internal/types"
	"github.com/pokt-network/pocket-runner/x/runner"
	tmTypes "github.com/tendermint/tendermint/types"
)

func main() {
	args := os.Args[1:]
	Run(args)
}

func Run(args []string) {
	cfg, err := types.GetConfigFromEnv()
	if err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
	// Initial launcher, separated from loop due to passphrase
	cmd, err := runner.LaunchProcess(cfg, args, os.Stdout, os.Stderr, os.Stdin)
	time.Sleep(time.Second * 10)

	var eventListener *runner.EventListener
	waitForUpgrade := func(errs chan error, upgrades chan *types.UpgradeInfo) {
		for {
			var upgrade *types.UpgradeInfo
			select {
			case rawTxEvt := <-eventListener.TxChan:
				if len(rawTxEvt.Events["upgrade.action"]) == 1 {
					if err := upgrade.SetUpgrade(strings.Join(rawTxEvt.Events["upgrade.action"], "")); err != nil {
						errs <- err
					}

					if err := types.CheckBinary(cfg.UpgradeBin(upgrade.Name)); err != nil {
						if cfg.AllowDownload {
							// download
						} else {
							errs <- err
						}
					}
					upgrades <- upgrade
				}
			}
		}
	}
	waitForBlockHeight := func(errs chan error, upgrades chan *types.UpgradeInfo) {
		for {
			select {
			case upgrade := <-upgrades:
				for {
					select {
					case rawHeaderEvt := <-eventListener.HeaderChan:
						headerEvt := rawHeaderEvt.Data.(tmTypes.EventDataNewBlockHeader)
						if upgrade.Height != headerEvt.Header.Height {
							continue
						}
						if err := cmd.Process.Kill(); err != nil {
							errs <- err
						}
						if err := runner.Upgrade(cfg, upgrade); err != nil {
							errs <- err
						}
						cmd, err = runner.LaunchProcess(cfg, args, os.Stdout, os.Stderr, os.Stdin)
						if err != nil {
							errs <- err
						}
					}
				}
			}
		}
	}

	var errs chan error
	var upgrades chan *types.UpgradeInfo

	go waitForUpgrade(errs, upgrades)
	go waitForBlockHeight(errs, upgrades)

	for {
		select {
		case err := <-errs:
			fmt.Printf("%+v\n", err)
			os.Exit(1)
		}
	}
}
