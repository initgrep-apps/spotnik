---
title: "Rename Page A/B and Nerd Status to Music/Stats"
feature: 16-mono-themes-page-rename
status: open
---

## Background

Internal identifiers `PageA` / `PageB` and the user-facing preset name `"Nerd Status"` are opaque. Renaming to `Music` / `Stats` makes the UI self-describing. This is a pure refactor — no behavioural changes, no new types, no new dependencies.

**Depends on:** Features 08 (Theming), 14 (Page B Redesign), 15 (Error Resilience) — all done.

**Scope:** Every Go identifier, comment, preset name, test name, documentation reference, and README mention.

---

## Design

### Identifiers

| Old | New |
|---|---|
| `PageA` | `PageMusic` |
| `PageB` | `PageStats` |
| `PageAPresets` | `PageMusicPresets` |
| `PageBPresets` | `PageStatsPresets` |
| `PresetNerdStatus` | `PresetStats` |
| Preset `Name: "Nerd Status"` | `Name: "Stats"` |

### `pageLabel()` Behaviour

```go
func pageLabel(page layout.PageID) string {
    switch page {
    case layout.PageMusic:
        return "Music"
    case layout.PageStats:
        return "Stats"
    default:
        return "?"
    }
}
```

Header bar shows `spotnik ─ Music ─ preset 0` (Music) or `spotnik ─ Stats` (Stats).

### Key Comment Replacements

| Old | New |
|---|---|
| `Page A pane N` | `Music pane N` |
| `Page B pane N` | `Stats pane N` |
| `Page A (10 bindings)` | `Music page (10 bindings)` |
| `Page B (8 bindings)` | `Stats page (8 bindings)` |
| `1-8 on Page A` | `1-8 on Music page` |
| `1-5 on Page B` | `1-5 on Stats page` |
| `Page B panes` | `Stats page panes` |
| `Nerd Status` | `Stats` |

---

## Tasks

### Task 1: `layout` package identifiers and comments

**Files:**
- `internal/ui/layout/pane.go` — rename constants + comments
- `internal/ui/layout/presets.go` — rename vars + comments + preset Name
- `internal/ui/layout/layout.go` — use new constants + comments
- `internal/ui/layout/border.go` — comment update

**Details:**
- `PaneNowPlaying` comment: `Music pane 1` (not `Page A pane 1`)
- `PaneNetworkLog` comment: `Stats pane 5` (not `Page B pane 5`)
- `PageMusic` / `PageStats` constants with updated comments
- `PresetStats` with `Name: "Stats"`
- `PageMusicPresets` / `PageStatsPresets` vars
- `TogglePane` comment: `keys 1-8 on Music page, 1-5 on Stats page`
- `NewManager` uses `PageMusic` / `PageStats`

**Tests:** Rename and update `layout_test.go`, `presets_test.go`, `pane_test.go`, `border_test.go`.

### Task 2: `app` package identifiers, comments, and tests

**Files:**
- `internal/app/render.go` — `pageLabel()`, `appKeyMap` comments, `ShortHelp()`, `FullHelp()`, `renderHeader()`, `renderStatusBar()` comments
- `internal/app/app.go` — comments only (no code changes)
- `internal/app/handlers.go` — comments only
- `internal/app/routing.go` — comments only
- `internal/app/routing_test.go` — test rename + comments
- `internal/app/app_test.go` — test renames + comments
- `internal/app/render_test.go` — test renames + string assertions

**Details:**
- `pageLabel()` returns `"Music"` / `"Stats"`
- `ShortHelp()` checks `layout.PageMusic`
- `FullHelp()` checks `layout.PageMusic`
- `renderHeader()` checks `layout.PageStats` for preset hide
- All `Page A` / `Page B` strings in test assertions become `Music` / `Stats`
- Test names: `TestApp_StatsPage_Panes_Registered`, `TestRenderHeader_MusicPage_ShowsPreset`, etc.

### Task 3: `uikit` comments and tests

**Files:**
- `internal/uikit/header_bar.go` — comments
- `internal/uikit/header_bar_test.go` — test renames + assertions
- `internal/uikit/pane_chrome.go` — comment
- `internal/uikit/status_bar_test.go` — test body strings

**Details:**
- Header bar doc comment uses `Music` / `Stats` examples
- `Preset` field comment: `Stats page` instead of `Page B`
- `header_bar_test.go`: `TestHeaderBar_LeftSegment_Music`, assertion `"Music"` not `"Page A"`

### Task 4: `config` comments

**File:** `internal/config/config.go`
- Update `Preset` field comment: `Music page layout preset index`

### Task 5: `panes` comments

**Files:**
- `internal/ui/panes/base_pane.go` — comments
- `internal/ui/panes/gateway_health_pane.go` — ToggleKey comment
- `internal/ui/panes/polling_traffic_pane.go` — ToggleKey comment
- `internal/ui/panes/gateway_live_pane.go` — ToggleKey comment
- `internal/ui/panes/networklog_pane.go` — ToggleKey comment

### Task 6: Documentation

**Files:**
- `docs/system/design.md` — §4, §17, §19, preset tables, keybinding table
- `docs/system/architecture.md` — Page A/B refs
- `docs/system/tui.md` — header bar examples, Page B labels
- `README.md` — keybinding table + description text
- `CHANGELOG.md` — section headers

**Details:**
- `Page A` → `Music page` (or `Music` where it flows)
- `Page B` → `Stats page` (or `Stats` where it flows)
- `Nerd Status` → `Stats`
- `PageAPresets` → `PageMusicPresets`
- `PageBPresets` → `PageStatsPresets`
- `PresetNerdStatus` → `PresetStats`
- README: `Press 0 to switch between Music and Stats`
- README keybinding table: `Toggle Music / Stats`, `Toggle pane visibility (Music)`, `Toggle pane visibility (Stats)`

---

## Acceptance Criteria

- [ ] `layout.PageMusic` and `layout.PageStats` exist; `layout.PageA` and `layout.PageB` do not
- [ ] `PresetStats.Name == "Stats"`
- [ ] `pageLabel(PageMusic) == "Music"` and `pageLabel(PageStats) == "Stats"`
- [ ] `make ci` passes (lint + tests + 80% coverage)
- [ ] No `Page A`, `Page B`, or `Nerd Status` strings remain in Go source or docs (except CHANGELOG history and git history)
