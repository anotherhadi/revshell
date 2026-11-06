// Package config loads revshell's configuration: the shared sectui theme and
// its own keys, from the same layered files as operations and steep.
package config

import (
	_ "embed"
	"path/filepath"

	"github.com/sectui/foam"
	"github.com/sectui/foam/commands"
	fconfig "github.com/sectui/foam/config"
	"github.com/sectui/foam/theme"
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
			filepath.Join(fconfig.Dir("sectui"), "config.yaml"),
			filepath.Join(fconfig.Dir("revshell"), "config.yaml"),
		},
	}, nil)
	return cfg, err
}
