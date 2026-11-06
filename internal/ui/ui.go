// Package ui is revshell's TUI: search the catalog, edit the ip/port/shell
// in the settings popup, copy the rendered one-liner.
package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/sectui/foam/theme"

	"github.com/sectui/revshell/internal/config"
)

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	theme.Set(cfg.Theme)

	m, err := New(cfg)
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(m).Run()
	return err
}
