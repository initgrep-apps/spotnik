# Feature 62 — Request Flow Boxed Layout

> **Enhancement:** The Request Flow pane renders as a flat text table with three padded
> columns. DESIGN.md §19 specifies three **bordered sub-boxes** (APP, GATEWAY, SPOTIFY)
> connected by animated arrows — a graphical visualization, not a table. This feature
> restructures `View()` to match the design spec.

## Background

DESIGN.md §19 defines the Request Flow pane as three rounded-corner boxes arranged
horizontally, connected by per-request arrows:

```
╭──────────────╮           ╭──────────────────╮           ╭──────────────╮
│ ▶ /player    │───────→───│ ●●●●●●●●○○ 8/10  │───────→───│  200  45ms   │
│   /queue     │───→ dedup │ ■■■■■□□□□□  3/5  │    ╳      │  200  62ms   │
│   /playlists │─── wait ──│ ⏳ backoff  2.1s │───────→───│  429  12ms   │
│              │           │                  │           │  200  95ms   │
╰──────────────╯           ╰──────────────────╯           ╰──────────────╯
```

**What's currently implemented:** A flat table layout with plain text columns and no
bordered sub-boxes. Gateway metrics (token bucket, semaphore, backoff, dedup) render as a
separate block below all request rows, not inside the center column alongside them.

```
APP              GATEWAY              SPOTIFY
▶ /player        ──→──                200  45ms
  /queue         ──→ dedup            200  62ms

tokens  ●●●●●●●●○○ 8/10
conc    ■■■■■□□□□□ 3/5
```

**What the design expects:** Three independently bordered boxes with gateway metrics
displayed vertically inside the center box. Arrows connect the boxes between them.
Each request row spans all three boxes on the same horizontal line.

**Depends on:** Feature 61 (Fix Request Flow Gateway Viz) — data layer (GatewayDecision,
theme colors, arrow states, staleness) is already complete. This feature only restructures
the rendering layout.

---

## Gap Summary

| # | Gap | Severity | Description |
|---|-----|----------|-------------|
| G1 | No bordered sub-boxes | Critical | Design shows three `╭╮╰╯` boxes; implementation uses flat padded text columns |
| G2 | Gateway metrics below, not in center box | Critical | Token bucket, semaphore, backoff, dedup render as a separate block below request rows instead of inside the GATEWAY box |
| G3 | Single arrow column | High | Only one arrow between APP and SPOTIFY; design shows two arrows per row (APP→GW and GW→SPOTIFY) |
| G4 | No horizontal row alignment across boxes | High | Request endpoint, gateway decision, and response status are not visually connected across bordered boxes |

---

## Design Reference

### Target Layout (from DESIGN.md §19)

The three boxes have these proportions, fitting within the pane's content area:

```
╭─ Request Flow ───────────────────────────────────────────────────────────────────────╮
│                                                                                      │
│   APP                          GATEWAY                          SPOTIFY              │
│  ╭──────────────╮           ╭──────────────────╮           ╭──────────────╮          │
│  │ ▶ /player    │───────→───│ ●●●●●●●●○○ 8/10  │───────→───│  200  45ms   │          │
│  │   /queue     │───→ dedup │ ■■■■■□□□□□  3/5  │    ╳      │  200  62ms   │          │
│  │   /playlists │─── wait ──│ ⏳ backoff  2.1s │───────→───│  429  12ms   │          │
│  │              │           │                  │           │  200  95ms   │          │
│  ╰──────────────╯           ╰──────────────────╯           ╰──────────────╯          │
│                                                                                      │
│  POLLING  tick: 1s  state: active  idle: 0s    STORE  fetching: [playlists, queue]   │
╰──────────────────────────────────────────────────────────────────────────────────────╯
```

### Box Proportions

