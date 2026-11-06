```
█▀▀ █▀▀ █▀▀ ▀█▀ █ █ █   ▄▀ █▀█ █▀▀ █ █ █▀▀ █ █ █▀▀ █   █  
▄▄█ ██▄ █▄▄  █  █▄█ █ ▄▀   █▀▄ ██▄ ▀▄▀ ▄▄█ █▀█ ██▄ █▄▄ █▄▄
```

# Revshell: reverse/bind shell one-liners, fast

Fill `{ip}`/`{port}`/`{shell}` into known reverse shell, bind shell, MSFVenom,
HoaxShell and shellcode one-liners from a searchable TUI or a scriptable CLI.
Part of the **sectui** tools.

The catalog comes from
[0dayCTF/reverse-shell-generator](https://github.com/0dayCTF/reverse-shell-generator)
(MIT, Ryan Montgomery), vendored as a git submodule and regenerated with
`go generate`

![demo](.github/assets/demo.gif)

## Install

With go:

```sh
go install github.com/sectui/revshell/cmd/revshell@latest
```

With Nix, from the [sectui NUR](https://github.com/sectui/nur):

```sh
nix run github:sectui/nur#revshell
```

```nix
# flake: inputs.nur-sectui.url = "github:sectui/nur";
environment.systemPackages = [ nur-sectui.packages.${system}.revshell ];

# or through NUR
environment.systemPackages = [ pkgs.nur.repos.sectui.revshell ];
```

Each [GitHub release](https://github.com/sectui/revshell/releases) has an
archive per platform (Linux, macOS, Windows and FreeBSD, amd64 and arm64) with
the `revshell` binary, and a `SHA256SUMS` file.

## TUI

```sh
revshell
```

- `ctrl+h`/`ctrl+l`: move between the panes; `alt+1`/`alt+2`: jump to the list
  or the preview; `alt+z`: zoom the pane
- `/`: search the catalog (type to filter, `esc` to stop)
- `ctrl+o`: the ip, port and shell the commands are rendered with
- `↑`/`↓`, `j`/`k`, `pgup`/`pgdown`, `home`/`end`: move the list, scroll the
  preview
- `c`: cycle category (reverse, bind, msfvenom, hoaxshell, shellcode)
- `o`: cycle the OS filter
- `e`: cycle encoding (none, url, url2, base64)
- `y` / `enter`: copy the rendered command

## CLI

```sh
revshell list [--category ReverseShell] [--os linux]
revshell search bash
revshell gen "Bash -i" --ip 10.0.0.1 --port 4444 --shell /bin/bash
revshell gen "Windows - Reverse TCP (x86)" --ip 10.0.0.1 --port 4444   # shellcode: ip/port hex-escaped
revshell listeners --port 4444
```

`gen`/`list`/`search`/`listeners` print to stdout for piping, e.g.
`revshell gen ... | wl-copy` or straight into an `nc`/`ssh` command.
