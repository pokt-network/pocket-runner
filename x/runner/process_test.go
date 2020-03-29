package runner

import (
	"os"
	"testing"

	"github.com/pokt-network/pocket-runner/internal/types"
)

func TestLaunchProcess(t *testing.T) {
	home, err := copyTestData("validate")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	cfg := &types.Config{Home: home, Name: "custom-core"}
	defer os.RemoveAll(home)

	args := []string{"start", "--blockTime", "1"} // NOTE add short block times for testing purposes
	_, err = LaunchProcess(cfg, args)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