- **APP box**: ~25% of content width — shows request endpoints
- **Left arrow column**: ~8% — animated arrow showing APP→GW decision
- **GATEWAY box**: ~26% — shows gateway metrics vertically (token bucket, semaphore, backoff, dedup, in-flight keys)
- **Right arrow column**: ~8% — animated arrow showing GW→SPOTIFY outcome
- **SPOTIFY box**: ~20% — shows response status + latency
- Remaining: padding between elements

Minimum content width for three boxes: ~60 columns. At narrower widths, fall back to
the current flat layout (no boxes, just columns) as a graceful degradation.

### Row Alignment

Each request occupies the same vertical line across all three boxes and both arrow columns:

```
Row 1:  │ ▶ /player    │───────→───│ ●●●●●●●●○○ 8/10  │───────→───│  200  45ms   │
Row 2:  │   /queue     │───→ dedup │ ■■■■■□□□□□  3/5  │    ╳      │  200  62ms   │
Row 3:  │   /playlists │─── wait ──│ ⏳ backoff  2.1s │───────→───│  429  12ms   │
```

- Row 1 in APP shows `/player`, row 1 in GATEWAY shows token bucket, row 1 in SPOTIFY shows `200 45ms`
- Rows are filled top-down: request rows first, then remaining gateway metrics fill the GATEWAY box
- If there are more gateway metrics than request rows, the GATEWAY box is taller and APP/SPOTIFY boxes have empty padding rows at the bottom
- If there are more requests than gateway metric lines, gateway metrics stop but request rows continue

### Sub-Box Borders

Use the same rounded-corner style as the existing `InfoBox` component:
- Top: `╭─ LABEL ───╮`  (label is "APP", "GATEWAY", or "SPOTIFY")
- Sides: `│ content │`
- Bottom: `╰───────────╯`
- Border color: `theme.TextSecondary()` (unfocused), `theme.TextPrimary()` when pane is focused
- Label color: `theme.TextSecondary()` bold

### Arrow Columns

Arrows sit **between** the boxes (not inside them). Each request row has two arrows:

1. **Left arrow** (APP → GATEWAY): shows the gateway decision for this request
   - `───────→───` : DecisionAllowed (animated, Success color)
   - `─── wait ──` : DecisionWaited (Warning color)
   - `───→ dedup`  : DecisionDeduped (TextSecondary color)
   - `── ╳ ──`     : DecisionBlocked (Error color)

2. **Right arrow** (GATEWAY → SPOTIFY): shows the HTTP outcome
   - `───────→───` : Success (animated, Success color) for 2xx
   - `── ╳ ──`     : Warning color for 429
   - `───────→───` : Error color for 5xx
   - `── ╳ ──`     : TextMuted for status 0 (blocked, no HTTP call)

The left arrow reuses the existing `renderArrow()` decision logic. The right arrow is new
and reflects the HTTP response outcome.

### GATEWAY Box Content

The GATEWAY box renders metrics vertically, one per line, using the same content that
currently appears in `renderGatewayState()`:

```
│ ●●●●●●●●○○ 8/10  │   ← row 1: token bucket (always shown)
│ ■■■■■□□□□□  3/5   │   ← row 2: semaphore (always shown)
│ ⏳ backoff  2.1s  │   ← row 3: backoff timer (only when throttled)
│ dedup 3 in-flight │   ← row 4: dedup waiters (only when > 0)
│   → GET /player   │   ← row 5+: in-flight keys (up to 3)
│   → GET /queue    │
│   … +1 more       │
```

These lines are aligned with request rows where possible:
- Row 1 of the box shows token bucket, aligned with the first (newest) request
- Row 2 shows semaphore, aligned with the second request
- Remaining metrics fill subsequent rows

### Status Strip

The status strip renders **below** the three boxes, spanning the full pane width. This
part is unchanged from the current implementation:

```
POLLING  tick: 1000ms  state: active    STORE  fetching: [playlists]  stale: albums(45s)
```

---

## Task 1: Create `renderSubBox()` helper

