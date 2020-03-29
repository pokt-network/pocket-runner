package types

import (
	"os"

	"github.com/pkg/errors"
)

// EnsureBinary ensures the file exists and is executable, or returns an error
func CheckBinary(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return errors.Wrap(err, "cannot stat home dir")
	}
	if !info.Mode().IsRegular() {
		return errors.Errorf("%s is not a regular file", info.Name())
	}
	// this checks if the world-executable bit is set (we cannot check owner easily)
	exec := info.Mode().Perm() & 0001
	if exec == 0 {
		return errors.Errorf("%s is not world executable", info.Name())
	}
	return nil
}
