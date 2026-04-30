---
title: "OverlayChrome migration: devices + help + search overlays"
feature: 13-tui-design-system
status: done
---

## Background

Story 185 migrated the bordered-pane callers (`PaneChrome`). This story migrates the
modal-overlay callers (`OverlayChrome`):

- **`internal/ui/panes/devices.go:166–175`** — direct `layout.RenderPaneBorder` for the
  device-switch overlay, no glyph fields.
- **`internal/ui/panes/help_overlay.go:152–161`** — direct `layout.RenderPaneBorder`
  for the help overlay, no glyph fields.
- **`internal/ui/panes/search.go:778, 880, 943`** — three direct `RenderPaneBorder`
  calls for the search overlay (one per result-state: idle / loading / results), no
  glyph fields.

After this story, every overlay surface in the app routes through
`uikit.OverlayChrome.Render`. Combined with story 185, the only callers of
`layout.RenderPaneBorder` left in the app are the four `uikit` chrome primitives.

The chrome CI guard (`scripts/check-render-pane-border.sh`) that enforces this rule
ships in story 192, not here, to avoid a merge-order trap if 186 lands before 185.

**Depends on:** story 183 (catalogue audit). Implicitly assumes 185 has landed (so
`renderGrid` is migrated) but does not block on it — the two can be reviewed in
parallel and merged in either order; the CI guard in 192 is what enforces the
combined invariant.

**Plan tasks:** 3.3, 3.4, 3.5 in `docs/superpowers/plans/2026-04-29-glyph-fallback.md`.

**Files:** `internal/ui/panes/devices.go`, `internal/ui/panes/devices_test.go`,
`internal/ui/panes/help_overlay.go`, `internal/ui/panes/help_overlay_test.go`,
`internal/ui/panes/search.go`, `internal/ui/panes/search_test.go`.

## Design

### `devices.go` — device-switch overlay

Replace the `layout.RenderPaneBorder(inner, cfg)` block at lines 166–175 with:

```go
chrome := uikit.OverlayChrome{
    Width:   o.width,
    Height:  o.height,
    Title:   "Devices",
    Actions: o.Actions(),
    Theme:   o.theme,
}
return chrome.Render(inner)
```

(Title / Actions match the existing border configuration. The inline `BorderConfig`
literal and the `layout.RenderPaneBorder` call are deleted.)

### `help_overlay.go`

Same pattern at lines 152–161:

```go
chrome := uikit.OverlayChrome{
    Width:  h.width,
    Height: h.height,
    Title:  "Help",
    Theme:  h.theme,
}
return chrome.Render(inner)
```

### `search.go` — three overlay-render paths

Lines 778, 880, 943 each render a different search-result state (the audit identified
all three need to migrate). Each call site receives the same substitution:

```go
chrome := uikit.OverlayChrome{
    Width:   s.width,
    Height:  s.height,
    Title:   "Search",
    Actions: s.Actions(),
    Theme:   s.theme,
}
return chrome.Render(content)
```

Title may differ between sites — match what the existing border title was at each line.

### Test pattern

`devices_test.go`, `help_overlay_test.go`, `search_test.go` each gain a per-overlay
`_AsciiBorder` test:

```go
uikit.SetModeForTest(uikit.GlyphASCII)
defer uikit.SetModeForTest(uikit.GlyphUnicode)

o := NewDevicesOverlay(theme.Load("black"))
o.SetSize(50, 20)
out := stripANSI(o.View())
if strings.ContainsAny(out, "╭╮╰╯─│") {
    t.Errorf("ascii overlay must not contain unicode borders, got: %q", out)
}
```

For `search.go`, one test per state-route (idle / loading / results) so all three
migrated call sites are covered.

## Acceptance Criteria

- [ ] `panes/devices.go:166–175` calls `uikit.OverlayChrome.Render`; no inline
      `BorderConfig` literal or direct `layout.RenderPaneBorder` call remains
- [ ] `panes/help_overlay.go:152–161` calls `uikit.OverlayChrome.Render`; same
- [ ] `panes/search.go` lines 778, 880, 943 each call `uikit.OverlayChrome.Render`;
      no direct `RenderPaneBorder` calls remain in `search.go`
- [ ] Each overlay's existing render contract (Title text, Actions list, dimensions)
      is preserved — the migration is functionally invisible in unicode mode
- [ ] New tests `TestDevicesOverlay_AsciiBorder`, `TestHelpOverlay_AsciiBorder`,
      `TestSearchOverlay_AsciiBorder` (one per state) confirm no `╭╮╰╯─│` in ASCII
      mode
- [ ] All existing search/help/devices overlay tests pass unchanged
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Tasks 3.3, 3.4, 3.5 in `docs/superpowers/plans/2026-04-29-glyph-fallback.md`.

- [ ] Branch: `feat/13-overlay-chrome-migration`
- [ ] Write failing `TestDevicesOverlay_AsciiBorder` → FAIL
- [ ] Migrate `panes/devices.go:166–175` to `uikit.OverlayChrome.Render` → PASS
- [ ] Commit: `fix(devices): migrate overlay to uikit.OverlayChrome for ascii fallback`
- [ ] Write failing `TestHelpOverlay_AsciiBorder` → FAIL
- [ ] Migrate `panes/help_overlay.go:152–161` to `uikit.OverlayChrome.Render` → PASS
- [ ] Commit: `fix(help): migrate help overlay to uikit.OverlayChrome`
- [ ] Write failing `TestSearchOverlay_AsciiBorder` covering each render state → FAIL
- [ ] Migrate the three `panes/search.go` call sites (lines 778, 880, 943) to
      `uikit.OverlayChrome.Render` → PASS
- [ ] Commit: `fix(search): migrate three search-overlay render paths to OverlayChrome`
- [ ] `make ci` → PASS
- [ ] Push branch + open PR
