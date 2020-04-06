package runner

import (
	"os"
	"testing"

	"github.com/pokt-network/pocket-runner/internal/types"
)

// TODO: test with download (and test all download functions)
func TestUpgrade(t *testing.T) {
	home, err := copyTestData("validate")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer os.RemoveAll(home)

	cfg := &types.Config{Home: home, Name: "test-runnerd"}

	currentBin, err := cfg.CurrentBin()
	t.Log(currentBin)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if cfg.GenesisBin() != currentBin {
		t.Errorf("Genesis Bin & Current Bin do not match")
		t.FailNow()
	}

	// make sure it updates a few times
	for _, upgrade := range []string{"RC-0.2.0"} {
		// now set it to a valid upgrade and make sure CurrentBin is now set properly
		info := &types.UpgradeInfo{Name: upgrade}
		err = Upgrade(cfg, info)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		// we should see current point to the new upgrade dir
		upgradeBin := cfg.UpgradeBin(upgrade)
		currentBin, err := cfg.CurrentBin()
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		t.Log(currentBin)
		t.Log(upgradeBin)

		if upgradeBin != currentBin {
			t.Error(err)
			t.FailNow()
		}
	}
}
