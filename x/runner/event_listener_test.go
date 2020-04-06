package runner

import (
	"errors"
	"strings"
	"testing"

	sdk "github.com/pokt-network/posmint/types"
	"github.com/pokt-network/posmint/x/gov"
	govTypes "github.com/pokt-network/posmint/x/gov/types"
	tmTypes "github.com/tendermint/tendermint/types"
)

func TestEventListener(t *testing.T) {
	/*
		Use In Memory App, due to the use of termios for password insertion its not possible to
		pass data to stdin since termios happens at the most primitve UNIX level,
		therefore its not possible to run the binary & pass the password.
	*/
	const version = "RC-0.2.0"
	_, kb, cleanup := NewInMemoryTendermintNode(t, oneValTwoNodeGenesisState())
	cb, err := kb.GetCoinbase()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	memCli, stopCli, evtChan := subscribeTo(t, tmTypes.EventNewBlock)
	var tx *sdk.TxResponse
	var eventListener *EventListener
	select {
	case <-evtChan:
		memCli, stopCli, evtChan = subscribeTo(t, tmTypes.EventNewBlockHeader)
		eventListener = NewEventListener()
		tx, err = gov.UpgradeTx(memCodec(), memCli, kb, cb.GetAddress(), govTypes.Upgrade{
			Height:  2,
			Version: version,
		}, "test")
		if tx == nil {
			t.Error(errors.New("tx is nil"))
			t.FailNow()
		}
	}
	select {
	case tx := <-eventListener.TxChan:
		if len(tx.Events["upgrade.action"]) == 1 {
			// EVENT WAS RECEIVED TEST HAS BEEN SUCCESSFUL
			t.Log(strings.Join(tx.Events["upgrade.action"], ""))
			if !strings.Contains(strings.Join(tx.Events["upgrade.action"], ""), version) {
				t.Error(errors.New("Does not contain expected version"))
				t.FailNow()
			}
			stopCli()
			eventListener.Stop()
			cleanup()
		}
	}
}
