package runner

import (
	"io"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/pokt-network/pocket-runner/internal/types"
)

// LaunchProcess runs a subprocess and returns when the subprocess exits,
// either when it dies, or *after* a successful upgrade.
func LaunchProcess(cfg *types.Config, args []string, stdout, stderr io.Writer, stdin io.Reader) (bool, error) {
	bin, err := cfg.CurrentBin()
	if err != nil {
		return false, errors.Wrap(err, "error creating symlink to genesis")
	}
	cmd := exec.Command(bin, args...)

	// NOTE visibility into the process
	cmd.Stdout = stdout
	cmd.Stdin = stdin
	cmd.Stderr = stderr

	err = cmd.Start()
	if err != nil {
		return false, errors.Wrapf(err, "problem running command %s", cmd.String())
	}

	return false, nil
}
