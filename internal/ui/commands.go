package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/sectui/foam/commands"
	"github.com/sectui/foam/form"
	"github.com/sectui/foam/geom"
	"github.com/sectui/foam/help"
	"github.com/sectui/foam/notify"
	"github.com/sectui/foam/overlay"
	"github.com/sectui/foam/palette"

	"github.com/sectui/revshell/internal/catalog"
)

// global are always active: the search field already got the printable keys.
func (m *Model) global() []commands.Command {
	focus := func(p pane) func() tea.Cmd {
		return func() tea.Cmd { m.setFocus(p); return nil }
	}
	nav := func(d geom.Dir) func() tea.Cmd {
		return func() tea.Cmd { m.navigate(d); return nil }
	}
	return []commands.Command{
		{ID: "app.quit", Title: "Quit", Help: "quit", Run: func() tea.Cmd { return tea.Quit }},
		{ID: "app.help", Title: "Show the shortcuts", Help: "help", Run: func() tea.Cmd {
			return overlay.Open(help.New(m.reg.Groups(m.path())))
		}},
		{ID: "app.palette", Title: "Command palette", Run: func() tea.Cmd {
			return overlay.Open(palette.New(m.reg.Active(m.path()), "app.palette"))
		}},
		{ID: "focus.list", Title: "Go to the list", Run: focus(paneList)},
		{ID: "focus.preview", Title: "Go to the preview", Run: focus(panePreview)},
		{ID: "pane.left", Title: "Go to the pane on the left", Run: nav(geom.Left)},
		{ID: "pane.down", Title: "Go to the pane below", Run: nav(geom.Down)},
		{ID: "pane.up", Title: "Go to the pane above", Run: nav(geom.Up)},
		{ID: "pane.right", Title: "Go to the pane on the right", Run: nav(geom.Right)},
		{ID: "pane.zoom", Title: "Zoom the pane in or out", Run: func() tea.Cmd {
			m.zoomed = !m.zoomed
			m.layout()
			return nil
		}},
		{ID: "view.search", Title: "Search the catalog", Help: "search", Run: func() tea.Cmd {
			m.searching = true
			m.setFocus(paneList)
			return nil
		}},
		{ID: "view.escape", Title: "Back to the list", Run: func() tea.Cmd {
			m.searching = false
			m.setFocus(paneList)
			return nil
		}},
		{ID: "view.category", Title: "Cycle the category filter", Help: "category", Run: func() tea.Cmd {
			m.category = (m.category + 1) % len(categoryFilters)
			m.filter()
			return nil
		}},
		{ID: "view.os", Title: "Cycle the OS filter", Help: "os", Run: func() tea.Cmd {
			m.os = (m.os + 1) % len(osFilters)
			m.filter()
			return nil
		}},
		{ID: "view.encoding", Title: "Cycle the output encoding", Help: "encoding", Run: func() tea.Cmd {
			m.encoding = (m.encoding + 1) % (catalog.Base64Encoding + 1)
			return nil
		}},
		{ID: "view.copy", Title: "Copy the rendered command", Help: "copy", Run: m.copy},
		{ID: "settings.open", Title: "Edit the ip, port and shell", Help: "settings", Run: func() tea.Cmd {
			return overlay.Open(form.New("Settings", &m.vars))
		}},
	}
}

// path is the chain of scopes for the focused pane, outermost first.
func (m *Model) path() []commands.Scope {
	s := []commands.Scope{{Name: "revshell", Commands: m.global()}}
	switch m.focus {
	case paneList:
		s = append(s, commands.Scope{Name: "List", Commands: m.list.Commands()})
	case panePreview:
		s = append(s, commands.Scope{Name: "Preview", Commands: m.preview.Commands()})
	}
	return s
}

// allScopes lists every scope, for checking the keys at startup.
func (m *Model) allScopes() []commands.Scope {
	return []commands.Scope{
		{Name: "revshell", Commands: m.global()},
		{Name: "List", Commands: m.list.Commands()},
		{Name: "Preview", Commands: m.preview.Commands()},
		help.New(nil).Scope(),
	}
}

func (m *Model) navigate(d geom.Dir) {
	var cands []geom.Rect
	var panes []pane
	for p := range paneCount {
		if p != m.focus && !m.rects[p].Empty() {
			cands = append(cands, m.rects[p])
			panes = append(panes, p)
		}
	}
	if i := geom.Neighbor(m.rects[m.focus], cands, d); i >= 0 {
		m.setFocus(panes[i])
	}
}

func (m *Model) copy() tea.Cmd {
	e, ok := m.selected()
	if !ok {
		return nil
	}
	text := catalog.Render(e, m.renderVars(), m.encoding)
	return tea.Batch(tea.SetClipboard(text), notify.Success("Copied "+e.Name))
}
