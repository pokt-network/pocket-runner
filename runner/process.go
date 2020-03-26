package runner

import (
	"io"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/pokt-network/pocket-runner/config"
)

// LaunchProcess runs a subprocess and returns when the subprocess exits,
// either when it dies, or *after* a successful upgrade.
func LaunchProcess(cfg *config.Config, args []string, stdout, stderr io.Writer) (bool, error) {
	bin, err := cfg.CurrentBin()
	if err != nil {
		return false, errors.Wrap(err, "error creating symlink to genesis")
	}
	cmd := exec.Command(bin, args...)

	err = cmd.Start()
	if err != nil {
		return false, errors.Wrapf(err, "launching process %s %s", bin, strings.Join(args, " "))
	}
	return false, nil
}
