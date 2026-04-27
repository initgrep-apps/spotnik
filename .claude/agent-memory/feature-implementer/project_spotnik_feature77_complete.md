---
name: project_spotnik_feature77_complete
description: Story 77 (Overlay Backgrounds, Border Corner, Network Log Colors, Status Bar Pills): patterns established, collision gotcha, ANSI test strategies
type: project
---

## Story 77 — Overlay/Border/Theme/StatusBar Fixes

**Built:**
- Removed explicit `Background()` from non-cursor rows in ThemeOverlay.renderRow() + DeviceOverlay.renderDevice()
- Made `rightSuffix` conditional in RenderPaneBorder (empty when no actions, " " when actions exist)
- Updated all 11 TOML theme files w/ vibrant, distinct network_log border colors
- Added `Background(StatusBarBg())` to `keyStyle` in renderStatusBar(), moved space inside bgStyle.Render

**Key files:**
- `internal/ui/panes/themes.go` — renderRow() split cursor/non-cursor branches
- `internal/ui/panes/devices.go` — renderDevice() split cursor/non-cursor branches
- `internal/ui/layout/border.go` — rightSuffix conditional (line ~121)
- `internal/app/render.go` — keyStyle now has Background, space moved into bgStyle.Render(" "+h.Label)
- `internal/ui/theme/themes/*.toml` — all 11 files updated for network_log

**Gotchas:**

1. **TOML network_log color collisions**: First round caused 8/11 themes w/ network_log matching existing pane border (albums, recently_played, playlists, etc.). Check ALL 9 existing border values for collision, not just "recently_played". Run collision check script before committing TOML changes.

2. **Test strategy for "no background"**: Assert rendered string NOT contains `"48;2;"`. TDD-correct: test must fail w/ old implementation — just checking `"48;2;"` absence fails to distinguish "correct" from "no color profile active". Always set `lipgloss.SetColorProfile(termenv.TrueColor)` in test.

3. **Status bar key style test**: Testing renderStatusBar renders keys WITH Background tricky because bgStyle (always present) also has background. Working approach: render same key char w/ and w/o Background, check rendered output contains WITH-background version + does NOT contain NO-background version.

4. **spec vs reality for color saturation**: Spec said `#83a598` for gruvbox but saturation 34 (< 50 threshold). Deviated to `#458588` (gruvbox canonical blue/aqua, sat=67). Verify spec colors against test's own threshold.

5. **rightSuffix fix + width calc**: `fixedWidth` uses `lipgloss.Width(rightSuffix)` so changing rightSuffix to "" auto-adjusts dash fill. Existing `TestRenderPaneBorder_WidthMatchesRequested` still passes — width math correct.

**Testing notes:**
- `TestThemeOverlay_NonCursorRow_NoExplicitBackground` — assert NOT contains `"48;2;"` w/ TrueColor forced
- `TestDeviceOverlay_NonCursorRow_NoExplicitBackground` — same pattern
- `TestAllThemes_NetworkLogBorderIsVibrant` — parse hex, max-min channel spread > 50
- `TestRenderStatusBar_KeyStyleHasConsistentBackground` — compare rendered key w/wo bg
- `extractBgANSISeq` helper added to render_test.go for extracting "48;2;R;G;B" patterns

**Coverage:** 86.5% overall (above 80% threshold)