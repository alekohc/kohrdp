# rdpkoh

Terminal UI for picking and launching RDP sessions with `xfreerdp3`.

It uses the same session file as the old bash launcher, while storing passwords
through the OS keyring with `go-keyring`.

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
go build -o rdpkoh .
install -Dm755 rdpkoh "$HOME/.local/bin/rdpkoh"
```

Or use the Makefile:

```sh
make install
```

By default that installs to `~/.local/bin/rdpkoh`.

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
make install-tarball TARBALL=dist/release/rdpkoh-0.1.0-linux-amd64.tar.gz
```

## Usage

Run:

```sh
rdpkoh
```

Old direct-connect style is also supported:

```sh
rdpkoh administrator 192.168.11.10 --ignore-cert --name tp-test
```

That launches a connection directly without opening the TUI. If the password is
not already in the keyring, the app prompts in the terminal and can save it.

Show the installed version:

```sh
rdpkoh --version
```

Export sessions to move them to another machine:

```sh
rdpkoh --export sessions.json
```

Export to stdout:

```sh
rdpkoh --export -
```

Import sessions from another machine:

```sh
rdpkoh --import sessions.json
```

Import from stdin:

```sh
rdpkoh --import -
```

Import modes:

- `--import-mode replace`: replace the current session config
- `--import-mode merge`: merge imported sessions into the current config

Sessions are loaded from:

```text
~/.config/kokoarch/rdp-sessions.json
```

Passwords are never stored in that JSON file. Import/export only moves session
definitions. Passwords stay in the system keyring.

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
Those logs are stored in temp files while the app is running, so long sessions do
not keep growing the in-memory UI state. Temp log files are cleaned up when the
app exits.

Session status values:

- `idle`: not running
- `starting`: process started and is still coming up
- `active`: running
- `exited`: process ended cleanly
- `failed`: launch failed or the process exited with an error

## Keys

- `enter`: connect
- `m`: connect with multimon
- `n`: new session
- `e`: edit session
- `d`: delete session
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

- FreeRDP logs are captured and shown in the TUI.
- The TUI follows the terminal's ANSI color palette.
- FreeRDP itself is still launched with pipes, not a PTY, so its own native
  tty-detection behavior is not enabled.
