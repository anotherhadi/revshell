// Command revshell generates reverse/bind shell, MSFVenom, HoaxShell and
// shellcode one-liners from the anotherhadi/revshell catalog. With no
// arguments it opens the TUI; subcommands are for scripting.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/anotherhadi/revshell/internal/ui"
)

// Set with -ldflags "-X main.version=1.2.3".
var version string

func getVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return "dev"
}

type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

func usagef(format string, a ...any) error { return &usageError{fmt.Sprintf(format, a...)} }

func main() {
	// Get EPIPE when the reader goes away (`revshell list | head`), and exit quietly.
	signal.Ignore(syscall.SIGPIPE)
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(argv []string, stdout, stderr *os.File) int {
	err := dispatch(argv, stdout)
	var usage *usageError
	switch {
	case err == nil, errors.Is(err, syscall.EPIPE):
		return 0
	case errors.As(err, &usage):
		fmt.Fprintln(stderr, "revshell:", err)
		return 2
	default:
		fmt.Fprintln(stderr, "revshell:", err)
		return 1
	}
}

func dispatch(argv []string, stdout *os.File) error {
	if len(argv) == 0 {
		return ui.Run()
	}
	command, rest := argv[0], argv[1:]
	switch command {
	case "-h", "--help":
		fmt.Fprint(stdout, usage)
		return nil
	case "-v", "--version":
		fmt.Fprintln(stdout, getVersion())
		return nil
	case "list":
		return runList(rest, stdout)
	case "search":
		return runSearch(rest, stdout)
	case "gen":
		return runGen(rest, stdout)
	case "listeners":
		return runListeners(rest, stdout)
	default:
		return usagef("unknown command %q, see `revshell --help`", command)
	}
}

const usage = `revshell generates reverse/bind shell, MSFVenom, HoaxShell and shellcode one-liners.

Usage:
  revshell                                  launch the TUI
  revshell list [--category C] [--os OS]    list catalog entries
  revshell search <term>                    fuzzy search entry names
  revshell gen <name> --ip IP --port PORT [--shell SHELL] [--encoding ENC]
  revshell listeners [--port PORT] [--ip IP]

Categories: ReverseShell, BindShell, MSFVenom, HoaxShell, Assembled.
Encodings: none, url, url2, base64.
`
