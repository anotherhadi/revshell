// Package settings feeds the settings popup.
package settings

import (
	"fmt"
	"strconv"

	"github.com/sectui/foam/form"
)

// Vars are the values behind the catalog's {ip}/{port}/{shell} placeholders.
type Vars struct {
	IP    string
	Port  string
	Shell string
}

// Default returns the factory values.
func Default() Vars { return Vars{IP: "10.10.10.10", Port: "9001", Shell: "/bin/sh"} }

var meta = []form.Field{
	{ID: "ip", Label: "IP", Help: "Address the shell connects back to (tun0, LAN IP)."},
	{ID: "port", Label: "Port", Help: "Listener port, 1-65535."},
	{ID: "shell", Label: "Shell", Help: "Shell to spawn: /bin/sh, /bin/bash, cmd.exe…"},
}

func (v Vars) value(id string) string {
	switch id {
	case "ip":
		return v.IP
	case "port":
		return v.Port
	default:
		return v.Shell
	}
}

func (v *Vars) set(id, value string) {
	switch id {
	case "ip":
		v.IP = value
	case "port":
		v.Port = value
	default:
		v.Shell = value
	}
}

func (v *Vars) Fields() []form.Field {
	def := Default()
	out := make([]form.Field, 0, len(meta))
	for _, f := range meta {
		f.Value = v.value(f.ID)
		if f.Value == def.value(f.ID) {
			f.Origin = "default"
		}
		out = append(out, f)
	}
	return out
}

func (v *Vars) Set(id, value string) error {
	if id == "port" {
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > 0xFFFF {
			return fmt.Errorf("port must be 1-65535")
		}
	}
	v.set(id, value)
	return nil
}

func (v *Vars) Reset(id string) error {
	v.set(id, Default().value(id))
	return nil
}
