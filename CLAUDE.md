# kokoarch-rdp

A Go TUI for launching RDP sessions, replacing the `kokoarch-rdp` bash launcher.

## Read first

**[AGENTS.md](AGENTS.md) is the single source of truth** for this repo's rules,
constitution, and conventions — read it before doing any work here. This
`CLAUDE.md` only points to it; the rules are not duplicated.

- Rules & constitution → **[AGENTS.md](AGENTS.md)**
- Build plan & compatibility contract → **[PLAN.md](PLAN.md)**

## Start of every session

1. Read [AGENTS.md](AGENTS.md).
2. Read [PLAN.md](PLAN.md) if working on the implementation.
3. Honour the user's global philosophy in `~/.claude/CLAUDE.md` (prefer simple).

## Non-negotiables (full text in AGENTS.md)

- Password never on the command line (`/from-stdin`, never `/p:`), never on disk
  — keyring only.
- Stay compatible with the bash launcher's JSON and keyring entries.
- Keep it small. A launcher, not a platform.
