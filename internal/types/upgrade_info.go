package types

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// UpgradeInfo is the details from the regexp
type UpgradeInfo struct {
	Name    string
	Height  int64
	Version string
}

func (ui *UpgradeInfo) SetUpgrade(s string) error {
	tx := strings.Split(s, " ")
	height, err := strconv.Atoi(strings.TrimSuffix(tx[5], "]"))
	if err != nil {
		errors.Wrapf(err, "could not convert string: %s to integer", tx[5])
	}
	ui.Name = tx[2]
	ui.Height = int64(height)
	ui.Version = tx[2]
	return nil
}
