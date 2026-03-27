---
name: project_spotnik_feature58_complete
description: Feature 58 (NowPlaying Split Layout): InfoBox+Visualizer split, compact removal, splitLines shared helper discovery
type: project
---

## Feature 58 — NowPlaying Split Layout (btop-inspired)

**What was built:**
- Rewrote `NowPlayingPane.View()` with a horizontal split: InfoBox (left, ~1/4 width, min 28 chars) + Visualizer (right, ~3/4 width), gradient seek bar at bottom
- Removed `compact bool` field, `renderCompact()`, and `interpolateHexCompact`/`parseHexParts`/`lerpByte` helpers (resolves TODO(feature-53))
- Updated `Title()` to use `height < 8` instead of compact flag
- Updated `SetSize()` to compute split dimensions: `infoWidth = paneMax(contentWidth/4, 28)`, `vizWidth = contentWidth - infoWidth - 1`
- Added `infoBox *components.InfoBox` field initialized in constructor

**Key files:**
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/panes/nowplaying.go` — full implementation (339 lines, down from 450)
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/panes/nowplaying_test.go` — 9 compact tests removed, 8 new split tests added

**Patterns established:**
- `lipgloss.JoinHorizontal(lipgloss.Top, infoView, " ", vizView)` for the side-by-side body
- `lipgloss.JoinVertical(lipgloss.Left, body, seekBar)` to put seek bar below the split
- InfoBox receives pre-rendered lines (styled strings) via `infoLines []string` — InfoBox handles truncation/centering internally

**Gotchas:**
- `splitLines` helper was defined in `nowplaying_test.go` but used by `queue_test.go` too (same package). When removing it from nowplaying_test.go it broke queue_test.go. Had to keep `splitLines` in nowplaying_test.go with a doc comment marking it as shared.
- `filterNonEmpty` was only used by compact tests — removed entirely with no other callers.
- `vizWidth` can go negative at very small sizes (contentWidth=10, infoWidth=28 → vizWidth=-19). This is safe because `Visualizer.SetSize()` clamps to width=1.
- `gofmt` caught trailing space in `vizWidth := contentWidth - infoWidth - 1  // -1` (double space before comment) — run gofmt immediately after editing.

**Testing notes:**
- 8 new split layout tests all follow the same pattern: create pane, SetSize(80, 24), call View(), assert string contains expected element
- `TestNowPlayingPane_SplitLayout_ContainsBraille` requires `pane.visualizer.SetPlaying(true)` to see braille characters
- `TestNowPlayingPane_Title_ShowsTrackInfoWhenSmall` sets height to 6 (< 8) to trigger the compact title path
- panes coverage: 89.9%, overall: 86.1%
