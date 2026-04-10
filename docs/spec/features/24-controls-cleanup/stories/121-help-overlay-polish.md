---
title: "Help Overlay Polish"
feature: 24-controls-cleanup
status: open
---

## Background

Four cosmetic and clarity improvements to the help overlay and pane borders:

**5A — Lowercase labels.** All label strings in `helpContent` are lowercase ("search",
"quit", "filter"). The user wants title-case for visual consistency.

**5B — Key column not bold.** `renderColumn` builds `keyStyle` without `.Bold(true)`,
so key names render in the same weight as the label text. Bold keys create clearer
visual hierarchy.

**5C — j/k in Navigation is clutter.** `j` and `k` scroll every list pane — they are
so implicit that listing them in the help overlay wastes space and misleads users into
thinking they need the help overlay to discover them. Remove from Navigation section.

**5D — networklog pane border shows j/k.** `networklog_pane.go Actions()` includes
`{Key: "j/k", Label: "scroll"}`. Same reasoning as 5C: implicit, not needed.
The existing `networklog_pane_test.go` asserts on this entry and must be updated.

**Depends on:** Story 120 (removes several Pane Action entries from `helpContent`;
doing this story after 120 avoids re-touching the same entries). Can run independently
but ordering is recommended.

## Design

### help_overlay.go — renderColumn

Add `.Bold(true)` to `keyStyle` (line 173):
```go
keyStyle := lipgloss.NewStyle().Foreground(o.theme.KeyHint()).Bold(true)
```

### help_overlay.go — helpContent

**Capitalize all label strings.** Full mapping (after Story 120 removals):

Global section:
- `"search"` → `"Search"`
- `"devices"` → `"Devices"`
- `"profile"` → `"Profile"`
- `"theme"` → `"Theme"`
- `"help"` → `"Help"`
- `"quit"` → `"Quit"`
- `"toggle page"` → `"Toggle page"`
- `"toggle pane"` → `"Toggle pane"`
- `"preset"` → `"Preset"`

Navigation section (after removing j/k):
- `"next pane"` → `"Next pane"`
- `"prev pane"` → `"Prev pane"`
- `"close overlay"` → `"Close overlay"`

Playback section (after Story 118 removes "n"):
- `"play / pause"` → `"Play / Pause"`
- `"prev / next"` → `"Prev / Next"`
- `"volume"` → `"Volume"`
- `"shuffle"` → `"Shuffle"`
- `"repeat"` → `"Repeat"`
- `"visualizer"` → `"Visualizer"`

Pane Actions section (after Story 120 removals, after Story 119 adds g):
- `"select / play"` → `"Select / Play"`
- `"filter"` → `"Filter"`
- `"Cycle time range"` → already title-case from Story 119

**Remove j/k from Navigation section:**
```go
{title: "Navigation", bindings: []helpBinding{
    {"Tab", "Next pane"}, {"Shift+Tab", "Prev pane"},
    {"Esc", "Close overlay"},
}},
```

### networklog_pane.go

**`Actions()` default branch** (lines 94–97): remove `{Key: "j/k", Label: "scroll"}`:
```go
return []layout.Action{
    {Key: "f", Label: "filter"},
}
```

### Keybinding docs — j/k removal only (same commit)

Capitalization and bold are cosmetic UI changes only — no keybinding semantics change,
so only the j/k removal requires the three-location sync:

**`docs/keybinding.md`** Navigation section: remove `| j / k | Scroll down / up |`

**`docs/DESIGN.md §17`**: remove `j / k` from the navigation keybinding table.

**`internal/ui/panes/help_overlay.go`**: already updated above (Navigation binding removed).

## Acceptance Criteria

- [ ] All label strings in the help overlay are title-case (verified visually or via string assertions in tests)
- [ ] Key column in the help overlay renders bold (verified via `lipgloss` style inspection in test or via snapshot)
- [ ] `j / k` does not appear in the Navigation section of the help overlay
- [ ] NetworkLog pane border does not show `j/k` hint when filter is inactive
- [ ] NetworkLog pane border still shows `f: filter` when filter is inactive
- [ ] `docs/keybinding.md` and `docs/DESIGN.md §17` no longer list `j / k` in Navigation
- [ ] `make ci` passes

## Tasks

- [ ] Add `TestHelpOverlay_Labels_TitleCase` to `internal/ui/panes/help_overlay_test.go` — assert that none of the label strings in `helpContent` are all-lowercase (e.g., fail if `"search"` is present, pass if `"Search"` is present)
  - test: `go test ./internal/ui/panes/... -run TestHelpOverlay_Labels_TitleCase -v` → FAIL
- [ ] Capitalize all label strings in `helpContent`
  - test: title-case test → PASS
- [ ] Add `TestHelpOverlay_Navigation_NoJK` — verify the Navigation section has no binding with key `"j / k"` or `"j/k"`
  - test: → FAIL
- [ ] Remove `{"j / k", "scroll"}` from Navigation bindings in `helpContent`
  - test: no-jk test → PASS
- [ ] Add `.Bold(true)` to `keyStyle` in `renderColumn`
  - test: run `go build ./...` → clean (bold flag does not require a unit test)
- [ ] Update `TestNetworkLogPane_Actions` in `networklog_pane_test.go` to not assert on `"j/k"` key presence
- [ ] Remove `{Key: "j/k", Label: "scroll"}` from `networklog_pane.go Actions()`
  - test: `go test ./internal/ui/panes/... -run TestNetworkLogPane -v` → PASS
- [ ] Update `docs/keybinding.md`, `docs/DESIGN.md §17`, and confirm `help_overlay.go` is consistent — all in a single commit
  - test: `go build ./...` clean; grep confirms `j / k` does not appear in Navigation sections of all three
- [ ] `make ci` passes
