# Build plan — kohrdp (Go TUI)

A standalone Go TUI for launching RDP sessions. It keeps the
same JSON session data and FreeRDP launch behavior, while using `go-keyring` for
password storage.

The binary also supports a direct compatibility CLI mode for launching one
connection without opening the TUI.

## Stack

- **github.com/charmbracelet/bubbletea** — TUI runtime (Elm-style model/update/view)
- **github.com/charmbracelet/bubbles** — prebuilt `table`, `textinput`, `key`, `help`, `spinner`
- **github.com/charmbracelet/lipgloss** — styling/layout
- Keyring: `go-keyring`
- Go 1.22+, single static binary

## Repo layout

```
kohrdp/
├── go.mod
├── main.go              # entry: load config, start bubbletea
├── internal/
│   ├── config/
│   │   └── config.go    # load/save sessions.json, Session struct
│   ├── keyring/
│   │   └── keyring.go   # go-keyring wrappers
│   ├── rdp/
│   │   └── launch.go    # build xfreerdp3 args, pipe password, spawn
│   └── ui/
│       ├── model.go     # bubbletea model + Update/View
│       ├── table.go     # session list view
│       ├── form.go      # new/edit form view
│       └── keys.go      # keybinding definitions + help
└── README.md
```

## Data model

`~/.config/kohrdp/sessions.json` → `map[string]Session`

On first load after the rename, an existing
`~/.config/rdpkoh/sessions.json` is copied to the new location. Credentials
found under the former `rdpkoh` keyring service are stored under `kohrdp` when
they are next used.

```go
type Session struct {
    User       string `json:"user"`
    Host       string `json:"host"`
    LastUsed   int64  `json:"lastUsed,omitempty"`
    IgnoreCert *bool  `json:"ignore_cert,omitempty"` // nil = "never asked"
}
```

- Map keyed by session name.
- Table sorted by `LastUsed` desc.
- `IgnoreCert` as `*bool` preserves the three-state logic (unset / true / false)
  the bash script uses.

## Critical compatibility details

Existing saved session definitions MUST keep working.

1. **Username munge** — replicate `if user has no '\\', prefix '.\\'` before any
   keyring lookup or `/u:`.
2. **Keyring identity** — passwords are stored under the app's own keyring
   service name using the munged user and host as the entry identity.
3. **xfreerdp3 invocation** — port verbatim:
   ```
   /u:<user> /from-stdin /v:<host> [/wm-class:kohrdp-<class>] [/cert:ignore]
   +auto-reconnect +clipboard +fonts /sound /dynamic-resolution
   /gfx /bpp:32 /drive:Downloads,$HOME/Downloads
   ```
   Password piped to the child's **stdin** (`/from-stdin`) — never `/p:`
   (that leaks into `ps`). Set `cmd.Stdin` to a pipe, write `password + "\n"`, close.

## TUI behavior

**Main view** — table: `name · user@host · status · cert · last used` (relative time),
with a per-session FreeRDP log pane on the right.

Per-session logs are retained in temp files during the app lifetime so the UI can
scroll them without keeping the whole log history in memory.

| key | action |
|-----|--------|
| `enter` | connect to selected |
| `n` | new session (form) |
| `e` | edit selected (form) |
| `d` | delete (with confirm) |
| `c` | toggle `ignore_cert` |
| `p` | clear keyring password (force reprompt next connect) |
| `m` | connect with multimon |
| `[` / `PgUp` | older logs for selected session |
| `]` / `PgDn` | newer logs for selected session |
| `?` | toggle help |
| `q`/`esc` | quit |

**Form view** (new/edit) — `textinput` fields: name, user, host, + cert toggle.
Enter saves to JSON, esc cancels.

**Connect flow**:
1. Resolve munged user → OS keyring lookup.
2. If no password: prompt with an in-TUI password `textinput` state, offer to
   store in keyring.
3. Resolve cert: flag/stored pref → `/cert:ignore`; if `IgnoreCert == nil`, show a
   confirm and persist the choice.
4. Bump `LastUsed`, save JSON, spawn xfreerdp3 **detached** (like bash `&`), return
   to the table.
5. Keep per-session runtime state in the table (`idle`, `starting`, `active`, `exited`,
   `failed`) and show recent FreeRDP logs in the side pane.

## Tricky bit to plan for

bubbletea owns the terminal. Spawning xfreerdp (GUI, detached) is fine — just
`cmd.Start()` and don't wait. But the **interactive password prompt** on a keyring
miss should be its own bubbletea state (a password `textinput`), not a raw `read`,
to avoid fighting the renderer. If you must shell out interactively, use
`tea.ExecProcess`.

## Build / install

- `make build`
- `make install`
- `make install-tarball TARBALL=...`
- `make check`
- `make release-tarball` for a generic versioned binary tarball
- `make dist-arch` then `cd dist/arch && makepkg -si` for Arch Linux packaging
- Or manually: `go build -o kohrdp .`
- Install to `~/.local/bin` with the command name `kohrdp`.
- `kohrdp --export FILE` / `--export -` and `kohrdp --import FILE`
  / `--import -` move session definitions between machines without exporting
  passwords.
- `--import-mode replace|merge` controls whether import replaces the current
  config or overlays it.
- `kohrdp USER HOST [--ignore-cert] [--name NAME] [--multimon]` launches a
  direct connection in compatibility mode.

## Migration / parity checklist

- [ ] Reads existing `sessions.json` unchanged
- [ ] Recency ordering matches (`lastUsed` desc)
- [ ] Cert prompt persists `ignore_cert` like the bash version
- [ ] Password never appears in `ps`
- [ ] xfreerdp flags identical to bash launcher
