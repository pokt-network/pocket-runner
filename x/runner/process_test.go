package runner

import (
	"bytes"
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
	var stdout, stderr, stdin bytes.Buffer

	args := []string{"start", "--blockTime", "1"} // NOTE add short block times for testing purposes
	_, err, _ = LaunchProcess(cfg, args, &stdout, &stderr, &stdin)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
