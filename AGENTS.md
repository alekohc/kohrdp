# kokoarch-rdp — Constitution & Rules

Single source of truth for this repo, for humans and agents alike. Any other
instruction file (`CLAUDE.md`, `.cursorrules`, `.github/copilot-instructions.md`)
**points to this file** — it is not duplicated. One source of truth.

> Scope note: this is a small, single-binary personal tool. The rules below are
> intentionally lighter than a production service would carry. Keeping them small
> *is* one of the rules — see Article II.

---

## Preamble

`kokoarch-rdp` is a terminal UI for launching RDP sessions. It handles
**credentials** and spawns a **remote-desktop client**, often from untrusted
networks. The rules exist so that a momentary convenience never becomes a
credential leak, and so the tool stays small enough that one person can hold it
in their head.

This document states the **principles** (Articles) and the **enforceable rules**
that follow from each. When a rule and a principle seem to disagree, the
principle wins and the rule is the bug.

---

## Articles

### Article I — Credentials are sacred

The password is the asset. Protect it above convenience.

- **Never pass the password on the command line** (`/p:`, args, or env that shows
  in `ps`). Feed it to xfreerdp via `/from-stdin` over a pipe.
- **Never write the password to disk** — not in `rdp-sessions.json`, not in logs,
  not in temp files. The only persistence is the system keyring via `go-keyring`.
- **Never log secrets.** No password, no full keyring contents, in any output or
  debug line.
- Clear the in-memory password as soon as it has been handed to the child process.

### Article II — Stay small

This is a launcher, not a platform. Simplicity is a feature.

- Prefer the simplest thing that works. If a file grows an abstraction it doesn't
  yet need, that's a finding.
- No feature beyond what was asked. No configurability "for later." No error
  handling for impossible states.
- Ask: *would a senior engineer call this overcomplicated?* If yes, simplify.
- This rule applies to the docs too: do not grow this repo into Marvin-sized
  governance. One `AGENTS.md`, one `CLAUDE.md` pointer, one `PLAN.md`.

### Article III — Compatibility with launcher data

The Go TUI replaces `kokoarch/bin/kokoarch-rdp`. Keep session data and launch
behavior compatible where it matters, but the password backend is owned by this
app.

- **Same config file**: `~/.config/kokoarch/rdp-sessions.json`, same schema
  (`user`, `host`, `lastUsed`, `ignore_cert`). See [PLAN.md](PLAN.md).
- **Same username munge**: prefix `.\` when no `\` is present before passing the
  username to FreeRDP or deriving the keyring entry identity.
- **Same xfreerdp flags** as the bash launcher (see PLAN.md). Changes to the flag
  set are a deliberate decision, noted in the commit.

### Article IV — Truthfulness

Docs describe the code that exists, not the code that was planned.

- When code and docs disagree, one of them is a bug — fix it in the same change.
- `PLAN.md` is a plan; once something is built, the README (not the plan) is the
  source of truth for how it actually behaves.

### Article V — Quality bar

- `go build`, `go vet`, and `gofmt -l` must all be clean before a change is done.
- The credential path (keyring lookup/store, password piping) gets a test or a
  documented manual verification — it is the one part where a silent bug is a
  security bug.
- Errors are surfaced, not swallowed. A failed keyring or spawn call tells the
  user what happened.

### Article VI — Dependencies

In the era of agents, prefer a few lines inline over a new dependency.

Approved baseline:
- `github.com/charmbracelet/bubbletea`, `bubbles`, `lipgloss` — the TUI.
- `github.com/zalando/go-keyring` — native OS keyring access.
- Standard library for everything else (`os/exec` for `xfreerdp3`, `encoding/json`
  for config).

Anything outside this list needs a one-line justification in the commit: what it
solves, and why inline isn't viable.

### Article VII — Amendments

Change this file deliberately, not by drive-by edit. An amendment states:
1. What prompted it (what went wrong, or what we learned).
2. The new/revised rule in full.
3. How it interacts with existing articles (extend / narrow / supersede).

---

## Precedence

When two sources disagree, the higher wins:

1. **This file** (`AGENTS.md`) — principles and rules.
2. **[PLAN.md](PLAN.md)** — the build plan and compatibility contract.
3. **`~/.claude/CLAUDE.md`** — the user's global engineering philosophy (it
   reinforces Article II).
4. **Code comments / README** — local detail; never override a rule.

A doc or plan that contradicts this file is the bug.

---

## Git discipline

`git push`, `git commit`, `git rebase`, `git reset`, destructive `git checkout`,
force-push, and merge/tag operations are gated on **explicit approval** (see
`~/.claude/CLAUDE.md` § Git). Commit messages are one short sentence, no
`Co-Authored-By` lines. Don't commit unless asked.

---

## How to read this repo

1. **This file** — the rules.
2. **[PLAN.md](PLAN.md)** — what's being built and the compatibility contract.
3. **README.md** — how the finished tool actually works (once it exists).
4. The code under `internal/`.
