package runner

import (
	"bytes"
	"os"
	"testing"

	"github.com/pokt-network/pocket-runner/config"
)

func TestLaunchProcess(t *testing.T) {
	home, err := copyTestData("validate")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	cfg := &config.Config{Home: home, Name: "test-runnerd"}
	defer os.RemoveAll(home)

	var stdout, stderr bytes.Buffer
	args := []string{"-blockTime", "1"} // NOTE add short block times for testing purposes
	_, err = LaunchProcess(cfg, args, &stdout, &stderr)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