**Problem:** There is no utility to render a small bordered sub-box with a title label
inside the RequestFlowPane. The `InfoBox` component is a full struct with its own state.
We need a lightweight helper for rendering simple bordered rectangles.

**Fix:**

Add a `renderSubBox()` method to `requestflow_pane.go` that takes a title, content lines,
and dimensions, and returns a bordered string with rounded corners:

```go
// renderSubBox renders a small bordered box with a title label.
// lines are the content lines (already styled). The box is sized to exactly
// width columns × (len(lines) + 2) rows (content + top/bottom border).
func (p *RequestFlowPane) renderSubBox(title string, lines []string, width int) string
```

Implementation:
1. Top border: `╭─ TITLE ──...──╮` (fill dashes to width)
2. Content lines: `│ line... │` (pad/truncate each line to `width - 4` inner width, 1 space padding each side)
3. Bottom border: `╰──...──╯` (fill dashes to width)
4. Border color: `p.theme.TextSecondary()` (unfocused default)
5. Title: `p.theme.TextSecondary()` bold

If width < 8, return empty string (too narrow for meaningful box).

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` — add `renderSubBox()` method

**Tests:**
- Unit: `renderSubBox("APP", []string{"line1", "line2"}, 20)` contains `╭─ APP`, `╰`, `│`, rounded corners
- Unit: Content lines are padded to inner width
- Unit: Long content lines are truncated with `…`
- Unit: Width < 8 returns empty string
- Unit: Empty lines slice → box with only borders (height = 2)

**Commit:** `feat(ui): add renderSubBox helper for Request Flow bordered layout`

---

## Task 2: Create `renderRightArrow()` for GATEWAY→SPOTIFY

**Problem:** The current `renderArrow()` shows gateway decisions (APP→GATEWAY). The design
requires a second arrow column (GATEWAY→SPOTIFY) showing the HTTP response outcome.

**Fix:**

Add `renderRightArrow()` to `requestflow_pane.go`:

```go
// renderRightArrow renders the connecting arrow between GATEWAY and SPOTIFY columns.
// The arrow style reflects the HTTP response outcome:
//   - 2xx: animated flowing arrow (Success color)
//   - 429: "── ╳ ──" (Warning color)
//   - 5xx: animated arrow (Error color)
//   - 0:   "── ╳ ──" (TextMuted — blocked, no HTTP call made)
func (p *RequestFlowPane) renderRightArrow(r reqDisplay, colWidth int) string
```

Logic:
- `statusCode == 0` (blocked by gateway): `── ╳ ──` in TextMuted
- `statusCode == 429`: `── ╳ ──` in Warning
- `statusCode >= 500`: animated frames in Error color
- `statusCode >= 200 && < 300`: animated frames in Success color
- Default: animated frames in TextSecondary

Reuse the same `frames` animation array and `p.frameIndex` as existing `renderArrow()`.

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` — add `renderRightArrow()` method

**Tests:**
- Unit: StatusCode 200 → contains animated arrow text, Success color (ANSI check)
- Unit: StatusCode 429 → contains "╳", Warning color
- Unit: StatusCode 500 → animated arrow, Error color
- Unit: StatusCode 0 → contains "╳", TextMuted color
- Unit: Arrow width respects colWidth parameter

**Commit:** `feat(ui): add renderRightArrow for GATEWAY→SPOTIFY connection`

---

## Task 3: Build content generators for each sub-box

**Problem:** The current render methods produce flat column text. For the boxed layout,
each box needs its content as a slice of styled lines (one per row), aligned vertically.

**Fix:**

Add three content generator methods:

### 3a. `buildAppBoxLines(maxRows int) []string`

Returns one styled line per recent request (up to `maxRows`):
- Format: `▶ /endpoint` (active) or `  /endpoint` (dimmed)
- Styling: same as current `renderAppEntry()` but returns just the text, no padding
- Pad with empty lines if fewer requests than `maxRows`

