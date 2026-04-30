---
title: "PlaybackControls primitive + controls.go wrapper + nowplaying title glyphs"
feature: 13-tui-design-system
status: done
---

## Background

`internal/ui/components/controls.go:36–57` hand-rolls the transport-controls strip
with seven hardcoded glyphs (`⇄`, `⏸`, `▷`, `≡`, `↻¹`, `↻`) — no mode check, no
fallback. Under `ui.glyphs = "ascii"` the strip renders as raw unicode regardless of
config.

`internal/ui/panes/nowplaying.go:85,87,91` `Title()` similarly hardcodes the
playback-state glyphs (`▶`, `⏸`) and the `─` separator.

The audit (§3.3 / §4.2) recommends introducing a `uikit.PlaybackControls` primitive
that owns the seven transport glyphs and resolves them through `GlyphFor`, mirroring
the active-state colour role from the design-system role matrix. `controls.go` then
becomes a thin compatibility wrapper that translates the legacy string repeat-mode
arg (`"off"` / `"context"` / `"track"`) to the typed `uikit.RepeatMode` enum.

`nowplaying.go:Title()` migrates in the same story because it shares the same
playback-glyph domain — bundling the three changes keeps the playback transport-
glyph migration in one PR.

`docs/TUI-DESIGN-SYSTEM.md` gains a §3.19 entry documenting the primitive (per
CLAUDE.md rule 17 — new primitive ships with its doc row in the same commit).

**Depends on:** story 183 (catalogue audit). The primitive uses existing playback
roles (`GlyphPlaying`, `GlyphPaused`, `GlyphPausedPB`, `GlyphShuffle`, `GlyphQueue`,
`GlyphRepeatAll`, `GlyphRepeatOne`, `GlyphRepeatOff`) all of which are already in
`glyph.go`.

**Plan tasks:** 4.1, 4.2, 4.4 in `docs/superpowers/plans/2026-04-29-glyph-fallback.md`.

**Files created:** `internal/uikit/playback_controls.go`,
`internal/uikit/playback_controls_test.go`. **Modified:**
`internal/ui/components/controls.go`, `internal/ui/components/controls_test.go`,
`internal/ui/panes/nowplaying.go`, `internal/ui/panes/nowplaying_test.go`,
`docs/TUI-DESIGN-SYSTEM.md`.

## Design

### `uikit.PlaybackControls` — new primitive

```go
type RepeatMode int

const (
    RepeatOff RepeatMode = iota // ⟳ rendered in inactive (TextSecondary) colour
    RepeatAll                   // ↻ rendered in active (PlayingIndicator) colour
    RepeatOne                   // ↻¹ rendered in active colour
)

type PlaybackControls struct {
    Playing    bool
    Shuffle    bool
    RepeatMode RepeatMode
    Theme      theme.Theme
}

func (c PlaybackControls) Render() string {
    m := ActiveMode()
    activeStyle   := lipgloss.NewStyle().Foreground(c.Theme.PlayingIndicator())
    inactiveStyle := lipgloss.NewStyle().Foreground(c.Theme.TextSecondary())

    pickStyle := func(active bool) lipgloss.Style {
        if active {
            return activeStyle
        }
        return inactiveStyle
    }

    shuffle := pickStyle(c.Shuffle).Render(GlyphFor(GlyphShuffle, m))

    var playPause string
    if c.Playing {
        playPause = activeStyle.Render(GlyphFor(GlyphPaused, m))
    } else {
        playPause = inactiveStyle.Render(GlyphFor(GlyphPausedPB, m))
    }

    queue := inactiveStyle.Render(GlyphFor(GlyphQueue, m))

    var repeat string
    switch c.RepeatMode {
    case RepeatOne:
        repeat = activeStyle.Render(GlyphFor(GlyphRepeatOne, m))
    case RepeatAll:
        repeat = activeStyle.Render(GlyphFor(GlyphRepeatAll, m))
    default:
        repeat = inactiveStyle.Render(GlyphFor(GlyphRepeatOff, m))
    }

    return shuffle + "  " + playPause + "  " + queue + "  " + repeat
}
```

The strip uses two-space gaps between icons. Width arithmetic relies on
`lipgloss.Width` upstream (the strip is composed by callers within an outer styled
block).

The visual change vs. the existing `controls.go`: the catalogue separates
`GlyphRepeatOff` (`⟳` / `ro`) from `GlyphRepeatAll` (`↻` / `rp`). The previous
`controls.go` rendered `↻` for both off and all states with different colours; the
new primitive uses the catalogue's intent-distinct glyphs. Reviewer-visible diff —
flag in the PR description.

### `components.Controls` — compatibility wrapper

Replace the body of `internal/ui/components/controls.go` entirely:

```go
package components

import (
    "github.com/initgrep-apps/spotnik/internal/uikit"
    "github.com/initgrep-apps/spotnik/internal/ui/theme"
)

type Controls struct {
    inner uikit.PlaybackControls
}

func NewControls(t theme.Theme, isPlaying, shuffleOn bool, repeatMode string) Controls {
    var rm uikit.RepeatMode
    switch repeatMode {
    case "track":
        rm = uikit.RepeatOne
    case "context":
        rm = uikit.RepeatAll
    default:
        rm = uikit.RepeatOff
    }
    return Controls{
        inner: uikit.PlaybackControls{
            Playing:    isPlaying,
            Shuffle:    shuffleOn,
            RepeatMode: rm,
            Theme:      t,
        },
    }
}

func (c Controls) Render() string { return c.inner.Render() }
```

