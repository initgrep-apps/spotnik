---
title: "Remaining inline glyph leaks + EmptyState migrations + ProgressBar + table/viz block"
feature: 13-tui-design-system
status: done
---

## Background

After stories 183–190 land, every chrome surface, every catalogue add, and every
search/devices content path resolves through `GlyphFor`. The audit (§3.2 / §3.3)
identified one final mechanical sweep — small literal-→-`GlyphFor` swaps and one
custom-bar-rendering migration — that closes the remaining inline-glyph leaks across
panes and components.

Bundling these together: each is a few lines of change, the tests are independent,
and the final CI guard in story 192 is what enforces "no catalogue characters outside
`uikit/glyph.go`". This story is the last fix needed before the guard runs green.

The fixes fall into four groups:

1. **Empty-state migrations** — `panes/recentlyplayed_pane.go:134` and
   `panes/nowplaying.go:314–319` ship custom "no data" strings; both migrate to
   `uikit.EmptyState`.
2. **Inline glyph swaps** — `panes/networklog_pane.go:275,277` (`◷`/`⚡`),
   `panes/profile.go:228` (`…`), `panes/help_overlay.go:140` (`│`),
   `components/gradient.go:197,200` (`♪`), `app/render.go:132` (banner `♪`),
   `:320–322` (`•` × 3), `:526,551` (`…`), `components/table.go:13`
   (`const playingSymbol = "▶"` → lazy-resolve), `components/viz/block.go:45`
   (`█`).
3. **ProgressBar migration** — `panes/gateway_health_pane.go:169–184` rolls its own
   dot-bar for capacity. Migrate to `uikit.ProgressBar`.
4. **Smoke test** — single ASCII-mode app render asserts the overall app output
   contains no `♪`, `•`, `…` literals.

The reviewer-visible diff is large but mechanical. Story 192's CI guards catch any
miss.

**Depends on:** story 183 (catalogue audit). All glyphs swapped here use roles already
in `glyph.go` (no new catalogue rows needed beyond what 183 added).

**Plan tasks:** 5.4, 5.5, 5.6, 5.7 in `docs/superpowers/plans/2026-04-29-glyph-fallback.md`.

**Files:** `internal/ui/panes/recentlyplayed_pane.go`, `internal/ui/panes/nowplaying.go`,
`internal/ui/panes/networklog_pane.go`, `internal/ui/panes/profile.go`,
`internal/ui/panes/help_overlay.go`, `internal/ui/panes/gateway_health_pane.go`,
`internal/ui/components/gradient.go`, `internal/ui/components/table.go`,
`internal/ui/components/viz/block.go`, `internal/app/render.go`,
`internal/app/render_test.go`.

## Design

### Empty states

`panes/recentlyplayed_pane.go:134`:

```go
if len(p.tracks) == 0 {
    return uikit.EmptyState{
        Text:   "No recently played tracks",
        Hint:   "Listen to something to populate this list",
        Width:  p.width,
        Height: p.height,
        Theme:  p.theme,
    }.Render()
}
```

`panes/nowplaying.go:314–319` — replace the "Nothing playing" custom render with the
same `uikit.EmptyState` pattern.

### Inline glyph swaps

| File | Replacement |
|---|---|
| `panes/networklog_pane.go:275,277` | `uikit.GlyphFor(uikit.GlyphDeadline, m)` / `uikit.GlyphFor(uikit.GlyphRunning, m)` |
| `panes/profile.go:228` truncation | `ell := uikit.GlyphFor(uikit.GlyphEllipsis, m); return s[:max-len(ell)] + ell` |
| `panes/help_overlay.go:140` divider | `uikit.GlyphFor(uikit.GlyphVRule, m)` |
| `components/gradient.go:197,200` | `uikit.GlyphFor(uikit.GlyphMusicNote, m)` |
| `app/render.go:132` banner | `uikit.GlyphFor(uikit.GlyphMusicNote, m) + "  spotnik"` |
| `app/render.go:320–322` bullets | `bullet := uikit.GlyphFor(uikit.GlyphBullet, m); …` |
| `app/render.go:526,551` ellipsis | `ell := uikit.GlyphFor(uikit.GlyphEllipsis, m); …` |
| `components/table.go:13` | `func playingSymbol() string { return uikit.GlyphFor(uikit.GlyphPlaying, uikit.ActiveMode()) }` (rename from const → func; update every reference to `playingSymbol()`) |
| `components/viz/block.go:45` `█` | `uikit.GlyphFor(uikit.GlyphBarFull, uikit.ActiveMode())` |

