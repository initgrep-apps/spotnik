---
title: "Mono Themes + Page Rename"
status: done
---

## Description

Two independent polish items bundled for atomic shipping:

1. **Page rename** — Replace opaque internal/user-facing labels `Page A` / `Page B` and `Nerd Status` with `Music` / `Stats`. Every Go identifier, comment, preset name, test name, documentation reference, and README mention is updated. Zero behavioural changes.

2. **Mono themes** — Add `mono-dark` and `mono-light` to the embedded TOML theme set. Both strip all per-pane colour differentiation; every accent is grayscale. No Go code changes — `ConfigTheme` + `Load()` picks them up automatically.

**Design record:** `docs/superpowers/specs/2026-05-14-mono-theme-page-rename-design.md`

**Implementation plan:** `docs/superpowers/plans/2026-05-14-mono-theme-page-rename.md`

## Acceptance Criteria

- [ ] `layout.PageA` renamed to `layout.PageMusic`; `layout.PageB` renamed to `layout.PageStats`
- [ ] `PresetNerdStatus` renamed to `PresetStats`; preset `Name` changed from `"Nerd Status"` to `"Stats"`
- [ ] `pageLabel()` returns `"Music"` / `"Stats"` (not `"A"` / `"B"`)
- [ ] Header bar renders `spotnik ─ Music ─ preset 0` and `spotnik ─ Stats`
- [ ] All comments, test names, and documentation updated to `Music page` / `Stats page`
- [ ] `mono-dark.toml` and `mono-light.toml` are valid TOML themes loaded by `theme.Available()`
- [ ] Both mono themes use uniform neutral gray for all `pane_borders` entries
- [ ] `make ci` passes after every story

## Stories

| # | Story | File |
|---|---|---|
| 207 | Rename Page A/B and Nerd Status to Music/Stats | `stories/207-page-music-stats-rename.md` |
| 208 | Add mono-dark and mono-light themes | `stories/208-mono-themes.md` |
