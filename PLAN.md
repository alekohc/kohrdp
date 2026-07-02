# Build plan — kokoarch-rdp (Go TUI)

A standalone Go TUI to replace the `kokoarch-rdp` bash launcher. Designed to be
**drop-in compatible** with the bash script's data (same JSON, same keyring
entries) so both can run during the transition.

## Stack

- **github.com/charmbracelet/bubbletea** — TUI runtime (Elm-style model/update/view)
- **github.com/charmbracelet/bubbles** — prebuilt `table`, `textinput`, `key`, `help`, `spinner`
- **github.com/charmbracelet/lipgloss** — styling/layout
- Keyring: shell out to `secret-tool` (matches existing entries exactly) — see compat note
- Go 1.22+, single static binary

## Repo layout

```
kokoarch-rdp/
├── go.mod
├── main.go              # entry: load config, start bubbletea
├── internal/
│   ├── config/
│   │   └── config.go    # load/save rdp-sessions.json, Session struct
│   ├── keyring/
│   │   └── keyring.go   # secret-tool lookup/store/clear wrappers
│   ├── rdp/
│   │   └── launch.go    # build xfreerdp3 args, pipe password, spawn
│   └── ui/
│       ├── model.go     # bubbletea model + Update/View
│       ├── table.go     # session list view
│       ├── form.go      # new/edit form view
│       └── keys.go      # keybinding definitions + help
└── README.md
```

## Data model (must match existing JSON)

`~/.config/kokoarch/rdp-sessions.json` → `map[string]Session`

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

Existing saved sessions and keyring passwords MUST keep working.

1. **Username munge** — replicate `if user has no '\', prefix '.\'` before any
   keyring lookup or `/u:`. Keyring entries were stored against the munged name
   (`.\admin`); this must match byte-for-byte.
2. **Keyring attributes** — `secret-tool lookup rdp host <HOST> rdp user <MUNGED_USER>`
   with the same attribute pairs and order. Same for `store --label="RDP <HOST>"`
   and `clear`.
3. **xfreerdp3 invocation** — port verbatim:
   ```
   /u:<user> /from-stdin /v:<host> [/wm-class:kokoarch-rdp-<class>] [/cert:ignore]
   +auto-reconnect +clipboard +fonts /sound /dynamic-resolution
   /gfx /bpp:32 /drive:Downloads,$HOME/Downloads
   ```
   Password piped to the child's **stdin** (`/from-stdin`) — never `/p:`
   (that leaks into `ps`). Set `cmd.Stdin` to a pipe, write `password + "\n"`, close.

## TUI behavior

**Main view** — table: `name · user@host · cert · last used` (relative time).

| key | action |
|-----|--------|
| `enter` | connect to selected |
| `n` | new session (form) |
| `e` | edit selected (form) |
| `d` | delete (with confirm) |
| `c` | toggle `ignore_cert` |
| `p` | clear keyring password (force reprompt next connect) |
| `m` | connect with multimon |
| `/` | filter (bubbles table built-in) |
| `?` | toggle help |
| `q`/`esc` | quit |

**Form view** (new/edit) — `textinput` fields: name, user, host, + cert toggle.
Enter saves to JSON, esc cancels.

**Connect flow**:
1. Resolve munged user → `secret-tool lookup`.
2. If no password: drop out of alt-screen, prompt (bubbletea password `textinput`
   state), offer to store in keyring.
3. Resolve cert: flag/stored pref → `/cert:ignore`; if `IgnoreCert == nil`, show a
   confirm and persist the choice.
4. Bump `LastUsed`, save JSON, spawn xfreerdp3 **detached** (like bash `&`), return
   to the table.

## Tricky bit to plan for

bubbletea owns the terminal. Spawning xfreerdp (GUI, detached) is fine — just
`cmd.Start()` and don't wait. But the **interactive password prompt** on a keyring
miss should be its own bubbletea state (a password `textinput`), not a raw `read`,
to avoid fighting the renderer. If you must shell out interactively, use
`tea.ExecProcess`.

## Build / install

- `go build -o kokoarch-rdp .`
- Copy/symlink into `~/.local/bin` (keep the command name so fish aliases / the
  launcher keep working).

## Migration / parity checklist

- [ ] Reads existing `rdp-sessions.json` unchanged
- [ ] Finds passwords stored by the bash script (munge + attributes match)
- [ ] Recency ordering matches (`lastUsed` desc)
- [ ] Cert prompt persists `ignore_cert` like the bash version
- [ ] Password never appears in `ps`
- [ ] xfreerdp flags identical to bash launcher
