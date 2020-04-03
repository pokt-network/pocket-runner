package runner

import (
	"github.com/pokt-network/pocket-runner/internal/types"
	"os"
	"testing"
)

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
