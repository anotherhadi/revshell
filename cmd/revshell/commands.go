package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/sectui/foam/fuzzy"
	"github.com/sectui/revshell/internal/catalog"
)

func runList(argv []string, out io.Writer) error {
	fs := pflag.NewFlagSet("list", pflag.ContinueOnError)
	category := fs.String("category", "", "filter by category")
	os := fs.String("os", "", "filter by OS tag")
	if err := fs.Parse(argv); err != nil {
		return usagef("%s", err)
	}
	for _, e := range catalog.All() {
		if *category != "" && !strings.EqualFold(string(e.Category), *category) {
			continue
		}
		if *os != "" && !hasTag(e.Tags, *os) {
			continue
		}
		fmt.Fprintf(out, "%s\t%s\t%s\n", e.Category, e.Name, strings.Join(e.Tags, ","))
	}
	return nil
}

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

func runSearch(argv []string, out io.Writer) error {
	if len(argv) == 0 {
		return usagef("expected a search term, see `revshell --help`")
	}
	term := strings.Join(argv, " ")
	all := catalog.All()
	texts := make([]string, len(all))
	for i, e := range all {
		texts[i] = catalog.SearchText(e)
	}
	for _, i := range fuzzy.Filter(term, texts) {
		fmt.Fprintf(out, "%s\t%s\n", all[i].Category, all[i].Name)
	}
	return nil
}

func parseEncoding(s string) (catalog.Encoding, error) {
	switch strings.ToLower(s) {
	case "", "none":
		return catalog.NoEncoding, nil
	case "url":
		return catalog.URLEncoding, nil
	case "url2":
		return catalog.URLEncodingDouble, nil
	case "base64":
		return catalog.Base64Encoding, nil
	default:
		return 0, usagef("unknown encoding %q, expected none, url, url2 or base64", s)
	}
}

func findByName(name string) (catalog.Entry, bool) {
	for _, e := range catalog.All() {
		if strings.EqualFold(e.Name, name) {
			return e, true
		}
	}
	return catalog.Entry{}, false
}

func runGen(argv []string, out io.Writer) error {
	fs := pflag.NewFlagSet("gen", pflag.ContinueOnError)
	ip := fs.String("ip", "", "target/listener IP")
	port := fs.Int("port", 9001, "port")
	shell := fs.String("shell", "/bin/sh", "shell to invoke")
	encoding := fs.String("encoding", "none", "none, url, url2 or base64")
	if err := fs.Parse(argv); err != nil {
		return usagef("%s", err)
	}
	if fs.NArg() != 1 {
		return usagef("expected exactly one entry name, see `revshell --help`")
	}
	name := fs.Arg(0)

	e, ok := findByName(name)
	if !ok {
		return fmt.Errorf("no entry named %q, see `revshell list`", name)
	}
	enc, err := parseEncoding(*encoding)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, catalog.Render(e, catalog.Vars{IP: *ip, Port: *port, Shell: *shell}, enc))
	return nil
}

func runListeners(argv []string, out io.Writer) error {
	fs := pflag.NewFlagSet("listeners", pflag.ContinueOnError)
	ip := fs.String("ip", "", "IP (only used by the msfconsole listener)")
	port := fs.Int("port", 9001, "port")
	if err := fs.Parse(argv); err != nil {
		return usagef("%s", err)
	}
	r := strings.NewReplacer("{port}", strconv.Itoa(*port), "{ip}", *ip)
	for _, l := range catalog.Listeners() {
		fmt.Fprintf(out, "%s\t%s\n", l.Name, r.Replace(l.Command))
	}
	return nil
}
