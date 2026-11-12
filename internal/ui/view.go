package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/anotherhadi/foam/draw"
	"github.com/anotherhadi/foam/geom"
	"github.com/anotherhadi/foam/minsize"
	"github.com/anotherhadi/foam/theme"
	"github.com/anotherhadi/foam/viewer"

	"github.com/anotherhadi/revshell/internal/catalog"
)

// minSize is the smallest terminal revshell draws in.
var minSize = minsize.Size{W: 72, H: 20}

const sideBySideBelow = 100

// categoryWidth is how categoryLabel is padded so the list column aligns.
const categoryWidth = 9

func categoryLabel(c catalog.Category) string {
	switch c {
	case "":
		return "all"
	case catalog.ReverseShell:
		return "reverse"
	case catalog.BindShell:
		return "bind"
	case catalog.MSFVenom:
		return "msfvenom"
	case catalog.HoaxShell:
		return "hoaxshell"
	case catalog.Shellcode:
		return "shellcode"
	default:
		return string(c)
	}
}

func osLabel(tag string) string {
	switch tag {
	case "":
		return "all"
	case "apple_ios":
		return "iOS"
	case "mac":
		return "macOS"
	default:
		return tag
	}
}

func encodingLabel(e catalog.Encoding) string {
	switch e {
	case catalog.URLEncoding:
		return "url"
	case catalog.URLEncodingDouble:
		return "url2"
	case catalog.Base64Encoding:
		return "base64"
	default:
		return "none"
	}
}

// sideBySide: the panes sit next to each other; narrower, the preview
// stacks under the list.
func (m *Model) sideBySide() bool { return m.w >= sideBySideBelow }

// searchShown reports whether the search row is on screen.
func (m *Model) searchShown() bool { return m.searching || m.search.Value() != "" }

// layout places the panes between the top bar and the help bar.
func (m *Model) layout() {
	m.rects = [paneCount]geom.Rect{}
	if m.w == 0 || m.h == 0 {
		return
	}
	top, bottom := 1, 0
	if theme.HelpBar() {
		bottom = 1
	}
	if theme.Spacious() {
		top++
		if bottom > 0 {
			bottom++
		}
	}
	body := geom.Rect{X: 0, Y: top, W: m.w, H: max(m.h-top-bottom, 0)}
	if m.sideBySide() {
		cols := body.Columns(geom.Ratio(2), geom.Ratio(3))
		m.rects[paneList], m.rects[panePreview] = cols[0], cols[1]
	} else {
		rows := body.Rows(geom.Ratio(3), geom.Ratio(2))
		m.rects[paneList], m.rects[panePreview] = rows[0], rows[1]
	}
	if m.rects[m.focus].Empty() {
		m.focus = paneList
	}
	if m.zoomed {
		m.rects = [paneCount]geom.Rect{}
		m.rects[m.focus] = body
	}
	r := m.rects[panePreview]
	m.preview.SetSize(max(r.W-2, 0), max(r.H-2, 0))
	extra := 0
	if m.searchShown() {
		extra = 2 // the search row and its rule
	}
	m.list.SetHeight(max(m.rects[paneList].H-2-extra, 0))
}

func (m *Model) View() tea.View {
	v := tea.NewView("")
	v.AltScreen = true
	v.WindowTitle = "revshell"
	if theme.Mouse() {
		v.MouseMode = tea.MouseModeCellMotion
	}
	if m.w == 0 || m.h == 0 {
		return v
	}
	if !minSize.Fits(m.w, m.h) {
		v.SetContent(minSize.View(m.w, m.h))
		return v
	}
	m.layout()

	c := lipgloss.NewCanvas(m.w, m.h)
	layers := []*lipgloss.Layer{lipgloss.NewLayer(m.topBar())}
	for p := range paneCount {
		r := m.rects[p]
		if r.Empty() {
			continue
		}
		layers = append(layers, lipgloss.NewLayer(m.paneView(p, r)).X(r.X).Y(r.Y))
	}
	if theme.HelpBar() {
		layers = append(layers, lipgloss.NewLayer(m.helpBar()).Y(m.h-1))
	}
	c.Compose(lipgloss.NewCompositor(layers...))
	draw.FillRow(c, 0, theme.Surface())
	if theme.HelpBar() {
		draw.FillRow(c, m.h-1, theme.Surface())
	}
	m.overlays.Draw(c)
	if t := m.toasts(); len(t) > 0 {
		c.Compose(lipgloss.NewCompositor(t...))
	}
	v.SetContent(c.Render())
	return v
}

