# kohrdp

Terminal UI for picking and launching RDP sessions with `xfreerdp3`.

It stores sessions in its own config file and keeps passwords in the OS keyring
with `go-keyring`.

## Requirements

- Go 1.24+
- `make`
- `tar`
- `install`
- `xfreerdp3`
- Linux or macOS desktop with a working OS keyring

Linux prerequisites typically mean:

- FreeRDP with the `xfreerdp3` binary available in `PATH`
- a Secret Service-compatible keyring session available to your desktop login

macOS prerequisites typically mean:

- FreeRDP installed with `xfreerdp3` available in `PATH`
- access to the macOS Keychain

Build and packaging prerequisites:

- `make install` / `make build`: Go toolchain only
- `make release-tarball`: Go toolchain plus `tar`
- `make dist-arch`: Arch Linux tooling if you intend to use `makepkg`

## Install

Build once:

```sh
go build -o kohrdp .
install -Dm755 kohrdp "$HOME/.local/bin/kohrdp"
```

Or use the Makefile:

```sh
make install
```

By default that installs to `~/.local/bin/kohrdp`.

## Development

Useful targets:

```sh
make build
make install
make install-tarball TARBALL=...
make test
make vet
make fmt
make check
make dist-arch
make release-tarball
```

The base app version comes from the `VERSION` file.

Release/version workflow:

```sh
make check
make release-tarball
```

If you are preparing a real release, bump `VERSION` first. Tagged GitHub
releases should use `vX.Y.Z` tags matching the `VERSION` file.

`make dist-arch` prepares an Arch `makepkg` directory under `dist/arch/`.
From there:

```sh
cd dist/arch
makepkg -si
```

That packaging flow is for Arch Linux only. It is not usable on macOS.

`make release-tarball` creates a versioned binary tarball under `dist/release/`.
That is the generic packaging path and is usable on both Linux and macOS as a
build artifact format.

Install from a release tarball:

```sh
make install-tarball TARBALL=dist/release/kohrdp-<version>-linux-amd64.tar.gz
```

## Usage

Run:

```sh
kohrdp
```

Launch a saved session by name, without opening the TUI:

```sh
kohrdp win11
kohrdp win11 --multimon
```

This uses all of the session's stored properties (drives, size, redirection,
etc.). An ad-hoc user/host connection is also supported:

```sh
kohrdp administrator 192.168.11.10 --ignore-cert --name tp-test
```

Both launch a connection directly without opening the TUI. If the password is
not already in the keyring, the app prompts in the terminal and can save it.

Show the installed version:

```sh
kohrdp --version
```

Export sessions to move them to another machine:

```sh
kohrdp --export sessions.json
```

Export to stdout:

```sh
kohrdp --export -
```

Import sessions from another machine:

```sh
kohrdp --import sessions.json
```

Import from stdin:

```sh
kohrdp --import -
```

Import modes:

- `--import-mode replace`: replace the current session config
- `--import-mode merge`: merge imported sessions into the current config

Sessions are loaded from:

```text
~/.config/kohrdp/sessions.json
```

Passwords are never stored in that JSON file. Import/export only moves session
definitions. Passwords stay in the system keyring.

On first run after upgrading from `rdpkoh`, the existing session file is copied
to the new config location. Saved credentials are migrated within the OS
keyring as they are used. The installed command is now `kohrdp`; an existing
`rdpkoh` binary can be removed after installing the renamed application.

The app uses the OS keyring via `go-keyring`.

License: MIT. See [`LICENSE`](LICENSE).

## Main View

The table shows:

- session name
- `user@host`
- current runtime status
- certificate mode
- last used time

The right-hand pane shows recent FreeRDP logs for the selected session.
`xfreerdp3` writes its output straight to a per-session log file under
`/tmp/kohrdp/`. On quit, log files are removed for sessions that have ended but
kept for sessions still running, so a later launch can pick their logs back up.

Status and logs survive restarting the app: on startup kohrdp scans for running
`xfreerdp3` processes and re-attaches to them, marking the matching sessions
`active` and tailing their logs again. (Sessions launched by an older build that
did not write to `/tmp/kohrdp/` show as `active` but without historical logs.)

Session status values:

- `idle`: not running
- `starting`: process started and is still coming up
- `active`: running
- `exited`: process ended cleanly
- `failed`: launch failed or the process exited with an error

## Session fields

The new/edit form (`n` / `e`) has required and optional fields:

- `name`, `user`, `host` — required.
- `domain` — Windows domain (`/d:`).
- `gateway` — RD Gateway host, optionally `host:port` (`/gateway:g:`).
- `size` — resolution as `WxH` (e.g. `1920x1080`) or `percent%` (`/size:`).
- `drives` — space-separated `name,path` mounts (`/drive:`), e.g.
  `work,/home/you/work media,/mnt/media`. When empty, `~/Downloads` is mounted
  as before.

Optional text fields left blank are simply omitted from the `xfreerdp3` command.

The form also has toggles (navigate with `tab`/`↑↓`, flip with `space`):

- `cert` — certificate mode (unset / ignore / enforce).
- `clipboard` — on by default (`+clipboard` / `-clipboard`).
- `sound` — on by default (`/sound`).
- `microphone` — off by default (`/microphone`).
- `printer` — off by default (`/printer`).
- `smartcard` — off by default (`/smartcard`).

## Keys

- `enter`: connect
- `m`: connect with multimon
- `n`: new session
- `e`: edit session
- `d`: delete session
- `x`: disconnect the selected running session (asks to confirm)
- `c`: toggle certificate mode
- `p`: clear saved password from keyring
- `[` or `PgUp`: older logs for the selected session
- `]` or `PgDn`: newer logs for the selected session
- `?`: toggle help
- `q` or `esc`: quit

## Connect Flow

1. Look up the password in the keyring.
2. If missing, prompt in the TUI and optionally save it.
3. Resolve certificate behavior.
4. Start `xfreerdp3` with the password over stdin.
5. Return to the session list while logs and session state update in the UI.

## Notes

- FreeRDP logs are captured to `/tmp/kohrdp/` and shown in the TUI.
- The TUI follows the terminal's ANSI color palette.
- FreeRDP writes to a log file, not a PTY, so its own native tty-detection
  behavior is not enabled.
