package ui

import (
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/sectui/foam"
	fconfig "github.com/sectui/foam/config"
	"github.com/sectui/foam/theme"

	"github.com/sectui/revshell/internal/config"
)

func init() { log.SetOutput(io.Discard) }

func newModel(t *testing.T) *Model {
	t.Helper()
	cfg, err := fconfig.LoadBytes[config.Config](foam.Defaults, config.Defaults)
	if err != nil {
		t.Fatal(err)
	}
	theme.Set(cfg.Theme)
	m, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	return m
}

// do runs msg through Update, then the commands it returns, as bubbletea
// would.
func do(m *Model, msg tea.Msg) {
	_, cmd := m.Update(msg)
	drain(m, cmd, 0)
}

func drain(m *Model, cmd tea.Cmd, depth int) {
	if cmd == nil || depth > 20 {
		return
	}
	msg := cmd()
	switch msg := msg.(type) {
	case tea.BatchMsg:
		for _, c := range msg {
			drain(m, c, depth+1)
		}
	case nil:
	default:
		if v := reflect.ValueOf(msg); v.Kind() == reflect.Slice && v.Type().Elem() == reflect.TypeFor[tea.Cmd]() {
			for i := range v.Len() {
				drain(m, v.Index(i).Interface().(tea.Cmd), depth+1)
			}
			return
		}
		_, next := m.Update(msg)
		drain(m, next, depth+1)
	}
}

