// Package ui is revshell's root model: the panes, the layout, the keys.
package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/anotherhadi/foam/commands"
	"github.com/anotherhadi/foam/fuzzy"
	"github.com/anotherhadi/foam/geom"
	"github.com/anotherhadi/foam/input"
	"github.com/anotherhadi/foam/list"
	"github.com/anotherhadi/foam/notify"
	"github.com/anotherhadi/foam/overlay"
	"github.com/anotherhadi/foam/viewer"

	"github.com/anotherhadi/revshell/internal/catalog"
	"github.com/anotherhadi/revshell/internal/config"
	"github.com/anotherhadi/revshell/internal/ui/settings"
)

type pane int

const (
	paneList pane = iota
	panePreview
	paneCount
)

var paneNames = [paneCount]string{"Shells", "Preview"}

// categoryFilters and osFilters cycle with view.category and view.os; "" is
// "no filter".
var categoryFilters = append([]catalog.Category{""}, catalog.Categories...)
var osFilters = []string{"", "windows", "linux", "mac", "android", "apple_ios"}

type Model struct {
	cfg *config.Config
	reg *commands.Registry

	all   []catalog.Entry
	shown []catalog.Entry

	search    input.Model
	searching bool

	vars settings.Vars

	category int
	os       int
	encoding catalog.Encoding

	list    list.Model
	preview viewer.Model

	focus  pane
	zoomed bool

	overlays overlay.Stack
	notes    notify.Center

	shownPreview string // the preview's content, to redraw it only on change

	w, h       int
	rects      [paneCount]geom.Rect
	toastRects []toastRect
}

type toastRect struct {
	geom.Rect
	id int
}

func New(cfg *config.Config) (*Model, error) {
	reg, err := commands.New(cfg.Keys)
	if err != nil {
		return nil, fmt.Errorf("invalid config:\n%w", err)
	}

	m := &Model{cfg: cfg, reg: reg, all: catalog.All(), focus: paneList, vars: settings.Default()}
	m.search = input.New("/ ")
	m.search.Placeholder = "search"
	m.preview.Wrap = true
	m.notes.Max = 100
	m.filter()

	if err := reg.Check(m.allScopes()); err != nil {
		return nil, fmt.Errorf("invalid config:\n%w", err)
	}
	return m, nil
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) renderVars() catalog.Vars {
	port, err := strconv.Atoi(m.vars.Port)
	if err != nil || port <= 0 || port > 0xFFFF {
		port = 9001
	}
	return catalog.Vars{IP: m.vars.IP, Port: port, Shell: m.vars.Shell}
}

func (m *Model) selected() (catalog.Entry, bool) {
	if len(m.shown) == 0 {
		return catalog.Entry{}, false
	}
	return m.shown[m.list.Selected()], true
}

func (m *Model) filter() {
	name := ""
	if e, ok := m.selected(); ok {
		name = e.Name
	}
	texts := make([]string, 0, len(m.all))
	var candidates []catalog.Entry
	for _, e := range m.all {
		if cat := categoryFilters[m.category]; cat != "" && e.Category != cat {
			continue
		}
		if os := osFilters[m.os]; os != "" && !hasTag(e.Tags, os) {
			continue
		}
		candidates = append(candidates, e)
		texts = append(texts, catalog.SearchText(e))
	}
	m.shown = m.shown[:0]
	for _, i := range fuzzy.Filter(m.search.Value(), texts) {
		m.shown = append(m.shown, candidates[i])
	}
	m.list.SetLen(len(m.shown))
	// Keep the selection on the same entry when it is still shown.
	for i, e := range m.shown {
		if e.Name == name {
			m.list.Select(i)
			return
		}
	}
	m.list.Select(0)
}

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// Capturing: the search field takes the printable keys before any command.
func (m *Model) Capturing() bool { return m.searching }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := m.update(msg)
	return m, tea.Batch(cmd, m.afterUpdate())
}

func (m *Model) update(msg tea.Msg) tea.Cmd {
	if m.overlays.Handle(msg) {
		return nil
	}
	if cmd, ok := m.notes.Update(msg); ok {
		return cmd
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.layout()
	case tea.KeyPressMsg:
		return m.key(msg)
	case tea.PasteMsg:
		return m.paste(msg)
	case tea.MouseMsg:
		return m.mouse(msg)
	}
	return nil
}

// afterUpdate keeps the preview and the layout in step.
func (m *Model) afterUpdate() tea.Cmd {
	m.syncPreview()
	m.layout()
	return nil
}

func (m *Model) key(msg tea.KeyPressMsg) tea.Cmd {
	if !minSize.Fits(m.w, m.h) {
		// nothing to act on but quitting
		if b, ok := m.reg.Match(m.path()[:1], msg); ok && b.ID == "app.quit" {
			return b.Run()
		}
		return nil
	}
	if m.overlays.Top() != nil {
		return m.overlays.Key(m.reg, msg)
	}
	if m.Capturing() {
		// pane moves win over the search's ctrl+h/ctrl+k
		if b, ok := m.reg.Match(m.path(), msg); ok && strings.HasPrefix(b.ID, "pane.") {
			return b.Run()
		}
		if m.search.Update(msg) {
			m.filter()
			return nil
		}
	}
	if b, ok := m.reg.Match(m.path(), msg); ok {
		return b.Run()
	}
	return nil
}

func (m *Model) paste(msg tea.PasteMsg) tea.Cmd {
	if o := m.overlays.Top(); o != nil {
		return o.Update(msg)
	}
	if m.searching && m.search.Update(msg) {
		m.filter()
	}
	return nil
}

func (m *Model) mouse(msg tea.MouseMsg) tea.Cmd {
	if !minSize.Fits(m.w, m.h) {
		return nil
	}
	if m.overlays.Top() != nil {
		return m.overlays.Mouse(msg)
	}
	mm := msg.Mouse()
	if _, click := msg.(tea.MouseClickMsg); click {
		for _, t := range m.toastRects {
			if t.Contains(mm.X, mm.Y) {
				m.notes.DismissID(t.id)
				return nil
			}
		}
	}
	for p := range paneCount {
		r := m.rects[p]
		if r.Empty() || !r.Contains(mm.X, mm.Y) {
			continue
		}
		if _, click := msg.(tea.MouseClickMsg); click {
			m.setFocus(p)
		}
		mm.X, mm.Y = mm.X-r.X-1, mm.Y-r.Y-1
		if p == paneList && m.searchShown() {
			// the search row's click types in it
			if _, click := msg.(tea.MouseClickMsg); click && mm.Y < 2 {
				m.searching = true
				return nil
			}
			mm.Y -= 2 // skip the search row and its rule
		}
		var local tea.MouseMsg
		switch msg.(type) {
		case tea.MouseClickMsg:
			local = tea.MouseClickMsg(mm)
		case tea.MouseWheelMsg:
			local = tea.MouseWheelMsg(mm)
		default:
			return nil
		}
		switch p {
		case paneList:
			m.list.Mouse(local)
		case panePreview:
			m.preview.Mouse(local)
		}
		return nil
	}
	return nil
}

// setFocus moves the focus to p.
func (m *Model) setFocus(p pane) {
	m.focus = p
	m.layout()
}
