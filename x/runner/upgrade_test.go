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

		if upgradeBin != currentBin {
			t.Error(err)
			t.FailNow()
		}
	}
}

func TestDownloadBinary(t *testing.T) {
	home, err := copyTestData("validate")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer os.RemoveAll(home)

	cfg := &types.Config{Home: home, Name: "test-runnerd"}

	type args struct {
		cfg  *types.Config
		info *types.UpgradeInfo
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "defaultTest",
			args: args{cfg: cfg, info: &types.UpgradeInfo{
				Name:    "RC-0.2.1",
				Version: "RC-0.2.1",
			}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DownloadBinary(tt.args.cfg, tt.args.info); (err != nil) != tt.wantErr {
				t.Errorf("DownloadBinary() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
