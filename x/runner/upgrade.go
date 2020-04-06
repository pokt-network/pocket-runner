package runner

import (
	"github.com/pkg/errors"
	"github.com/pokt-network/pocket-runner/internal/types"
)

// Upgrade will be called after the log message has been parsed and the process has terminated.
// We can now make any changes to the underlying directory without interferance and leave it
// in a state, so we can make a proper restart
func Upgrade(cfg *types.Config, info *types.UpgradeInfo) error {
	err := types.CheckBinary(cfg.UpgradeBin(info.Name))

	// Simplest case is to switch the link
	if err != nil {
		return errors.Wrapf(err, "No binary available for upgrade")
	}
	// we have the binary - do it
	return cfg.SetCurrentUpgrade(info.Name)
}
