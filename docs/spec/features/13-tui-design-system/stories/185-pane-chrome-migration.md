---
title: "PaneChrome migration: renderGrid + themes + profile + infobox"
feature: 13-tui-design-system
status: open
---

## Background

`internal/ui/layout/border.go:71–97` already implements glyph fallback correctly:
`BorderConfig` exposes `CornerTL/TR/BL/BR/HRule/VRule/ToggleKeyStr` fields, and
`resolveGlyphs` falls back to hardcoded unicode (`╭╮╰╯─│`) when any field is empty.
`uikit.PaneChrome.Render` populates them via `GlyphFor`. The contract is sound.

**The callers do not fulfil it.** Almost every visible pane border on the running app
today is rendered by a direct `layout.RenderPaneBorder` call that never populates the
glyph fields, so every border falls through to the unicode defaults even when
`ui.glyphs = "ascii"`.

Audit §3.5 identified the offenders. This story fixes the four `PaneChrome`-shaped
ones (the bordered-pane variant, not the modal-overlay variant — story 186 covers
those):

1. **`internal/app/render.go:417–426` — `renderGrid`.** The single largest fallback gap
   in the app. `renderGrid` builds `BorderConfig` for every grid pane (10 panes × 2
   pages) and calls `RenderPaneBorder` directly without `CornerTL/…/HRule/VRule/
   ToggleKeyStr`. Result: every grid pane border is hardcoded unicode in ASCII mode.
2. **`internal/ui/panes/themes.go:160`** — direct `RenderPaneBorder` call, no glyph fields.
3. **`internal/ui/panes/profile.go:183`** — same pattern.
4. **`internal/ui/components/infobox.go:99,149,151,162,169,172,183`** — hand-rolls its
   own border with hardcoded `╭╮╰╯─│` literals. Bypasses `layout.RenderPaneBorder`
   entirely.

After this story, every bordered (non-overlay) surface in the app routes through
`uikit.PaneChrome.Render`, and `infobox.go` becomes a thin wrapper.

**Depends on:** story 183 (no new catalogue rows used here, but the regression-test
pattern relies on the catalogue audit being green).

**Plan tasks:** 3.1, 3.2, 3.6 in `docs/superpowers/plans/2026-04-29-glyph-fallback.md`.
Task 3.7 (the chrome CI guard) ships in story 192 to avoid a merge-order trap (the
guard would fail against renderGrid until 185 lands).

**Files:** `internal/app/render.go`, `internal/app/render_test.go`,
`internal/ui/panes/themes.go`, `internal/ui/panes/themes_test.go`,
`internal/ui/panes/profile.go`, `internal/ui/panes/profile_test.go`,
`internal/ui/components/infobox.go`, `internal/ui/components/infobox_test.go`.

## Design

### `renderGrid` migration

In `internal/app/render.go` `renderGrid` (lines 411–434 region), replace the inline
`BorderConfig` build + `layout.RenderPaneBorder` call with `uikit.PaneChrome.Render`:

```go
chrome := uikit.PaneChrome{
    Width:       rect.Width,
    Height:      rect.Height,
    Title:       pane.Title(),
    ToggleKey:   pane.ToggleKey(),
    Actions:     pane.Actions(),
    AccentColor: layout.PaneBorderColor(paneID, a.theme),
    Focused:     pane.IsFocused(),
    Theme:       a.theme,
}
if fqp, ok := pane.(layout.FilterQueryPane); ok {
    chrome.FilterQuery = fqp.ActiveFilterQuery()
}
bordered := chrome.Render(pane.View())
```

Add `"github.com/initgrep-apps/spotnik/internal/uikit"` to the import block.

### `themes.go` and `profile.go`

Both panes own their full pane geometry (single-pane page or full-pane overlay
depending on the page). The migration pattern:

```go
chrome := uikit.PaneChrome{
    Width:       p.width,
    Height:      p.height,
    Title:       p.Title(),
    ToggleKey:   p.ToggleKey(),
    Actions:     p.Actions(),
    AccentColor: layout.PaneBorderColor(p.paneID, p.theme),
    Focused:     p.focused,
    Theme:       p.theme,
}
return chrome.Render(inner)
```

