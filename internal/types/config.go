package types

import (
	"net/url"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	rootName    = "runner"
	genesisDir  = "genesis"
	upgradesDir = "upgrades"
	currentLink = "current"
)

// Config is the information passed in to control the daemon
type Config struct {
	Home                string
	Name                string
	RestartAfterUpgrade bool
}

// Root returns the root directory where all info lives
func (cfg *Config) Root() string {
	return filepath.Join(cfg.Home, rootName)
}

// GenesisBin is the path to the genesis binary - must be in place to start manager
func (cfg *Config) GenesisBin() string {
	return filepath.Join(cfg.Root(), genesisDir, "bin", cfg.Name)
}

// UpgradeBin is the path to the binary for the named upgrade
func (cfg *Config) UpgradeBin(upgradeName string) string {
	return filepath.Join(cfg.UpgradeDir(upgradeName), "bin", cfg.Name)
}

// UpgradeDir is the directory named upgrade
func (cfg *Config) UpgradeDir(upgradeName string) string {
	safeName := url.PathEscape(upgradeName)
	return filepath.Join(cfg.Root(), upgradesDir, safeName)
}

// Symlink to genesis
func (cfg *Config) SymLinkToGenesis() (string, error) {
	genesis := filepath.Join(cfg.Root(), genesisDir)
	link := filepath.Join(cfg.Root(), currentLink)

	if err := os.Symlink(genesis, link); err != nil {
		return "", err
	}
	// and return the genesis binary
	return cfg.GenesisBin(), nil
}

// CurrentBin is the path to the currently selected binary (genesis if no link is set)
// This will resolve the symlink to the underlying directory to make it easier to debug
func (cfg *Config) CurrentBin() (string, error) {
	cur := filepath.Join(cfg.Root(), currentLink)
	// if nothing here, fallback to genesis
	info, err := os.Lstat(cur)
	if err != nil {
		//Create symlink to the genesis
		return cfg.SymLinkToGenesis()
	}
	// if it is there, ensure it is a symlink
	if info.Mode()&os.ModeSymlink == 0 {
		//Create symlink to the genesis
		return cfg.SymLinkToGenesis()
	}

	// resolve it
	dest, err := os.Readlink(cur)
	if err != nil {
		//Create symlink to the genesis
		return cfg.SymLinkToGenesis()
	}

	// and return the binary
	dest = filepath.Join(dest, "bin", cfg.Name)
	return dest, nil
}

// GetConfigFromEnv will read the environmental variables into a config
// and then Validate it is reasonable
func GetConfigFromEnv() (*Config, error) {
	cfg := &Config{
		Home: os.Getenv("DAEMON_HOME"),
		Name: os.Getenv("DAEMON_NAME"),
	}
	if os.Getenv("DAEMON_RESTART_AFTER_UPGRADE") == "on" {
		cfg.RestartAfterUpgrade = true
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate returns an error if this config is invalid.
// it enforces Home/upgrade_manager is a valid directory and exists,
// and that Name is set
func (cfg *Config) Validate() error {
	if cfg.Name == "" {
		return errors.New("DAEMON_NAME is not set")
	}
	if cfg.Home == "" {
		return errors.New("DAEMON_HOME is not set")
	}

	if !filepath.IsAbs(cfg.Home) {
		return errors.New("DAEMON_HOME must be an absolute path")
	}

	// ensure the root directory exists
	info, err := os.Stat(cfg.Root())
	if err != nil {
		return errors.Wrap(err, "cannot stat home dir")
	}
	if !info.IsDir() {
		return errors.Errorf("%s is not a directory", info.Name())
	}

	return nil
}

// SetCurrentUpgrade sets the named upgrade to be the current link, returns error if this binary doesn't exist
func (cfg *Config) SetCurrentUpgrade(upgradeName string) error {
	// set a symbolic link
	link := filepath.Join(cfg.Root(), currentLink)
	safeName := url.PathEscape(upgradeName)
	upgrade := filepath.Join(cfg.Root(), upgradesDir, safeName)

	// remove link if it exists
	if _, err := os.Stat(link); err == nil {
		os.Remove(link)
	}

	// point to the new directory
	if err := os.Symlink(upgrade, link); err != nil {
		return errors.Wrap(err, "creating current symlink")
	}
	return nil
}
