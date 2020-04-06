package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
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

	waitForUpgrade := func(ctx context.Context, errs chan error, upgrades chan *types.UpgradeInfo, eventListener *runner.EventListener) {
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
							if er := runner.DownloadBinary(cfg, upgrade); er != nil {
								errs <- er
							}
						} else {
							errs <- err
						}
					}
					upgrades <- upgrade
				}
			case <-ctx.Done():
				return // singal to kill process was sent terminate exectuion
			}
		}
	}
	waitForBlockHeight := func(ctx context.Context, errs chan error, upgrades chan *types.UpgradeInfo, eventListener *runner.EventListener) {
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
						if err := cmd.Process.Kill(); err != nil { // PROCESS MUST DIE BEFORE UPGRADING; cfg.Current is a symlink otherwise bugs might happen
							errs <- err
						}
						if err := runner.Upgrade(cfg, upgrade); err != nil {
							errs <- err
						}
						cmd, err = runner.LaunchProcess(cfg, args, os.Stdout, os.Stderr, os.Stdin)
						if err != nil {
							errs <- err
						}
					case <-ctx.Done():
						return // singal to kill process was sent in case it was sent inside a particular upgrade terminate execution
					}
				}
			case <-ctx.Done():
				return // singal to kill process was sent in between upgrades terminate exection
			}
		}
	}

	var errs chan error
	var upgrades chan *types.UpgradeInfo
	var tmListener = runner.NewEventListener()
	ctx, cancel := context.WithCancel(context.Background())

	signals := make(chan os.Signal, 1)
	signal.Notify(signals,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
		os.Kill,
		os.Interrupt)

	go waitForUpgrade(ctx, errs, upgrades, tmListener)
	go waitForBlockHeight(ctx, errs, upgrades, tmListener)

	for {
		select {
		case err := <-errs:
			fmt.Printf("%+v\n", err)
			os.Exit(1)
		case <-signals:
			cancel()
			tmListener.Stop()
			if err := cmd.Process.Kill(); err != nil {
				fmt.Printf("%+v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}
}