func typeKeys(m *Model, s string) {
	for _, r := range s {
		do(m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func key(m *Model, code rune, mod tea.KeyMod) { do(m, tea.KeyPressMsg{Code: code, Mod: mod}) }

func screen(m *Model) string { return ansi.Strip(m.View().Content) }

func TestPaneNavigation(t *testing.T) {
	m := newModel(t)
	if m.focus != paneList {
		t.Fatalf("initial focus = %v", m.focus)
	}
	key(m, 'l', tea.ModCtrl) // right: to the preview
	if m.focus != panePreview {
		t.Fatalf("ctrl+l: focus = %v", m.focus)
	}
	key(m, 'h', tea.ModCtrl) // left: back to the list
	if m.focus != paneList {
		t.Fatalf("ctrl+h: focus = %v", m.focus)
	}
	key(m, '2', tea.ModAlt)
	if m.focus != panePreview {
		t.Fatalf("alt+2: focus = %v", m.focus)
	}
	key(m, '1', tea.ModAlt)
	if m.focus != paneList {
		t.Fatalf("alt+1: focus = %v", m.focus)
	}
	key(m, tea.KeyEscape, 0)
	if m.focus != paneList {
		t.Fatalf("esc: focus = %v", m.focus)
	}
}

func TestPaneKeysWinOverSearch(t *testing.T) {
	m := newModel(t)
	typeKeys(m, "/bash") // searching: the query goes to the field
	if !m.searching {
		t.Fatal("not searching")
	}
	if len(m.shown) == 0 {
		t.Fatal("search did not filter")
	}
	key(m, 'l', tea.ModCtrl) // right: to the preview, not into the text
	if m.focus != panePreview {
		t.Fatalf("ctrl+l: focus = %v", m.focus)
	}
	if got := m.search.Value(); got != "bash" {
		t.Fatalf("search = %q", got)
	}
	key(m, 'h', tea.ModCtrl) // back to the list
	// ctrl+h is the pane on the left, not the field's backspace
	if m.focus != paneList || m.search.Value() != "bash" {
		t.Fatalf("ctrl+h: focus %v, search %q", m.focus, m.search.Value())
	}
}

func TestSearchAndFilters(t *testing.T) {
	m := newModel(t)
	total := len(m.all)
	if !strings.Contains(screen(m), "Shells (") {
		t.Fatalf("list pane:\n%s", screen(m))
	}
	typeKeys(m, "/bash")
	if len(m.shown) == 0 || len(m.shown) >= total {
		t.Fatalf("search did not filter: %d/%d", len(m.shown), total)
	}
	if !strings.Contains(screen(m), "bash") {
		t.Fatalf("search not shown:\n%s", screen(m))
	}
	// the selection stays while the query narrows
	name := m.shown[m.list.Selected()].Name
	typeKeys(m, "i")
	if got := m.shown[m.list.Selected()].Name; got != name {
		t.Fatalf("selection lost: %q -> %q", name, got)
	}
	key(m, tea.KeyEscape, 0) // stop searching
	if m.searching {
		t.Fatal("esc did not stop the search")
	}
	key(m, 'c', 0) // cycle the category
	if categoryFilters[m.category] == "" {
		t.Fatal("category did not cycle")
	}
	key(m, 'o', 0) // cycle the os
	if osFilters[m.os] == "" {
		t.Fatal("os did not cycle")
	}
	key(m, 'e', 0) // cycle the encoding
	if m.encoding == 0 {
		t.Fatal("encoding did not cycle")
	}
	// the rendering follows
	e, _ := m.selected()
	if !strings.Contains(screen(m), e.Name) {
		t.Fatalf("preview does not follow:\n%s", screen(m))
	}
}

func TestCopy(t *testing.T) {
	m := newModel(t)
	typeKeys(m, "y")
	if h := fmt.Sprint(m.notes.History()); !strings.Contains(h, "Copied") {
		t.Fatalf("no confirmation: %s", h)
	}
}

func TestPalette(t *testing.T) {
	m := newModel(t)
	key(m, 'p', tea.ModCtrl)
	if m.overlays.Top() == nil {
		t.Fatal("no palette")
	}
	typeKeys(m, "encoding")
	key(m, tea.KeyEnter, 0)
	if m.overlays.Top() != nil {
		t.Fatal("palette did not close")
	}
	if m.encoding == 0 {
		t.Fatal("the command did not run")
	}
	if !strings.Contains(screen(m), "url") && !strings.Contains(screen(m), "encoding: url") {
		t.Fatalf("status line:\n%s", screen(m))
	}
}

func TestHelp(t *testing.T) {
	m := newModel(t)
	typeKeys(m, "?")
	if m.overlays.Top() == nil || !strings.Contains(screen(m), "Help") {
		t.Fatalf("no help overlay:\n%s", screen(m))
	}
	key(m, tea.KeyEscape, 0)
	if m.overlays.Top() != nil {
		t.Fatal("help did not close")
	}
}

func TestSettings(t *testing.T) {
	m := newModel(t)
	key(m, 'o', tea.ModCtrl) // the settings popup
	if m.overlays.Top() == nil {
		t.Fatal("no settings overlay")
	}
	if !strings.Contains(screen(m), "Settings") {
		t.Fatalf("settings not drawn:\n%s", screen(m))
	}
	key(m, tea.KeyEnter, 0)  // edit the ip field
	key(m, 'u', tea.ModCtrl) // clear the value typed in
	typeKeys(m, "192.168.1.1")
	key(m, tea.KeyEnter, 0) // submit
	if m.vars.IP != "192.168.1.1" {
		t.Fatalf("ip = %q", m.vars.IP)
	}
	// the preview follows the value
	if !strings.Contains(screen(m), "192.168.1.1") {
		t.Fatalf("preview did not follow:\n%s", screen(m))
	}
	// back to the default
	key(m, 'd', 0)
	if m.vars.IP != "10.10.10.10" {
		t.Fatalf("reset: ip = %q", m.vars.IP)
	}
	key(m, tea.KeyEscape, 0) // close
	if m.overlays.Top() != nil {
		t.Fatal("settings did not close")
	}
	// an invalid port is refused, with the error shown
	key(m, 'o', tea.ModCtrl)
	key(m, 'j', 0) // to the port field
	key(m, tea.KeyEnter, 0)
	key(m, 'u', tea.ModCtrl) // clear the value typed in
	typeKeys(m, "abc")
	key(m, tea.KeyEnter, 0)
	if m.vars.Port != "9001" {
		t.Fatalf("invalid port kept: %q", m.vars.Port)
	}
	if !strings.Contains(screen(m), "port must be 1-65535") {
		t.Fatalf("no error shown:\n%s", screen(m))
	}
	key(m, tea.KeyEscape, 0) // close
	if m.overlays.Top() != nil {
		t.Fatal("settings did not close")
	}
}

func TestSearchClick(t *testing.T) {
	m := newModel(t)
	r := m.rects[paneList]
	key(m, '/', 0) // the search row appears
	// the search row is the first line of the list pane's content
	do(m, tea.MouseClickMsg{X: r.X + 2, Y: r.Y + 1, Button: tea.MouseLeft})
	if !m.searching {
		t.Fatal("click on the search row: not searching")
	}
	typeKeys(m, "bash")
	if got := m.search.Value(); got != "bash" {
		t.Fatalf("search = %q", got)
	}
	// a click below the rule selects an entry and keeps the query
	do(m, tea.MouseClickMsg{X: r.X + 2, Y: r.Y + 4, Button: tea.MouseLeft})
	if !m.searching {
		t.Fatal("entry click stopped the search")
	}
	if m.list.Selected() != 1 {
		t.Fatalf("entry click: selected %d", m.list.Selected())
	}
	// with the query cleared the search row goes away and the first
	// content line selects again
	m.search.Reset()
	m.filter()
	m.searching = false
	m.layout()
	do(m, tea.MouseClickMsg{X: r.X + 2, Y: r.Y + 1, Button: tea.MouseLeft})
	if m.list.Selected() != 0 {
		t.Fatalf("list click: selected %d", m.list.Selected())
	}
}

func TestPreviewScroll(t *testing.T) {
	m := newModel(t)
	typeKeys(m, "ccccc") // to the shellcode entries: long, wrapped commands
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 20})
	key(m, 'l', tea.ModCtrl)
	key(m, 'j', 0)
	if m.preview.Scroll().Offset == 0 {
		t.Fatal("the preview did not scroll")
	}
}
