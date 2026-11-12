package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/anotherhadi/foam/notify"
	"github.com/anotherhadi/foam/theme"
)

// checkScreen asserts that the view fills w×h exactly. The renderer drops
// the trailing blanks of a line, so only a blank line may be shorter.
func checkScreen(t *testing.T, m *Model, w, h int, state string) {
	t.Helper()
	content := m.View().Content
	lines := strings.Split(content, "\n")
	if len(lines) != h {
		t.Fatalf("%s at %dx%d: %d lines", state, w, h, len(lines))
	}
	for i, l := range lines {
		if got := ansi.StringWidth(l); got > w || got < w && strings.TrimSpace(ansi.Strip(l)) != "" {
			t.Fatalf("%s at %dx%d: line %d is %d wide: %q", state, w, h, i, got, ansi.Strip(l))
		}
	}
}

func TestResponsive(t *testing.T) {
	sizes := [][2]int{
		{72, 20}, {72, 21}, {80, 20}, {83, 22}, {90, 25}, {99, 30}, {100, 24}, {101, 33},
		{120, 40}, {140, 40}, {157, 50}, {200, 60}, {300, 80},
		// below the minimum
		{71, 20}, {72, 19}, {40, 10}, {10, 3}, {1, 1}, {2, 200}, {300, 2},
	}
	states := []struct {
		name  string
		setup func(t *testing.T, m *Model)
	}{
		{"default", func(t *testing.T, m *Model) {}},
		{"search typing", func(t *testing.T, m *Model) {
			typeKeys(m, "/bash")
		}},
		{"no match", func(t *testing.T, m *Model) {
			typeKeys(m, "/zzzzzz")
		}},
		{"settings editing", func(t *testing.T, m *Model) {
			key(m, 'o', tea.ModCtrl)
			key(m, tea.KeyEnter, 0)
			key(m, 'u', tea.ModCtrl)
			typeKeys(m, "192.168.1.1")
		}},
		{"settings error", func(t *testing.T, m *Model) {
			key(m, 'o', tea.ModCtrl)
			key(m, 'j', 0)
			key(m, tea.KeyEnter, 0)
			key(m, 'u', tea.ModCtrl)
			typeKeys(m, "abc")
			key(m, tea.KeyEnter, 0)
		}},
		{"category filtered", func(t *testing.T, m *Model) { typeKeys(m, "cc") }},
		{"os filtered", func(t *testing.T, m *Model) { typeKeys(m, "o") }},
		{"encoding cycled", func(t *testing.T, m *Model) { typeKeys(m, "eee") }},
		{"multiline entry", func(t *testing.T, m *Model) {
			typeKeys(m, "/p0wny")
			key(m, tea.KeyEscape, 0)
		}},
		{"long values", func(t *testing.T, m *Model) {
			typeKeys(m, "/"+strings.Repeat("a", 100))
			key(m, 'o', tea.ModCtrl)
			key(m, 'j', 0)
			key(m, tea.KeyEnter, 0)
			key(m, 'u', tea.ModCtrl)
			typeKeys(m, strings.Repeat("1", 60))
		}},
		{"help", func(t *testing.T, m *Model) { typeKeys(m, "?") }},
		{"palette", func(t *testing.T, m *Model) { key(m, 'p', tea.ModCtrl) }},
		{"toasts", func(t *testing.T, m *Model) {
			for i := range 7 {
				m.Update(notify.Msg{Level: notify.Level(i % 4), Text: strings.Repeat("a long notification ", i+1)})
			}
		}},
		{"preview scrolled", func(t *testing.T, m *Model) {
			key(m, 'l', tea.ModCtrl)
			typeKeys(m, "jjjj")
		}},
		{"zoomed", func(t *testing.T, m *Model) {
			key(m, 'l', tea.ModCtrl)
			key(m, 'z', tea.ModAlt)
		}},
		{"preview focused", func(t *testing.T, m *Model) { key(m, '2', tea.ModAlt) }},
	}
	looks := []struct{ spacious, helpBar bool }{{false, true}, {true, true}, {true, false}, {false, false}}
	defer theme.Set(theme.Current())
	for _, look := range looks {
		th := theme.Current()
		th.Spacious, th.HelpBar = look.spacious, look.helpBar
		theme.Set(th)
		for _, st := range states {
			m := newModel(t)
			st.setup(t, m)
			for _, sz := range sizes {
				m.Update(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
				checkScreen(t, m, sz[0], sz[1], fmt.Sprintf("%s (spacious=%v helpBar=%v)", st.name, look.spacious, look.helpBar))
			}
		}
	}
}

// Below 100 columns the preview stacks under the list.
func TestPreviewStackedWhenNarrow(t *testing.T) {
	m := newModel(t)
	key(m, '2', tea.ModAlt) // to the preview
	m.Update(tea.WindowSizeMsg{Width: 99, Height: 30})
	if m.focus != panePreview {
		t.Fatalf("focus = %v", m.focus)
	}
	l, p := m.rects[paneList], m.rects[panePreview]
	if l.W != m.w || p.W != m.w || p.Y != l.Y+l.H {
		t.Fatalf("not stacked: %+v / %+v", l, p)
	}
	checkScreen(t, m, 99, 30, "narrow")
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	l, p = m.rects[paneList], m.rects[panePreview]
	if l.W == m.w || p.Y != l.Y {
		t.Fatalf("not side by side: %+v / %+v", l, p)
	}
}

func TestZoom(t *testing.T) {
	m := newModel(t)
	key(m, 'l', tea.ModCtrl)
	key(m, 'z', tea.ModAlt)
	body := m.rects[panePreview]
	if body.W != m.w || !m.rects[paneList].Empty() {
		t.Fatalf("not zoomed: %+v", m.rects)
	}
	// another pane takes the zoom
	key(m, '1', tea.ModAlt)
	if m.rects[paneList] != body || !m.rects[panePreview].Empty() {
		t.Fatalf("zoom did not follow the focus: %+v", m.rects)
	}
	checkScreen(t, m, m.w, m.h, "zoomed")
	key(m, 'z', tea.ModAlt)
	if m.rects[paneList].Empty() || m.rects[panePreview].Empty() {
		t.Fatalf("not back: %+v", m.rects)
	}
}

// A toast that would run past the bottom of the window is left out.
func TestToastsStayOnScreen(t *testing.T) {
	m := newModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	for range 5 {
		m.Update(notify.Msg{Text: strings.Repeat("a long notification ", 7)})
	}
	m.toasts()
	if len(m.toastRects) == 0 {
		t.Fatal("no toast")
	}
	for _, r := range m.toastRects {
		if b := r.Y + r.H; b > m.h {
			t.Fatalf("a toast ends at row %d of %d", b, m.h)
		}
	}
}

// Clicking a toast dismisses it.
func TestToastClick(t *testing.T) {
	m := newModel(t)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	m.Update(notify.Msg{Text: "copied"})
	m.toasts()
	if len(m.toastRects) == 0 {
		t.Fatal("no toast")
	}
	r := m.toastRects[0]
	do(m, tea.MouseClickMsg{X: r.X + 1, Y: r.Y, Button: tea.MouseLeft})
	if n := len(m.notes.Live()); n != 0 {
		t.Fatalf("toast not dismissed: %d live", n)
	}
}
