// Command gen evaluates the vendored reverse-shell-generator data.js in a
// real JS engine and dumps rsgData as JSON, so the catalog never has to be
// hand-transcribed again: refresh the submodule, rerun this.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dop251/goja"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: gen <data.js> <out.json>")
		os.Exit(2)
	}
	src, err := os.ReadFile(os.Args[1])
	if err != nil {
		fatal(err)
	}

	vm := goja.New()
	if _, err := vm.RunScript(filepath.Base(os.Args[1]), string(src)); err != nil {
		fatal(err)
	}

	v, err := vm.RunString("JSON.stringify(rsgData)")
	if err != nil {
		fatal(err)
	}

	var data any
	if err := json.Unmarshal([]byte(v.String()), &data); err != nil {
		fatal(err)
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(os.Args[2], out, 0o644); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "gen:", err)
	os.Exit(1)
}
