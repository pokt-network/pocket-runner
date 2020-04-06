package main

import (
	"bytes"
	"context"
	"os"
	"sync"
	"testing"

	"github.com/pkg/errors"
	"github.com/pokt-network/pocket-runner/internal/types"
	"github.com/pokt-network/pocket-runner/x/runner"
	"github.com/pokt-network/posmint/x/gov"
	govTypes "github.com/pokt-network/posmint/x/gov/types"
	tmTypes "github.com/tendermint/tendermint/types"
)

func TestRun(t *testing.T) {
	_, kb, cleanup := NewInMemoryTendermintNode(t, oneValTwoNodeGenesisState())
	cb, err := kb.GetCoinbase()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	ctx, cancel := context.WithCancel(context.Background())
	home, err := copyTestData("validate")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	cfg := &types.Config{Home: home, Name: "test-runnerd"}
	defer os.RemoveAll(home)
	const version = "RC-0.2.0"
	var stdout, stderr, stdin bytes.Buffer

	args := []string{"start", "--blockTime", "1"} // NOTE add short block times for testing purposes
	cmd, err := runner.LaunchProcess(cfg, args, &stdout, &stderr, &stdin)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	upgrades := make(chan *types.UpgradeInfo)
	complete := make(chan string)
	errs := make(chan error)

	memCli, stopCli, evtChan := subscribeTo(t, tmTypes.EventNewBlock)

	listener := runner.NewEventListener()
	go WaitForUpgrade(ctx, cfg, listener, upgrades, errs)
	go WaitForBlockHeight(ctx, cfg, args, cmd, listener, upgrades, complete, errs)

	// intercept any errors from Upgrades
	go func() {
		for {
			select {
			case err := <-errs:
				t.Error(err)
				t.FailNow()
			case <-ctx.Done():
				t.Log("error routine ending")
				return
			}
		}
	}()

	select {
	case <-evtChan:
		tx, err := gov.UpgradeTx(memCodec(), memCli, kb, cb.GetAddress(), govTypes.Upgrade{
			Height:  2,
			Version: version,
		}, "test")
		if tx == nil {
			t.Error(errors.New("tx is nil"))
			t.FailNow()
		}
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		for {
			select {
			case <-complete: // NOTE an upgrade was completed
				upgradeBin := cfg.UpgradeBin("RC-0.2.0")
				currentBin, err := cfg.CurrentBin()
				t.Log(upgradeBin)
				t.Log(currentBin)
				if err != nil {
					t.Error(err)
					t.FailNow()
				}
				if upgradeBin != currentBin {
					t.Errorf("upgrade bin: %s does not match current bin: %s", upgradeBin, currentBin)
				}
				wg.Done()
			case <-ctx.Done():
				t.Log("test assertion was completed")
				return
			}
		}
	}(wg)
	wg.Wait()
	cancel()
	listener.Stop()
	stopCli()
	cleanup()
	t.Log("test should have ended")
	return
}
func TestWaitForUpgrade(t *testing.T) {
	t.Log("test is beggining")
	_, kb, cleanup := NewInMemoryTendermintNode(t, oneValTwoNodeGenesisState())
	cb, err := kb.GetCoinbase()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	ctx, cancel := context.WithCancel(context.Background())
	home, err := copyTestData("validate")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	cfg := &types.Config{Home: home, Name: "test-runnerd"}
	defer os.RemoveAll(home)
	const version = "RC-0.2.0"

	upgrades := make(chan *types.UpgradeInfo)
	errs := make(chan error)

	memCli, stopCli, evtChan := subscribeTo(t, tmTypes.EventNewBlock)

	listener := runner.NewEventListener()
	go WaitForUpgrade(ctx, cfg, listener, upgrades, errs)
	go func() {
		for {
			select {
			case err := <-errs:
				t.Error(err)
				t.FailNow()
			case <-ctx.Done():
				t.Log("Error routine is over")
				return
			}
		}
	}() // intercept any errors from Upgrades

	select {
	case <-evtChan:
		memCli, stopCli, evtChan = subscribeTo(t, tmTypes.EventNewBlockHeader)
		tx, err := gov.UpgradeTx(memCodec(), memCli, kb, cb.GetAddress(), govTypes.Upgrade{
			Height:  2,
			Version: version,
		}, "test")
		if tx == nil {
			t.Error(errors.New("tx is nil"))
			t.FailNow()
		}
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
	}
	wg := &sync.WaitGroup{}
	go func(wg *sync.WaitGroup) {
		for {
			select {
			case upgrade := <-upgrades: // NOTE this means the tx was intercepted & sent as a valid upgrade for the runner
				if upgrade.Name != version {
					t.Errorf("received upgrade: %s does not match expected %s", upgrade.Name, version)
				}
				wg.Done()
			case <-ctx.Done():
				t.Log("test assertion was completed")
				return
			}
		}
	}(wg)
	wg.Wait()
	cancel()
	stopCli()
	listener.Stop()
	cleanup()
	t.Log("test ended")
	return
}