Field references match what the existing code uses for theme, dimensions, focus.

### `infobox.go` rewrite

`InfoBox.Render` strips its hand-rolled corner / rule literals and delegates entirely
to `uikit.PaneChrome.Render`:

```go
func (b InfoBox) Render(content string) string {
    chrome := uikit.PaneChrome{
        Width:       b.Width,
        Height:      b.Height,
        Title:       b.Title,
        AccentColor: b.AccentColor,
        Theme:       b.Theme,
    }
    return chrome.Render(content)
}
```

The `InfoBox` struct keeps its public-facing field set; only the rendering path
changes. The struct becomes a thin compatibility wrapper for callers that constructed
`InfoBox` literals.

### Regression-test pattern

Each migration adds a single ASCII snapshot test asserting:

1. `strings.ContainsAny(out, "╭╮╰╯─│")` is `false` in ASCII mode.
2. The output contains `+` (corner) and `-` (rule) in ASCII mode.
3. `Width` / `Height` constraints still hold after the swap.

For `renderGrid`, the test runs against the full app render so all 10 grid panes are
covered in one assertion sweep.

## Acceptance Criteria

- [ ] `internal/app/render.go` imports `internal/uikit`
- [ ] `renderGrid` builds a `uikit.PaneChrome{...}` per visible pane and calls
      `chrome.Render(pane.View())`; the inline `layout.BorderConfig{...}` literal
      and direct `layout.RenderPaneBorder` call are removed from `renderGrid`
- [ ] `FilterQueryPane` integration still passes — panes implementing the interface
      surface their query into `PaneChrome.FilterQuery`
- [ ] `panes/themes.go:160` calls `uikit.PaneChrome.Render`; no direct
      `layout.RenderPaneBorder` call remains in this file
- [ ] `panes/profile.go:183` calls `uikit.PaneChrome.Render`; no direct
      `layout.RenderPaneBorder` call remains in this file
- [ ] `components/infobox.go` `Render` delegates to `uikit.PaneChrome.Render`; lines
      99/149/151/162/169/172/183 (the hand-rolled corner / rule literals) are deleted
- [ ] New test `TestRenderGrid_AsciiBorders` confirms `renderGrid` output in ASCII
      mode contains `+` corners and contains none of `╭╮╰╯`
- [ ] New tests `TestThemesPane_AsciiBorder`, `TestProfilePane_AsciiBorder`,
      `TestInfoBox_AsciiBorder` confirm the same for each surface
- [ ] All existing pane tests still pass (no width/height regressions)
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Tasks 3.1, 3.2, 3.6 in `docs/superpowers/plans/2026-04-29-glyph-fallback.md`.

- [ ] Branch: `feat/13-pane-chrome-migration`
- [ ] Write failing `TestRenderGrid_AsciiBorders` → FAIL
- [ ] Migrate `renderGrid` to `uikit.PaneChrome.Render` (preserving `FilterQuery`
      handling for filter-aware panes) → PASS
- [ ] Commit: `fix(app): route renderGrid through uikit.PaneChrome for ascii fallback`
- [ ] Write failing `TestThemesPane_AsciiBorder` and `TestProfilePane_AsciiBorder` → FAIL
- [ ] Migrate `panes/themes.go:160` and `panes/profile.go:183` to
      `uikit.PaneChrome.Render` → PASS
- [ ] Commit: `fix(panes): migrate themes and profile panes to uikit.PaneChrome`
- [ ] Write failing `TestInfoBox_AsciiBorder` → FAIL
- [ ] Rewrite `components/infobox.go` `Render` to delegate to
      `uikit.PaneChrome.Render`; remove every hardcoded `╭╮╰╯─│` literal → PASS
- [ ] Commit: `refactor(components): InfoBox delegates to uikit.PaneChrome instead of rolling its own border`
- [ ] `make ci` → PASS
- [ ] Push branch + open PR