### `gateway_health_pane.go` — dot-bar → `uikit.ProgressBar`

`:169–184`:

```go
func renderDotBar(progress float64, width int, th theme.Theme) string {
    return uikit.ProgressBar{
        Width:    width,
        Progress: progress,
        Theme:    th,
    }.Render()
}
```

If the existing helper takes more parameters (e.g. an accent colour), keep the public
signature stable; the goal is "swap implementation, keep call sites unchanged."

### Smoke test

`internal/app/render_test.go`:

```go
func TestRender_AsciiInlineGlyphs(t *testing.T) {
    uikit.SetModeForTest(uikit.GlyphASCII)
    defer uikit.SetModeForTest(uikit.GlyphUnicode)

    a := newTestApp(t)
    out := stripANSI(a.renderAll())
    for _, banned := range []string{"♪", "•", "…"} {
        if strings.Contains(out, banned) {
            t.Errorf("ascii output must not contain %q", banned)
        }
    }
}
```

## Acceptance Criteria

- [ ] `panes/recentlyplayed_pane.go:134` and `panes/nowplaying.go:314–319` render
      empty states through `uikit.EmptyState`; no custom string concatenation
- [ ] `panes/networklog_pane.go:275,277` resolve `◷` and `⚡` via `uikit.GlyphFor`;
      no raw literals remain
- [ ] `panes/profile.go:228` truncation uses `uikit.GlyphFor(GlyphEllipsis, mode)`
- [ ] `panes/help_overlay.go:140` divider uses `uikit.GlyphFor(GlyphVRule, mode)`
- [ ] `components/gradient.go:197,200` resolve `♪` via `uikit.GlyphFor(GlyphMusicNote, mode)`
- [ ] `app/render.go:132` banner, `:320–322` bullets, `:526,551` ellipsis all
      resolve via `uikit.GlyphFor`
- [ ] `components/table.go` `playingSymbol` is a function (not a const) returning
      `uikit.GlyphFor(GlyphPlaying, ActiveMode())`; every reference updated
- [ ] `components/viz/block.go:45` `█` resolves via `uikit.GlyphFor(GlyphBarFull, mode)`
- [ ] `panes/gateway_health_pane.go:169–184` delegates the capacity bar to
      `uikit.ProgressBar`; the custom dot-rendering loop is removed
- [ ] Smoke test `TestRender_AsciiInlineGlyphs` confirms ASCII app render contains
      none of `♪`, `•`, `…`
- [ ] All existing tests in `internal/app/`, `internal/ui/panes/`,
      `internal/ui/components/`, `internal/ui/components/viz/` pass
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Tasks 5.4–5.7 in `docs/superpowers/plans/2026-04-29-glyph-fallback.md`.

- [ ] Branch: `fix/13-remaining-glyph-leaks`
- [ ] Migrate `panes/recentlyplayed_pane.go:134` and `panes/nowplaying.go:314–319`
      empty messages to `uikit.EmptyState`
- [ ] Run `go test ./internal/ui/panes/ -run "TestRecentlyPlayed|TestNowPlaying" -v`
      → PASS
- [ ] Commit: `fix(panes): migrate recentlyplayed and nowplaying empty states to uikit.EmptyState`
- [ ] Apply the inline glyph swaps in `networklog_pane.go`, `profile.go`,
      `help_overlay.go`, `gradient.go`, `app/render.go`
- [ ] Add `TestRender_AsciiInlineGlyphs`
- [ ] Run `go test ./internal/app/ ./internal/ui/panes/ ./internal/ui/components/ -v`
      → PASS
- [ ] Commit: `fix(ui): route remaining inline glyph leaks through GlyphFor`
- [ ] Migrate `gateway_health_pane.go:169–184` dot-bar to `uikit.ProgressBar`
- [ ] Run `go test ./internal/ui/panes/ -run TestGatewayHealth -v` → PASS
- [ ] Commit: `refactor(gateway-health): use uikit.ProgressBar for capacity bar`
- [ ] Convert `components/table.go:13` `const playingSymbol` to a function;
      replace every reference; swap `viz/block.go:45` `█` literal for
      `uikit.GlyphFor`
- [ ] Run `go test ./internal/ui/components/ ./internal/ui/components/viz/ -v` → PASS
- [ ] Commit: `fix(components): lazy-resolve playing symbol and viz block fill via GlyphFor`
- [ ] `make ci` → PASS
- [ ] Push branch + open PR