func (m *Model) paneView(p pane, r geom.Rect) string {
	w := max(r.W-2, 0)
	focused := p == m.focus && m.overlays.Top() == nil
	o := draw.FrameOpts{Title: paneIcons[p]() + paneNames[p], Active: focused}
	var content string
	switch p {
	case paneList:
		content = m.listView(w)
		o.Title = fmt.Sprintf("%s (%d/%d)", paneNames[p], len(m.shown), len(m.all))
		o.Scroll = m.list.Scroll()
	case panePreview:
		content = m.preview.View()
		if e, ok := m.selected(); ok {
			o.Title += " · " + e.Name
		}
		o.Scroll = m.preview.Scroll()
	}
	return draw.Frame(content, r.W, r.H, o)
}

// paneIcons put a Nerd Font glyph before each pane's title, nothing without.
var paneIcons = [paneCount]func() string{
	paneList:    func() string { return theme.Icon("\ueb01 ", "") },
	panePreview: func() string { return theme.Icon("\uf02d ", "") },
}

func (m *Model) listView(w int) string {
	var lines []string
	if m.searchShown() {
		lines = append(lines, m.search.View(w, m.searching), draw.Rule(w))
	}
	if len(m.shown) == 0 {
		lines = append(lines, theme.Subtle().Render(" no match"))
	}
	start, end := m.list.Visible()
	for i := start; i < end; i++ {
		e := m.shown[i]
		cat := ansi.Truncate(categoryLabel(e.Category), categoryWidth, "")
		cat += strings.Repeat(" ", max(categoryWidth-ansi.StringWidth(cat), 0))
		name := ansi.Truncate(e.Name, max(w-categoryWidth-3, 0), "…")
		row := " " + theme.Subtle().Render(cat) + " " + theme.Fg().Render(name)
		if i == m.list.Selected() {
			row = draw.Selected(row, w, m.focus == paneList && m.overlays.Top() == nil)
		}
		lines = append(lines, row)
	}
	return strings.Join(lines, "\n")
}

// syncPreview redraws the preview when the selection, the variables or the
// encoding change.
func (m *Model) syncPreview() {
	e, ok := m.selected()
	var text string
	if ok {
		text = catalog.Render(e, m.renderVars(), m.encoding) + "\n\n" + m.listenerHint()
	}
	if text == m.shownPreview {
		return
	}
	m.shownPreview = text
	if !ok {
		m.preview.SetContent(nil, nil)
		return
	}
	plain := strings.Split(text, "\n")
	styled := make([]string, len(plain))
	for i, l := range plain {
		// styled lines are rendered as-is: sanitize them like the viewer
		// does the plain ones, so no control char reaches the canvas
		l = viewer.Sanitize(l)
		plain[i] = l
		if i < len(plain)-1 {
			styled[i] = theme.Primary().Render(l)
		} else {
			styled[i] = theme.Subtle().Render(l)
		}
	}
	m.preview.SetContent(plain, styled)
}

func (m *Model) listenerHint() string {
	for _, l := range catalog.Listeners() {
		if l.Name == "nc" {
			r := strings.NewReplacer("{port}", m.vars.Port, "{ip}", m.vars.IP)
			return "listener: " + r.Replace(l.Command)
		}
	}
	return ""
}

func (m *Model) topBar() string {
	left := theme.Accent().Style().Bold(true).Render(" " + theme.Icon("\uea85 ", "") + "revshell ")
	right := theme.Subtle().Render(fmt.Sprintf("category: %s   os: %s   encoding: %s ",
		categoryLabel(categoryFilters[m.category]), osLabel(osFilters[m.os]), encodingLabel(m.encoding)))
	return m.bar(left, right)
}

func (m *Model) bar(left, right string) string {
	pad := max(m.w-ansi.StringWidth(left)-ansi.StringWidth(right), 0)
	return ansi.Truncate(left+strings.Repeat(" ", pad)+right, m.w, "")
}

func (m *Model) helpBar() string {
	ks, ls := theme.Primary().Style(), theme.Subtle().Style()
	var parts []string
	for _, h := range m.reg.Hints(m.path()) {
		first, _, _ := strings.Cut(h.Keys, " / ") // the others are in the help
		parts = append(parts, ks.Render(first)+" "+ls.Render(h.Label))
	}
	return draw.Pad(ansi.Truncate(" "+strings.Join(parts, ls.Render("  ·  ")), m.w, "…"), m.w)
}

// toasts draws the live notifications at the top right, newest first.
func (m *Model) toasts() []*lipgloss.Layer {
	var out []*lipgloss.Layer
	w := min(40, max(m.w/3, 20), m.w)
	y := 0
	m.toastRects = m.toastRects[:0]
	live := m.notes.Live()
	for i := len(live) - 1; i >= 0 && i >= len(live)-5; i-- {
		s := live[i].Toast(w)
		r := geom.Rect{X: max(m.w-w, 0), Y: y, W: w, H: lipgloss.Height(s)}
		if len(out) > 0 && y+r.H > m.h {
			break // the older ones would be cut
		}
		m.toastRects = append(m.toastRects, toastRect{r, live[i].ID})
		out = append(out, lipgloss.NewLayer(s).X(r.X).Y(r.Y).Z(20))
		y += r.H
	}
	return out
}