### 3b. `buildGatewayBoxLines(maxRows int) []string`

Returns gateway metric lines (up to `maxRows`):
- Line 1: token bucket bar + count (always)
- Line 2: semaphore bar + count (always)
- Line 3: backoff timer (only when throttled, else skip)
- Line 4: dedup waiters (only when > 0, else skip)
- Lines 5+: in-flight keys (up to 3, with "+N more" truncation)
- Pad with empty lines if fewer metrics than `maxRows`

Content is identical to current `renderGatewayState()` but returned as `[]string` lines
instead of a joined string.

### 3c. `buildSpotifyBoxLines(maxRows int) []string`

Returns one styled line per recent request (up to `maxRows`):
- Format: `200  45ms` with color-coded status
- Styling: same as current `renderSpotifyEntry()` but returns just the text
- Pad with empty lines if fewer requests than `maxRows`

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` — add three `build*BoxLines()` methods

**Tests:**
- Unit: `buildAppBoxLines(4)` with 2 requests → 2 content lines + 2 empty padding lines
- Unit: `buildAppBoxLines(4)` with 6 requests → 4 content lines (capped at maxRows)
- Unit: `buildGatewayBoxLines(4)` with throttle → includes backoff line
- Unit: `buildGatewayBoxLines(4)` without throttle → no backoff line, 2 metric lines + padding
- Unit: `buildSpotifyBoxLines(3)` with 429 → contains "⚠" suffix
- Unit: `buildSpotifyBoxLines(3)` with status 0 → TextMuted styling
- Unit: All methods handle maxRows=0 → empty slice

**Commit:** `feat(ui): add content generators for APP/GATEWAY/SPOTIFY sub-boxes`

---

## Task 4: Restructure `View()` to render boxed layout

**Problem:** `View()` currently calls `renderColumnHeaders()`, `renderRequestRows()`,
`renderGatewayState()`, and `renderStatusStrip()` in a flat vertical stack. The design
requires three bordered boxes arranged horizontally with arrow columns between them.

**Fix:**

Replace the core of `View()` with boxed layout composition:

```go
func (p *RequestFlowPane) View() string {
    if p.width <= 0 || p.height <= 0 {
        return ""
    }

    // Calculate box dimensions.
    contentWidth := p.width
    statusStripHeight := 1 // bottom status strip always 1 row
    boxAreaHeight := p.height - statusStripHeight - 1 // -1 for spacing

    // Minimum width check — fall back to flat layout if too narrow.
    if contentWidth < 60 {
        return p.viewFlat()
    }

    // Column widths (proportional).
    appBoxW := contentWidth * 25 / 100
    arrowW := contentWidth * 8 / 100
    gwBoxW := contentWidth * 26 / 100
    spotifyBoxW := contentWidth * 20 / 100
    // Remaining space is inter-box padding (absorbed by lipgloss.JoinHorizontal).

    // Inner row count = box height - 2 (top/bottom border).
    innerRows := boxAreaHeight - 2
    if innerRows < 1 {
        innerRows = 1
    }

    // Build content lines for each box.
    appLines := p.buildAppBoxLines(innerRows)
    gwLines := p.buildGatewayBoxLines(innerRows)
    spotifyLines := p.buildSpotifyBoxLines(innerRows)

    // Build arrow columns (one line per request row).
    leftArrows := p.buildLeftArrowLines(innerRows, arrowW)
    rightArrows := p.buildRightArrowLines(innerRows, arrowW)

    // Render bordered sub-boxes.
    appBox := p.renderSubBox("APP", appLines, appBoxW)
    gwBox := p.renderSubBox("GATEWAY", gwLines, gwBoxW)
    spotifyBox := p.renderSubBox("SPOTIFY", spotifyLines, spotifyBoxW)

    // Compose: APP | leftArrows | GATEWAY | rightArrows | SPOTIFY
    leftArrowBlock := strings.Join(leftArrows, "\n")
    rightArrowBlock := strings.Join(rightArrows, "\n")

    // Vertically pad arrow blocks to match box height (add top/bottom blank lines for borders).
    leftArrowBlock = "\n" + leftArrowBlock + "\n"
    rightArrowBlock = "\n" + rightArrowBlock + "\n"

    composite := lipgloss.JoinHorizontal(lipgloss.Top,
        appBox, leftArrowBlock, gwBox, rightArrowBlock, spotifyBox)

    // Append status strip below.
    statusStrip := p.renderStatusStrip()

    return composite + "\n" + statusStrip
}
```

### Arrow line builders

Add `buildLeftArrowLines()` and `buildRightArrowLines()`:

```go
// buildLeftArrowLines builds arrow strings for APP→GATEWAY (one per request row).
// Rows beyond request count are empty (padding).
func (p *RequestFlowPane) buildLeftArrowLines(maxRows, colWidth int) []string

