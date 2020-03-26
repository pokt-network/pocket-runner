package config

import (
	"path/filepath"
	"testing"
)

func TestConfigPaths(t *testing.T) {
	cases := map[string]struct {
		cfg           Config
		upgradeName   string
		expectRoot    string
		expectGenesis string
		expectUpgrade string
	}{
		"simple": {
			cfg:           Config{Home: "/foo", Name: "myd"},
			upgradeName:   "bar",
			expectRoot:    "/foo/runner",
			expectGenesis: "/foo/runner/genesis/bin/myd",
			expectUpgrade: "/foo/runner/upgrades/bar/bin/myd",
		},
		"handle space": {
			cfg:           Config{Home: "/longer/prefix/", Name: "yourd"},
			upgradeName:   "some spaces",
			expectRoot:    "/longer/prefix/runner",
			expectGenesis: "/longer/prefix/runner/genesis/bin/yourd",
			expectUpgrade: "/longer/prefix/runner/upgrades/some%20spaces/bin/yourd",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if gotRoot := tc.cfg.Root(); gotRoot != filepath.FromSlash(tc.expectRoot) {
				t.Fail()
			}
			if gotGenesis := tc.cfg.GenesisBin(); gotGenesis != filepath.FromSlash(tc.expectGenesis) {
				t.Fail()
			}
			if gotUpgrade := tc.cfg.UpgradeBin(tc.upgradeName); gotUpgrade != filepath.FromSlash(tc.expectUpgrade) {
				t.Fail()
			}
		})
	}
}

// Test validate
func TestValidate(t *testing.T) {
	relPath := filepath.Join("testdata", "validate")
	absPath, err := filepath.Abs(relPath)
	if err != nil {
		t.Error(err)
	}

	testdata, err := filepath.Abs("testdata")
	if err != nil {
		t.Error(err)
	}

	cases := map[string]struct {
		cfg   Config
		valid bool
	}{
		"happy": {
			cfg:   Config{Home: absPath, Name: "bind"},
			valid: true,
		},
		"missing home": {
			cfg:   Config{Name: "bind"},
			valid: false,
		},
		"missing name": {
			cfg:   Config{Home: absPath},
			valid: false,
		},
		"relative path": {
			cfg:   Config{Home: relPath, Name: "bind"},
			valid: false,
		},
		"no upgrade manager subdir": {
			cfg:   Config{Home: testdata, Name: "bind"},
			valid: false,
		},
		"no such dir": {
			cfg:   Config{Home: filepath.FromSlash("/no/such/dir"), Name: "bind"},
			valid: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := tc.cfg.Validate()
			switch tc.valid {
			case true:
				if err != nil {
				}
			default:
				if err == nil {
					t.Fail()
				}
			}
		})
	}
}
