---
name: project_spotnik_feature58_complete
description: Feature 58 (NowPlaying Split Layout): InfoBox+Visualizer split, compact removal, splitLines shared helper discovery
type: project
---

## Feature 58 — NowPlaying Split Layout (btop-inspired)

**Built:**
- Rewrote `NowPlayingPane.View()` w/ horizontal split: InfoBox (left, ~1/4 width, min 28 chars) + Visualizer (right, ~3/4 width), gradient seek bar bottom
- Removed `compact bool` field, `renderCompact()`, `interpolateHexCompact`/`parseHexParts`/`lerpByte` helpers (resolves TODO(feature-53))
- `Title()` uses `height < 8` instead of compact flag
- `SetSize()` computes split: `infoWidth = paneMax(contentWidth/4, 28)`, `vizWidth = contentWidth - infoWidth - 1`
- Added `infoBox *components.InfoBox` field, init in constructor

**Files:**
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/panes/nowplaying.go` — impl (339 lines, was 450)
- `/Users/irshadsheikh/dev/github/apps/spotnik/internal/ui/panes/nowplaying_test.go` — 9 compact tests removed, 8 split tests added

**Patterns:**
- `lipgloss.JoinHorizontal(lipgloss.Top, infoView, " ", vizView)` for side-by-side body
- `lipgloss.JoinVertical(lipgloss.Left, body, seekBar)` puts seek bar below split
- InfoBox gets pre-rendered styled lines via `infoLines []string` — handles truncation/centering internally

**Gotchas:**
- `splitLines` defined in `nowplaying_test.go` but used by `queue_test.go` too (same package). Removing broke queue_test.go. Kept in nowplaying_test.go w/ doc comment marking shared.
- `filterNonEmpty` only used by compact tests — removed, no other callers.
- `vizWidth` goes negative at tiny sizes (contentWidth=10, infoWidth=28 → vizWidth=-19). Safe: `Visualizer.SetSize()` clamps to width=1.
- `gofmt` caught trailing space in `vizWidth := contentWidth - infoWidth - 1  // -1` (double space before comment) — run gofmt after editing.

**Testing:**
- 8 split tests follow pattern: create pane, SetSize(80, 24), call View(), assert string contains expected element
- `TestNowPlayingPane_SplitLayout_ContainsBraille` needs `pane.visualizer.SetPlaying(true)` for braille chars
- `TestNowPlayingPane_Title_ShowsTrackInfoWhenSmall` sets height=6 (< 8) to trigger compact title path
- panes coverage: 89.9%, overall: 86.1%