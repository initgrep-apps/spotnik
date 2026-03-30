# Feature 15 — Fix UX Polish

> **Bug fix:** Missing view switcher hints in status bar.

## Bug Addressed

| # | Issue | Root Cause |
|---|---|---|
| B14 | View switching (1/2/3) not documented in status bar | Missing from hints |

**Note:** B6 (Tab order L→Q→P feels wrong) — current order P→L→Q is kept per owner decision.
No change needed.

---

## Root Cause

Status bar in all focus states doesn't mention `1`/`2`/`3` for view switching. DESIGN.md lists
these keys in the help overlay but they're not in the status bar context hints.

---

## Fix

- Add `2 stats` and `3 playlists` hints to main view status bar
- Ensure hints appear in all focus states (library, player, queue)
- Keep status bar concise — may abbreviate as `2 stats  3 lists` if space is tight

---

## Files

- `internal/app/app.go` — `renderStatusBar()` or equivalent
- Tests for view switcher hints

---

## Acceptance Criteria

- [ ] Status bar in all focus states shows `2 stats` and `3 playlists` hints
- [ ] Tests verify view switcher hints are present