// buildRightArrowLines builds arrow strings for GATEWAY→SPOTIFY (one per request row).
// Rows beyond request count are empty (padding).
func (p *RequestFlowPane) buildRightArrowLines(maxRows, colWidth int) []string
```

### Flat layout fallback

Rename the current `View()` body to `viewFlat()` — used when pane width < 60 columns:

```go
// viewFlat renders the original flat table layout as a fallback for narrow widths.
func (p *RequestFlowPane) viewFlat() string
```

This preserves all existing rendering logic as-is for narrow terminals.

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` — restructure `View()`, add `viewFlat()`,
  add `buildLeftArrowLines()`, `buildRightArrowLines()`

**Tests:**
- Unit: View() at width=80 → output contains `╭─ APP`, `╭─ GATEWAY`, `╭─ SPOTIFY` (three bordered boxes)
- Unit: View() at width=80 → output contains `╰` characters (bottom borders of boxes)
- Unit: View() at width=80 → output contains "tokens" or "●" inside GATEWAY section
- Unit: View() at width=40 → falls back to flat layout (no bordered boxes)
- Unit: View() at width=80, with 3 requests → 3 rows of arrows between boxes
- Unit: View() → status strip appears below the boxes
- Unit: View() at width=80, height=5 → renders with minimal inner rows (1)
- Unit: View() at width=80, height=0 → returns empty string

**Commit:** `feat(ui): restructure View() to three bordered sub-boxes with arrows`

---

## Task 5: Align arrow animation with sub-box rows

**Problem:** Arrow columns need vertical alignment with the content rows inside the
bordered boxes. The top and bottom borders of the boxes add 1 row each, so arrow lines
must be offset by 1 row from the box top to align with inner content.

**Fix:**

1. In `buildLeftArrowLines()` and `buildRightArrowLines()`, generate exactly `innerRows`
   lines (matching the number of content lines inside each box)

2. When composing with `lipgloss.JoinHorizontal()`, prepend a blank line and append a
   blank line to each arrow block to account for the box's top/bottom border:

   ```
   Box:     ╭─ APP ──╮      Arrow block:     (blank)
            │ /player │                       ───→──
            │ /queue  │                       ── ╳ ─
            ╰─────────╯                       (blank)
   ```

3. Ensure each arrow line is padded to `arrowW` using `padRightVisible()` so
   `JoinHorizontal` aligns correctly.