Existing call sites in `internal/ui/panes` continue to call `NewControls(...)` with the
same signature; the wrapper translates the string arg to the typed enum so callers do
not change yet.

### `nowplaying.go:Title()` migration

`internal/ui/panes/nowplaying.go:85,87,91`:

```go
m := uikit.ActiveMode()
var stateGlyph string
if p.isPlaying {
    stateGlyph = uikit.GlyphFor(uikit.GlyphPaused, m)
} else {
    stateGlyph = uikit.GlyphFor(uikit.GlyphPlaying, m)
}
sep := uikit.GlyphFor(uikit.GlyphHRule, m)
return fmt.Sprintf("%s %s %s %s", stateGlyph, p.trackName, sep, p.artistName)
```

### `docs/TUI-DESIGN-SYSTEM.md` §3.19

New section after §3.18 (Spinner) — documents the primitive's purpose, fields,
unicode + ASCII rendering snapshots, role mapping, glyph table, test contract.

## Acceptance Criteria

- [ ] `internal/uikit/playback_controls.go` defines `RepeatMode` (`RepeatOff`,
      `RepeatAll`, `RepeatOne`) and `PlaybackControls` struct with `Playing`,
      `Shuffle`, `RepeatMode`, `Theme` fields and a `Render() string` method
- [ ] `PlaybackControls.Render()` resolves all four positions (shuffle, play/pause,
      queue, repeat) via `GlyphFor`; no raw glyph literals appear in
      `playback_controls.go`
- [ ] Active glyph styling uses `theme.PlayingIndicator()`; inactive uses
      `theme.TextSecondary()`
- [ ] `playback_controls_test.go` includes:
      - `TestPlaybackControls_RenderUnicode_Playing` — output contains `⏸`, `≡`,
        `⇄`, `⟳` (off-state default)
      - `TestPlaybackControls_RenderASCII_Playing` — output contains `||`, `Q`,
        `sh`, `ro`; no `⏸`, `≡`, `⇄`, `↻`, `⏷` present
      - `TestPlaybackControls_RepeatModes` — `RepeatOff` → `⟳` / `ro`; `RepeatAll`
        → `↻` / `rp`; `RepeatOne` → `↻¹` / `rp1`
- [ ] `internal/ui/components/controls.go` is rewritten as a compatibility wrapper
      that translates `"off"` / `"context"` / `"track"` (and any unknown value) to
      `uikit.RepeatMode`; `Render()` delegates to `inner.Render()`
- [ ] `controls_test.go` covers `TestNewControls_RepeatModeTranslation` for all four
      string inputs
- [ ] `panes/nowplaying.go:85,87,91` resolve playback state and rule glyphs via
      `uikit.GlyphFor`; the literals `▶`, `⏸`, `─` are gone from `Title()`
- [ ] New test `TestNowPlaying_AsciiTitle` confirms ASCII output of `Title()`
      contains `||` (paused glyph) when playing and contains no `▶⏸─`
- [ ] `docs/TUI-DESIGN-SYSTEM.md` gains §3.19 documenting `PlaybackControls`:
      purpose, fields, unicode + ASCII rendering snapshots, role mapping, glyph
      list, test contract — committed in the same commit that adds
      `playback_controls.go`
- [ ] All existing `internal/app/*` and `internal/ui/panes/*` tests pass
- [ ] `make ci` → PASS

## Tasks

Step-by-step: Tasks 4.1, 4.2, 4.4 in `docs/superpowers/plans/2026-04-29-glyph-fallback.md`.

- [ ] Branch: `feat/13-playback-controls-primitive`
- [ ] Write failing `TestPlaybackControls_RenderUnicode_Playing`,
      `TestPlaybackControls_RenderASCII_Playing`, `TestPlaybackControls_RepeatModes`
      → FAIL
- [ ] Create `internal/uikit/playback_controls.go` with `RepeatMode` enum and
      `PlaybackControls` struct + `Render()` method routing through `GlyphFor`
- [ ] Append §3.19 to `docs/TUI-DESIGN-SYSTEM.md`
- [ ] Run tests → PASS
- [ ] Commit: `feat(uikit): add PlaybackControls primitive with ascii fallback`
      (single commit covers code + docs per CLAUDE.md rule 17)
- [ ] Replace the body of `components/controls.go` with the compatibility wrapper
- [ ] Update `controls_test.go` — keep unicode glyph assertions where they were and
      add `TestNewControls_RepeatModeTranslation`
- [ ] Run tests → PASS
- [ ] Commit: `refactor(controls): delegate to uikit.PlaybackControls`
- [ ] Write failing `TestNowPlaying_AsciiTitle` → FAIL
- [ ] Migrate `panes/nowplaying.go:85,87,91` to `uikit.GlyphFor` for playback state
      glyphs and the `─` separator → PASS
- [ ] Commit: `fix(nowplaying): route Title playback glyphs through GlyphFor`
- [ ] `make ci` → PASS
- [ ] Push branch + open PR
