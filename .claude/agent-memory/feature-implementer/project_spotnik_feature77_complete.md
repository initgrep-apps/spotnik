---
name: project_spotnik_feature77_complete
description: Story 77 (Overlay Backgrounds, Border Corner, Network Log Colors, Status Bar Pills): patterns established, collision gotcha, ANSI test strategies
type: project
---

## Story 77 — Overlay/Border/Theme/StatusBar Fixes

**What was built:**
- Removed explicit `Background()` from non-cursor rows in ThemeOverlay.renderRow() and DeviceOverlay.renderDevice()
- Made `rightSuffix` conditional in RenderPaneBorder (empty when no actions, " " when actions exist)
- Updated all 11 TOML theme files with vibrant, distinct network_log border colors
- Added `Background(StatusBarBg())` to `keyStyle` in renderStatusBar() and moved space inside bgStyle.Render

**Key files:**
- `internal/ui/panes/themes.go` — renderRow() split into cursor/non-cursor branches
- `internal/ui/panes/devices.go` — renderDevice() split into cursor/non-cursor branches
- `internal/ui/layout/border.go` — rightSuffix conditional (line ~121)
- `internal/app/render.go` — keyStyle now has Background, space moved into bgStyle.Render(" "+h.Label)
- `internal/ui/theme/themes/*.toml` — all 11 files updated for network_log

**Gotchas:**

1. **TOML network_log color collisions**: The first round of color choices caused 8/11 themes to have network_log matching an existing pane border (albums, recently_played, playlists, etc.). Always check ALL 9 existing border values for collision, not just "recently_played". Run the collision check script before committing TOML changes.

2. **Test strategy for "no background"**: To test that a row has no explicit background, assert that the rendered string does NOT contain `"48;2;"`. To test TDD-correctly, the test must fail with the old implementation — just checking `"48;2;"` absence fails to distinguish "correct" from "no color profile active". Always set `lipgloss.SetColorProfile(termenv.TrueColor)` in the test.

3. **Status bar key style test**: Testing that renderStatusBar renders keys WITH Background is tricky because bgStyle (which is always present) also has the background. The working approach: render the same key char with and without Background, check that the rendered output contains the WITH-background version and does NOT contain the NO-background version.

4. **spec vs reality for color saturation**: Spec specified `#83a598` for gruvbox but that has saturation 34 (< 50 threshold). Had to deviate from spec to pick `#458588` (gruvbox canonical blue/aqua with sat=67). Always verify spec-specified colors against the test's own threshold.

5. **rightSuffix fix and width calculations**: The `fixedWidth` computation uses `lipgloss.Width(rightSuffix)` so changing rightSuffix to "" automatically adjusts the dash fill. The existing `TestRenderPaneBorder_WidthMatchesRequested` continues to pass because width math is correct.

**Testing notes:**
- `TestThemeOverlay_NonCursorRow_NoExplicitBackground` — assert NOT contains `"48;2;"` with TrueColor forced
- `TestDeviceOverlay_NonCursorRow_NoExplicitBackground` — same pattern
- `TestAllThemes_NetworkLogBorderIsVibrant` — parse hex, compute max-min channel spread > 50
- `TestRenderStatusBar_KeyStyleHasConsistentBackground` — compare rendered key with/without bg
- `extractBgANSISeq` helper added to render_test.go for extracting "48;2;R;G;B" patterns

**Coverage:** 86.5% overall (above 80% threshold)