4. Rows beyond the request count in arrow blocks are blank (space-padded to `arrowW`).

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` — adjust arrow block padding in `View()`

**Tests:**
- Unit: View() output line count for boxes matches: content lines between `╭` and `╰` rows
  align with non-blank arrow lines
- Unit: Arrow block has same number of lines as box height (innerRows + 2 for borders)
- Unit: First and last arrow lines are blank (corresponding to box border rows)

**Commit:** `feat(ui): align arrow animation rows with sub-box content`

---

## Task 6: Preserve existing rendering methods

**Problem:** The refactored `View()` uses new content generators, but the existing
`renderAppEntry()`, `renderArrow()`, `renderSpotifyEntry()`, `renderGatewayState()`,
and `renderColumnHeaders()` should be preserved for the flat fallback and for reuse
by the new generators.

**Fix:**

1. `viewFlat()` calls the original methods: `renderColumnHeaders()`, `renderRequestRows()`,
   `renderGatewayState()`, `renderStatusStrip()` — no changes to these methods

2. The new `build*BoxLines()` methods can internally call `renderAppEntry()`,
   `renderArrow()`, `renderSpotifyEntry()` for consistent styling, then strip padding
   to fit inside box inner width

3. `renderGatewayState()` is refactored to optionally return `[]string` lines:
   - Add `gatewayStateLines() []string` that returns individual lines
   - `renderGatewayState()` calls `gatewayStateLines()` and joins with `\n` (backward compat)
   - `buildGatewayBoxLines()` calls `gatewayStateLines()` directly

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` — add `gatewayStateLines()`, update
  `renderGatewayState()` to use it, `buildGatewayBoxLines()` uses it

**Tests:**
- Unit: `renderGatewayState()` output is unchanged (backward compat for flat layout)
- Unit: `gatewayStateLines()` returns correct number of lines for various states
- Unit: `viewFlat()` output matches previous `View()` output for same state (regression check)

**Commit:** `refactor(ui): extract gatewayStateLines for reuse in boxed and flat layouts`

---

## Task 7: Update existing tests

**Problem:** Existing tests assert on the flat layout output. With boxed layout as default
(width >= 60), these tests need updating to check for bordered box output instead.

**Fix:**

1. Tests with `SetSize(80, 20)` now expect bordered box output:
   - Check for `╭─ APP` or `╭─ GATEWAY` or `╭─ SPOTIFY` in View() output
   - Check for rounded corners `╭`, `╮`, `╰`, `╯`
   - Check that gateway metrics appear inside the GATEWAY box section

2. Add new tests for flat fallback:
   - `SetSize(40, 20)` → View() does NOT contain `╭─ APP` (uses flat layout)
   - `SetSize(40, 20)` → View() contains column headers as plain text

3. Existing tests for arrow states, color coding, staleness, etc. remain valid —
   the content is the same, just the container changes. Update string assertions that
   checked for specific column positions.

4. Test helper: add `viewContainsBox(t, output, title)` that checks for the presence of
   `╭─ TITLE` in the output.

**Files:**
- Modify: `internal/ui/panes/requestflow_pane_test.go` — update assertions for boxed layout,
  add flat fallback tests

**Tests:**
- Update all View() tests that use width >= 60 to expect bordered boxes
- Add: `TestRequestFlowPane_View_FlatFallback` — width 40 uses flat layout
- Add: `TestRequestFlowPane_View_BoxedLayout` — width 80 has three boxes
- Add: `TestRequestFlowPane_View_BoxedLayout_GatewayInCenter` — GATEWAY metrics inside center box
- Add: `TestRequestFlowPane_View_BoxedLayout_DualArrows` — both left and right arrow columns present
- Preserve: all decision arrow state tests (just check content, not column position)
- Preserve: all color coding tests
- Preserve: all staleness tests
- Preserve: all animation frame tests

**Commit:** `test(ui): update Request Flow tests for boxed layout with flat fallback`

---

## Task 8: Update documentation

**Fix:**

1. Update `docs/ARCHITECTURE.md` — in the Page B / Request Flow section, note:
   - Three bordered sub-boxes layout (APP, GATEWAY, SPOTIFY)
   - Dual arrow columns
   - Flat fallback for narrow terminals (< 60 columns)
   - `renderSubBox()` helper pattern

2. Update `docs/features/00-overview.md` — add Feature 62 row

**Files:**
- Modify: `docs/ARCHITECTURE.md` — update Request Flow rendering description
- Modify: `docs/features/00-overview.md` — add Feature 62 entry

