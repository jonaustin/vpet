# Agent Instructions for this repo (vpet)

> **Codex bootstrap** (run automatically at session start):
> 1) `bd prime --json` for compact context; if it fails, run `bd init --quiet` then retry.
> 2) If `.beads/BD_GUIDE.md` is missing or older than one week, regenerate with `bd onboard --output .beads/BD_GUIDE.md`.
> 3) Use `bd ready --json` to pick work; prefer `bd list --status open --long` if ready is empty.
> 4) Before ending a session: `bd sync` to flush exports, then `git status` and report state.

Repo prefix will auto-detect from bd metadata (current DB lives in `.beads/`).

If bd reports staleness or daemon issues, add `--no-daemon --sandbox` to the command and rerun.
