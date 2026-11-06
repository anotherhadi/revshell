// Package catalog is the reverse/bind shell, MSFVenom, HoaxShell and
// shellcode one-liners from the 0dayCTF reverse-shell-generator
// (https://github.com/0dayCTF/reverse-shell-generator, MIT, Ryan Montgomery),
// vendored as a git submodule and regenerated with `go generate` (see
// generate.go) so the catalog tracks upstream instead of being hand-copied.
package catalog

import (
	_ "embed"
	"encoding/json"
	"slices"
	"strings"
)

type Category string

const (
	ReverseShell Category = "ReverseShell"
	BindShell    Category = "BindShell"
	MSFVenom     Category = "MSFVenom"
	HoaxShell    Category = "HoaxShell"
	Shellcode    Category = "Assembled"
)

var Categories = []Category{ReverseShell, BindShell, MSFVenom, HoaxShell, Shellcode}

// Entry is a single one-liner template, {ip}/{port}/{shell} placeholders
// unresolved. Tags holds everything from upstream's meta except the category
// itself: OS/arch (linux, mac, windows, android, apple_ios) and flags like
// bind, staged, meterpreter, bufferoverflow.
type Entry struct {
	Name     string
	Category Category
	Command  string
	Tags     []string
}

// Listener is a local listener one-liner (nc, socat, pwncat...), {port}/{ip}
// placeholders unresolved.
type Listener struct {
	Name    string
	Command string
}

//go:embed data.json
var dataJSON []byte

type rawEntry struct {
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Meta    []string `json:"meta"`
}

type rawData struct {
	ReverseShellCommands []rawEntry  `json:"reverseShellCommands"`
	ListenerCommands     [][2]string `json:"listenerCommands"`
	Shells               []string    `json:"shells"`
}

var (
	entries   []Entry
	listeners []Listener
	shells    []string
)

func init() {
	var raw rawData
	if err := json.Unmarshal(dataJSON, &raw); err != nil {
		panic("catalog: " + err.Error())
	}

	entries = make([]Entry, 0, len(raw.ReverseShellCommands))
	for _, re := range raw.ReverseShellCommands {
		e := Entry{Name: re.Name, Command: re.Command}
		for _, t := range re.Meta {
			if slices.Contains(Categories, Category(t)) {
				e.Category = Category(t)
				continue
			}
			e.Tags = append(e.Tags, t)
		}
		entries = append(entries, e)
	}

	listeners = make([]Listener, 0, len(raw.ListenerCommands))
	for _, l := range raw.ListenerCommands {
		listeners = append(listeners, Listener{Name: l[0], Command: l[1]})
	}

	shells = raw.Shells
}

func All() []Entry          { return slices.Clone(entries) }
func Listeners() []Listener { return slices.Clone(listeners) }
func Shells() []string      { return slices.Clone(shells) }

// SearchText is what a fuzzy query matches an Entry against: its name, then
// its category and tags.
func SearchText(e Entry) string {
	return e.Name + " " + string(e.Category) + " " + strings.Join(e.Tags, " ")
}