**Commit:** `docs: add Feature 62 boxed layout to architecture docs`

---

## Acceptance Criteria

- [ ] Three bordered sub-boxes (APP, GATEWAY, SPOTIFY) render with rounded corners
- [ ] Gateway metrics (token bucket, semaphore, backoff, dedup) appear inside the center GATEWAY box, not below
- [ ] Two arrow columns connect the boxes: APP→GW (gateway decision) and GW→SPOTIFY (HTTP outcome)
- [ ] Request rows align horizontally across all three boxes and both arrow columns
- [ ] Arrow states match Feature 61: flowing, wait, dedup, blocked (left arrows) and 2xx, 429, 5xx, blocked (right arrows)
- [ ] Theme colors are preserved: all existing color coding from Feature 61 works in the boxed layout
- [ ] Status strip renders below the three boxes spanning full width
- [ ] Flat fallback activates for pane width < 60 columns (identical to current behavior)
- [ ] Staleness display works unchanged in the status strip
- [ ] InFlightKeys render inside the GATEWAY box
- [ ] Animation frames advance correctly in both arrow columns
- [ ] All existing tests pass (updated for new layout) + new tests added
- [ ] `make ci` passes

---

## Verification

```bash
# Sub-box borders render
grep 'renderSubBox' internal/ui/panes/requestflow_pane.go

# Three boxes in View
grep 'GATEWAY.*SPOTIFY\|renderSubBox.*APP\|renderSubBox.*GATEWAY\|renderSubBox.*SPOTIFY' internal/ui/panes/requestflow_pane.go

# Right arrow exists
grep 'renderRightArrow\|buildRightArrowLines' internal/ui/panes/requestflow_pane.go

# Flat fallback preserved
grep 'viewFlat' internal/ui/panes/requestflow_pane.go

# Gateway state lines extraction
grep 'gatewayStateLines' internal/ui/panes/requestflow_pane.go

# Boxed layout tests
grep 'BoxedLayout\|FlatFallback' internal/ui/panes/requestflow_pane_test.go

# Full CI passes
make ci
```

---

## Implementation Notes for Agents

### Key files to read first
- `internal/ui/panes/requestflow_pane.go` — the file being modified (534 lines)
- `internal/ui/panes/requestflow_pane_test.go` — existing tests (740 lines)
- `internal/ui/components/infobox.go` — reference for how inner bordered boxes are drawn
- `internal/ui/layout/border.go` — the RenderPaneBorder pattern (outer pane border)
- `internal/ui/layout/truncate.go` — TruncateOrPad, padRight utilities

### Patterns to follow
- Use `lipgloss.JoinHorizontal(lipgloss.Top, ...)` for horizontal composition (same as NowPlaying pane)
- Use `lipgloss.Width()` for measuring styled string widths (ANSI-safe)
- Use `padRightVisible()` (already in the file) for padding styled strings
- Use `truncateStr()` (already in the file) for truncating content to fit box inner width
- Box border drawing: follow the `InfoBox` pattern — manual `╭╮│╰╯` construction with `lipgloss.NewStyle().Foreground()` for border color

### Do NOT
- Import any new dependencies — use lipgloss and stdlib only
- Change the outer pane border (that's handled by `RenderPaneBorder` in `layout/`)
- Modify `renderGatewayState()` signature — only add `gatewayStateLines()` alongside it
- Remove any existing methods — `viewFlat()` keeps them alive for narrow terminals
- Change data structures (`reqDisplay`, `RequestFlowPane`, etc.) — layout-only change

### Test patterns
- Use `newTestRequestFlowPane()` helper (already exists)
- Inject requests via `RequestCompletedMsg` with `CompletedAt: time.Now()`
- Check View() output with `strings.Contains()` and `assert.Contains()`
- Use `lipgloss.Width()` when checking column alignment

---

*Depends on: Feature 61*
*Blocks: Nothing*
