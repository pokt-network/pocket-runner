package runner

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"github.com/pokt-network/pocket-runner/config"
)

// copyTestData will make a tempdir and then
// "cp -r" a subdirectory under testdata there
// returns the directory (which can now be used as config.Config.Home) and modified safely
func copyTestData(subdir string) (string, error) {
	tmpdir, err := ioutil.TempDir("", "pocket-runner-test")
	if err != nil {
		return "", errors.Wrap(err, "create temp dir")
	}

	src := filepath.Join("testdata", subdir)

	options := Options{
		Recursive: true,
		Atomic:    true,
	}
	err = Copy(src, tmpdir, options)
	if err != nil {
		os.RemoveAll(tmpdir)
		return "", errors.Wrap(err, "copying files")
	}
	return tmpdir, nil
}

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
