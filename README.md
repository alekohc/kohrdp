# kokoarch-rdp

Terminal UI for picking and launching RDP sessions with `xfreerdp3`.

It uses the same session file as the old bash launcher, while storing passwords
through the OS keyring with `go-keyring`.

## Requirements

- Go 1.22+
- `xfreerdp3`
- Linux or macOS desktop with a working keyring

## Install

Build once:

```sh
go build -o kokoarch-rdp .
install -Dm755 kokoarch-rdp "$HOME/.local/bin/kokoarch-rdp"
```

Or use the Makefile:

```sh
make install
```

By default that installs to `~/.local/bin/kokoarch-rdp`.

## Development

Useful targets:

```sh
make build
make install
make test
make vet
make fmt
make check
make dist-arch
make release-tarball
```

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

## Usage

Run:

```sh
kokoarch-rdp
```

Show the installed version:

```sh
kokoarch-rdp --version
```

Export sessions to move them to another machine:

```sh
kokoarch-rdp --export sessions.json
```

Import sessions from another machine:

```sh
kokoarch-rdp --import sessions.json
```

Sessions are loaded from:

```text
~/.config/kokoarch/rdp-sessions.json
```

Passwords are never stored in that JSON file. Import/export only moves session
definitions. Passwords stay in the system keyring.

The app uses the OS keyring via `go-keyring`.

## Main View

The table shows:

- session name
- `user@host`
- current runtime status
- certificate mode
- last used time

The right-hand pane shows recent FreeRDP logs for the selected session.
Those logs are stored in temp files while the app is running, so long sessions do
not keep growing the in-memory UI state.

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
