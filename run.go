package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pokt-network/pocket-runner/internal/types"
	"github.com/pokt-network/pocket-runner/x/runner"
	tmTypes "github.com/tendermint/tendermint/types"
)

func Run(args []string) {
	cfg, err := types.GetConfigFromEnv()
	if err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
	// Initial launcher, separated from loop due to passphrase
	cmd, err := runner.LaunchProcess(cfg, args, os.Stdout, os.Stderr, os.Stdin)
	if err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
	time.Sleep(time.Second * 10)

	errors := make(chan error)
	upgrades := make(chan *types.UpgradeInfo)
	completed := make(chan string)
	var tmListener = runner.NewEventListener()
	ctx, cancel := context.WithCancel(context.Background())

	signals := make(chan os.Signal, 1)
	signal.Notify(signals,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
		os.Kill,
		os.Interrupt)

	go WaitForUpgrade(ctx, cfg, tmListener, upgrades, errors)
	go WaitForBlockHeight(ctx, cfg, args, cmd, tmListener, upgrades, completed, errors)

	go func() {
		for {
			select {
			case upgradeFufilled := <-completed:
				fmt.Printf("Upgrade to %s performed successfully!!\n", upgradeFufilled)
			}
		}
	}()

	for {
		select {
		case err := <-errors:
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

// WaitForBlockHeight listens for upgrades, per upgrade checks the current block header & upgrades if neccesary.
func WaitForBlockHeight(ctx context.Context, cfg *types.Config, args []string, cmd *exec.Cmd, listener *runner.EventListener, upgrades chan *types.UpgradeInfo, complete chan string, errors chan error) {
	var err error
	var currentUpgrade *types.UpgradeInfo

	for {
		select {
		case rawHeaderEvt := <-listener.HeaderChan:
			if currentUpgrade == nil {
				// wait for upgrade if no current upgrade this way the blockHeight won't change
				currentUpgrade = <-upgrades
			}
			upgrade := currentUpgrade
			headerEvt := rawHeaderEvt.Data.(tmTypes.EventDataNewBlockHeader)
			if upgrade.Height != headerEvt.Header.Height {
				continue
			}
			if err := cmd.Process.Kill(); err != nil { // PROCESS MUST DIE BEFORE UPGRADING; cfg.Current is a symlink otherwise bugs might happen
				errors <- err
			}
			if err := runner.Upgrade(cfg, upgrade); err != nil {
				errors <- err
			}
			cmd, err = runner.LaunchProcess(cfg, args, os.Stdout, os.Stderr, os.Stdin)
			if err != nil {
				errors <- err
			}
			complete <- upgrade.Name
			currentUpgrade = nil // Done with this upgrade wait for the next one to go on
		case <-ctx.Done():
			if err := cmd.Process.Kill(); err != nil {
				errors <- err
			}
			return // singal to kill process was sent in case it was sent inside a particular upgrade terminate execution
		}
	}
}

// WaitForUpgrade listens transactions and filters upgrades, passess them to the upgrade channel
func WaitForUpgrade(ctx context.Context, cfg *types.Config, listener *runner.EventListener, upgrades chan *types.UpgradeInfo, errors chan error) {
	for {
		upgrade := &types.UpgradeInfo{}
		select {
		case rawTxEvt := <-listener.TxChan:
			if len(rawTxEvt.Events["upgrade.action"]) == 1 {
				if err := upgrade.SetUpgrade(strings.Join(rawTxEvt.Events["upgrade.action"], "")); err != nil {
					errors <- err
				}

				if err := types.CheckBinary(cfg.UpgradeBin(upgrade.Name)); err != nil {
					if cfg.AllowDownload {
						if er := runner.DownloadBinary(cfg, upgrade); er != nil {
							errors <- er
						} else {
							upgrades <- upgrade
							continue
						}
					} else {
						errors <- err
					}
				}
				upgrades <- upgrade
			}
		case <-ctx.Done():
			return // singal to kill process was sent terminate exectuion
		}
	}
}
