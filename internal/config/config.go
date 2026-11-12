// Package config loads revshell's configuration: the shared foam theme and
// its own keys, from the same layered files as operations and steep.
package config

import (
	_ "embed"
	"path/filepath"

	"github.com/anotherhadi/foam"
	"github.com/anotherhadi/foam/commands"
	fconfig "github.com/anotherhadi/foam/config"
	"github.com/anotherhadi/foam/theme"
)

//go:embed default.yaml
var Defaults []byte

type Config struct {
	Theme theme.Theme       `koanf:",squash"`
	Keys  commands.Bindings `koanf:"keys"`
}

func Load() (*Config, error) {
	cfg, _, err := fconfig.Load[Config](fconfig.Options{
		Name: "revshell",
		Base: [][]byte{foam.Defaults, Defaults},
		Files: []string{
			filepath.Join(fconfig.Dir("foam"), "config.yaml"),
			filepath.Join(fconfig.Dir("revshell"), "config.yaml"),
		},
	}, nil)
	return cfg, err
}
