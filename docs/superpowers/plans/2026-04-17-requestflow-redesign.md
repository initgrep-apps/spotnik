# RequestFlow Pane Redesign — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the RequestFlow pane into a four-zone layout — GATEWAY health banner, APP/GATEWAY LOG/SPOTIFY three-column area, and AUTO-TRAFFIC strip — with distinct colors per zone and human-readable labels throughout.

**Architecture:** Two full-width single-row boxes (GATEWAY banner at top, AUTO-TRAFFIC strip at bottom) bracket a three-column area. Each zone uses a different border color from existing theme tokens (`ColumnPrimary()` for app-side, `PaneBorderRequestFlow()` for gateway, `Success()` for Spotify). The GATEWAY LOG column becomes a pure event stream; metrics move to the banner. The status strip is removed entirely.

**Tech Stack:** Go 1.22, Bubble Tea v0.27+, Lip Gloss, `internal/ui/panes`, `internal/domain`, `internal/state`

---

## File Map

| File | What changes |
|---|---|
| `internal/ui/panes/requestflow_pane.go` | Add `stripAPIPrefix()`, `humanInterval()`, `humanAge()`, `renderGatewayBanner()`, `renderAutoTrafficStrip()`; rewrite `viewBoxed()`, `formatDecisionLabel()`, `renderGatewayState()`; update `viewFlat()`; delete `renderStatusStrip()`, `renderStoreStatus()`, `renderStalenessStatus()` |
| `internal/ui/panes/requestflow_boxed.go` | Add `borderColor lipgloss.Color` param to `renderSubBox()`; rewrite `buildGatewayBoxLines()` (pure log); rewrite `buildSpotifyBoxLines()` (new format + omit blocked/dedup); delete `gatewayStateLines()`, `buildLeftArrowLines()`, `buildRightArrowLines()`, `renderRightArrow()` |
| `internal/ui/panes/requestflow_replay_test.go` | Add tests for `stripAPIPrefix()`, `humanInterval()`, `humanAge()`, `formatDecisionLabel()` |
| `internal/ui/panes/requestflow_boxed_test.go` | Add color param to all `renderSubBox` call sites; add `renderGatewayBanner`, `buildGatewayBoxLines`, `buildSpotifyBoxLines` tests; delete `renderRightArrow` tests |
| `internal/ui/panes/requestflow_pane_test.go` | Delete status strip tests; add `renderAutoTrafficStrip` tests |

No changes to `requestflow_replay.go`, `domain/`, `state/`, or any theme file.

---

## Task 1: Branch and CI baseline

**Files:** none

- [ ] **Create feature branch**

```bash
git checkout main && git pull origin main
git checkout -b feat/requestflow-redesign
```

- [ ] **Verify CI is green before touching any code**

```bash
make ci
```

Expected: PASS. If it fails, stop and fix before continuing.

---

## Task 2: `stripAPIPrefix()` helper + `formatDecisionLabel()` update

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` (near `truncateStr` at line 640)
- Modify: `internal/ui/panes/requestflow_replay_test.go`

The `formatDecisionLabel` function currently includes the full `/v1/me/player` path in every label. It also formats the `EventRequestEntered` label as `→ GET /v1/me/player entered [◷]`, which is verbose. This task strips the `/v1/me` prefix and rewrites the labels to be compact.

- [ ] **Write failing tests in `requestflow_replay_test.go`**

```go
func TestStripAPIPrefix(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"/v1/me/player", "/player"},
		{"/v1/me/player/volume", "/player/volume"},
		{"/v1/me/playlists", "/playlists"},
		{"/other/path", "/other/path"},   // no prefix — unchanged
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, stripAPIPrefix(tt.in))
		})
	}
}

func TestFormatDecisionLabel_EventRequestEntered_Background(t *testing.T) {
	e := domain.GatewayEvent{
		Kind:     domain.EventRequestEntered,
		Method:   "GET",
		Path:     "/v1/me/player",
		Priority: domain.PriorityBackground,
	}
	assert.Equal(t, "◷ GET /player", formatDecisionLabel(e))
}

func TestFormatDecisionLabel_EventRequestEntered_Interactive(t *testing.T) {
	e := domain.GatewayEvent{
		Kind:     domain.EventRequestEntered,
		Method:   "PUT",
		Path:     "/v1/me/player/volume",
		Priority: domain.PriorityInteractive,
	}
	assert.Equal(t, "⚡ PUT /player/volume", formatDecisionLabel(e))
}

func TestFormatDecisionLabel_EventRequestAllowed(t *testing.T) {
	e := domain.GatewayEvent{Kind: domain.EventRequestAllowed, Method: "GET", Path: "/v1/me/player"}
	assert.Equal(t, "✓ GET /player  allowed", formatDecisionLabel(e))
}

func TestFormatDecisionLabel_EventRequestBlocked(t *testing.T) {
	e := domain.GatewayEvent{Kind: domain.EventRequestBlocked, Method: "PUT", Path: "/v1/me/player/volume"}
	assert.Equal(t, "✗ PUT /player/volume  blocked", formatDecisionLabel(e))
}

func TestFormatDecisionLabel_EventDedupJoined(t *testing.T) {
	e := domain.GatewayEvent{Kind: domain.EventDedupJoined, Method: "GET", Path: "/v1/me/player"}
	assert.Equal(t, "⧖ GET /player  dedup joined", formatDecisionLabel(e))
}

func TestFormatDecisionLabel_EventHttpCompleted(t *testing.T) {
	e := domain.GatewayEvent{Kind: domain.EventHttpCompleted, StatusCode: 200, DurationMs: 43}
	assert.Equal(t, "✓ 200  43ms", formatDecisionLabel(e))
}

func TestFormatDecisionLabel_EventBackoffStarted(t *testing.T) {
	e := domain.GatewayEvent{
		Kind:     domain.EventBackoffStarted,
		Snapshot: domain.GatewayStateSnapshot{BackoffRemaining: 10.0},
	}
	assert.Equal(t, "⏳ backoff started  10s", formatDecisionLabel(e))
}
```

- [ ] **Run tests — verify they fail**

```bash
go test ./internal/ui/panes/... -run "TestStripAPIPrefix|TestFormatDecisionLabel" -v
```

Expected: FAIL — `stripAPIPrefix undefined` or wrong label formats.

- [ ] **Add `stripAPIPrefix()` to `requestflow_pane.go` near `truncateStr`**

```go
// stripAPIPrefix removes the "/v1/me" prefix common to all Spotify API paths.
// Keeps labels compact without losing meaning — all Spotify paths start with /v1/me.
func stripAPIPrefix(path string) string {
	return strings.TrimPrefix(path, "/v1/me")
}
```

- [ ] **Rewrite `formatDecisionLabel()` in `requestflow_pane.go`**

Replace the existing function (line ~477) with:

```go
// formatDecisionLabel builds the display string for a decision log entry.
func formatDecisionLabel(e domain.GatewayEvent) string {
	path := stripAPIPrefix(e.Path)
	switch e.Kind {
	case domain.EventRequestEntered:
		tag := "◷"
		if e.Priority == domain.PriorityInteractive {
			tag = "⚡"
		}
		return fmt.Sprintf("%s %s %s", tag, e.Method, path)
	case domain.EventTokenConsumed:
		return fmt.Sprintf("⊖ token consumed → %d", e.Snapshot.TokensAvailable)
	case domain.EventTokenRefilled:
		return fmt.Sprintf("↻ tokens refilled → %d", e.Snapshot.TokensAvailable)
	case domain.EventSemaphoreAcquired:
		return fmt.Sprintf("⊞ semaphore acquired (%d/%d)",
			e.Snapshot.ConcurrentActive, e.Snapshot.ConcurrentMax)
	case domain.EventSemaphoreReleased:
		return fmt.Sprintf("⊟ semaphore released (%d/%d)",
			e.Snapshot.ConcurrentActive, e.Snapshot.ConcurrentMax)
	case domain.EventBackoffStarted:
		return fmt.Sprintf("⏳ backoff started  %ds", int(e.Snapshot.BackoffRemaining))
	case domain.EventBackoffExpired:
		return "✓ backoff cleared"
	case domain.EventRequestAllowed:
		return fmt.Sprintf("✓ %s %s  allowed", e.Method, path)
	case domain.EventRequestBlocked:
		return fmt.Sprintf("✗ %s %s  blocked", e.Method, path)
	case domain.EventDedupJoined:
		return fmt.Sprintf("⧖ %s %s  dedup joined", e.Method, path)
	case domain.EventDedupResolved:
		return fmt.Sprintf("✓ dedup resolved  %d", e.StatusCode)
	case domain.EventHttpCompleted:
		return fmt.Sprintf("✓ %d  %dms", e.StatusCode, e.DurationMs)
	default:
		return "? unknown event"
	}
}
```

- [ ] **Run tests — verify they pass**

```bash
go test ./internal/ui/panes/... -run "TestStripAPIPrefix|TestFormatDecisionLabel" -v
```

Expected: PASS.

- [ ] **Verify build**

```bash
go build ./internal/ui/panes/...
```

- [ ] **Commit**

```bash
git add internal/ui/panes/requestflow_pane.go internal/ui/panes/requestflow_replay_test.go
git commit -m "refactor(requestflow): strip /v1/me prefix and rewrite decision labels"
```

---

## Task 3: `humanInterval()` and `humanAge()` helpers

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` (add after `stripAPIPrefix`)
- Modify: `internal/ui/panes/requestflow_replay_test.go`

- [ ] **Write failing tests in `requestflow_replay_test.go`**

```go
func TestHumanInterval(t *testing.T) {
	tests := []struct {
		ms   int
		want string
	}{
		{1000, "1s"},
		{3000, "3s"},
		{500, "500ms"},
		{999, "999ms"},
		{0, "?"},
		{-1, "?"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%dms", tt.ms), func(t *testing.T) {
			assert.Equal(t, tt.want, humanInterval(tt.ms))
		})
	}
}

func TestHumanAge_JustNow(t *testing.T) {
	fa := time.Now().Add(-30 * time.Second)
	assert.Equal(t, "just now", humanAge(fa))
}

func TestHumanAge_Minutes(t *testing.T) {
	fa := time.Now().Add(-21 * time.Minute)
	assert.Equal(t, "21m ago", humanAge(fa))
}

func TestHumanAge_Hours(t *testing.T) {
	fa := time.Now().Add(-2 * time.Hour)
	assert.Equal(t, "2h ago", humanAge(fa))
}

func TestHumanAge_HoursAndMinutes(t *testing.T) {
	fa := time.Now().Add(-(1*time.Hour + 2*time.Minute))
	assert.Equal(t, "1h 2m ago", humanAge(fa))
}
```

- [ ] **Run tests — verify they fail**

```bash
go test ./internal/ui/panes/... -run "TestHumanInterval|TestHumanAge" -v
```

Expected: FAIL — `humanInterval undefined`, `humanAge undefined`.

- [ ] **Add helpers to `requestflow_pane.go` after `stripAPIPrefix`**

```go
// humanInterval converts a polling interval in milliseconds to a human-readable string.
// Intervals >= 1000ms are shown in whole seconds. Below 1000ms are shown as-is.
func humanInterval(ms int) string {
	if ms <= 0 {
		return "?"
	}
	if ms >= 1000 {
		return fmt.Sprintf("%ds", ms/1000)
	}
	return fmt.Sprintf("%dms", ms)
}

// humanAge converts a past time.Time to a human-readable age string.
func humanAge(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if m == 0 {
		return fmt.Sprintf("%dh ago", h)
	}
	return fmt.Sprintf("%dh %dm ago", h, m)
}
```

- [ ] **Run tests — verify they pass**

```bash
go test ./internal/ui/panes/... -run "TestHumanInterval|TestHumanAge" -v
```

Expected: PASS.

- [ ] **Commit**

```bash
git add internal/ui/panes/requestflow_pane.go internal/ui/panes/requestflow_replay_test.go
git commit -m "feat(requestflow): add humanInterval and humanAge helpers"
```

---

## Task 4: `renderSubBox()` — add `borderColor` param

**Files:**
- Modify: `internal/ui/panes/requestflow_boxed.go` (signature change)
- Modify: `internal/ui/panes/requestflow_pane.go` (`viewBoxed()` call sites)
- Modify: `internal/ui/panes/requestflow_boxed_test.go` (all call sites)

- [ ] **Update all `renderSubBox` calls in `requestflow_boxed_test.go`**

Every call `p.renderSubBox("X", lines, width)` becomes `p.renderSubBox("X", lines, width, p.theme.PaneBorderRequestFlow())`. There are 7 tests. Update each:

```go
// TestRenderSubBox_ContainsRoundedCorners
out := p.renderSubBox("APP", []string{"line1", "line2"}, 20, p.theme.PaneBorderRequestFlow())

// TestRenderSubBox_ContainsTitle
out := p.renderSubBox("APP", []string{"line1"}, 20, p.theme.PaneBorderRequestFlow())

// TestRenderSubBox_ContainsSideBorders
out := p.renderSubBox("APP", []string{"line1", "line2"}, 20, p.theme.PaneBorderRequestFlow())

// TestRenderSubBox_ContentLinesPresent
out := p.renderSubBox("GW", []string{"hello", "world"}, 20, p.theme.PaneBorderRequestFlow())

// TestRenderSubBox_LongLineTruncated
out := p.renderSubBox("T", []string{longLine}, 20, p.theme.PaneBorderRequestFlow())

// TestRenderSubBox_TooNarrowReturnsEmpty
out := p.renderSubBox("APP", []string{"hi"}, 7, p.theme.PaneBorderRequestFlow())

// TestRenderSubBox_EmptyLinesSlice
out := p.renderSubBox("APP", []string{}, 20, p.theme.PaneBorderRequestFlow())
```

- [ ] **Run tests — verify they fail to compile**

```bash
go test ./internal/ui/panes/... -run TestRenderSubBox -v
```

Expected: compile error — `renderSubBox` called with wrong number of args (tests now pass 4, function still takes 3).

- [ ] **Update `renderSubBox()` signature in `requestflow_boxed.go`**

Change the function signature from:
```go
func (p *RequestFlowPane) renderSubBox(title string, lines []string, width int) string {
```
to:
```go
func (p *RequestFlowPane) renderSubBox(title string, lines []string, width int, borderColor lipgloss.Color) string {
```

Inside the function body, replace the two lines that hard-code the border color:
```go
// BEFORE (two separate lines):
borderColor := p.theme.PaneBorderRequestFlow()
borderStyle := lipgloss.NewStyle().Foreground(borderColor)
borderChar := borderStyle.Render("│")

// AFTER (borderColor is now the param):
borderStyle := lipgloss.NewStyle().Foreground(borderColor)
borderChar := borderStyle.Render("│")
```

Also update the title styling line inside the function:
```go
// BEFORE:
titleStyled := lipgloss.NewStyle().Foreground(p.theme.PaneBorderRequestFlow()).Bold(true).Render(title)

// AFTER:
titleStyled := lipgloss.NewStyle().Foreground(borderColor).Bold(true).Render(title)
```

- [ ] **Update the three `renderSubBox` call sites in `viewBoxed()` in `requestflow_pane.go`**

Temporarily pass `p.theme.PaneBorderRequestFlow()` for all three to keep existing behavior. The correct colors are wired in Task 9.

```go
appBox := p.renderSubBox("APP", appLines, appBoxW, p.theme.PaneBorderRequestFlow())
gwBox := p.renderSubBox("GATEWAY", gwLines, gwBoxW, p.theme.PaneBorderRequestFlow())
spotifyBox := p.renderSubBox("SPOTIFY", spotifyLines, spotifyBoxW, p.theme.PaneBorderRequestFlow())
```

- [ ] **Run tests — verify they pass**

```bash
go test ./internal/ui/panes/... -run TestRenderSubBox -v
```

Expected: PASS.

- [ ] **Verify build**

```bash
go build ./internal/ui/panes/...
```

- [ ] **Commit**

```bash
git add internal/ui/panes/requestflow_boxed.go internal/ui/panes/requestflow_pane.go internal/ui/panes/requestflow_boxed_test.go
git commit -m "refactor(requestflow): renderSubBox accepts borderColor param"
```

---

## Task 5: `renderGatewayBanner()`

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go` (add new function)
- Modify: `internal/ui/panes/requestflow_boxed_test.go` (add tests)

The banner is a full-width single-row box showing token bar, slot bar, backoff state, and dedup count.

- [ ] **Write failing tests in `requestflow_boxed_test.go`**

```go
func TestRenderGatewayBanner_ContainsTokens(t *testing.T) {
	p := newInternalTestPane()
	p.displayState.snapshot = domain.GatewayStateSnapshot{
		TokensAvailable: 8,
		TokensMax:       10,
		ConcurrentActive: 0,
		ConcurrentMax:   5,
	}
	out := p.renderGatewayBanner(80)
	assert.Contains(t, out, "TOKENS")
	assert.Contains(t, out, "8/10")
}

func TestRenderGatewayBanner_ContainsSlots(t *testing.T) {
	p := newInternalTestPane()
	p.displayState.snapshot = domain.GatewayStateSnapshot{
		TokensMax: 10, ConcurrentActive: 2, ConcurrentMax: 5,
	}
	out := p.renderGatewayBanner(80)
	assert.Contains(t, out, "SLOTS")
	assert.Contains(t, out, "2/5")
}

func TestRenderGatewayBanner_BackoffNone_WhenNotThrottled(t *testing.T) {
	p := newInternalTestPane()
	p.displayState.snapshot = domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5}
	out := p.renderGatewayBanner(80)
	assert.Contains(t, out, "BACKOFF")
	assert.Contains(t, out, "none")
}

func TestRenderGatewayBanner_DedupNone_WhenZeroWaiters(t *testing.T) {
	p := newInternalTestPane()
	p.displayState.snapshot = domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5, DedupWaiters: 0}
	out := p.renderGatewayBanner(80)
	assert.Contains(t, out, "DEDUP")
	assert.Contains(t, out, "none")
}

func TestRenderGatewayBanner_DedupWaiters(t *testing.T) {
	p := newInternalTestPane()
	p.displayState.snapshot = domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5, DedupWaiters: 3}
	out := p.renderGatewayBanner(80)
	assert.Contains(t, out, "3 waiting")
}

func TestRenderGatewayBanner_HasBorders(t *testing.T) {
	p := newInternalTestPane()
	p.displayState.snapshot = domain.GatewayStateSnapshot{TokensMax: 10, ConcurrentMax: 5}
	out := p.renderGatewayBanner(80)
	assert.Contains(t, out, "╭")
	assert.Contains(t, out, "╰")
}
```

- [ ] **Run tests — verify they fail**

```bash
go test ./internal/ui/panes/... -run TestRenderGatewayBanner -v
```

Expected: FAIL — `renderGatewayBanner undefined`.

- [ ] **Add `renderGatewayBanner()` to `requestflow_pane.go`**

Add after `renderGatewayState()`:

```go
// renderGatewayBanner renders a full-width single-row box showing live gateway health.
// Content: token bar · slot bar · backoff state · dedup count.
func (p *RequestFlowPane) renderGatewayBanner(width int) string {
	snap := p.displayState.snapshot

	successStyle := lipgloss.NewStyle().Foreground(p.theme.Success())
	warnStyle := lipgloss.NewStyle().Foreground(p.theme.Warning())
	errorStyle := lipgloss.NewStyle().Foreground(p.theme.Error())
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
	secondaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())

	tokenBar := p.renderColoredDotBar(snap.TokensAvailable, snap.TokensMax, '●', '○', successStyle, mutedStyle)
	tokenSeg := secondaryStyle.Render("TOKENS") + "  " + tokenBar + "  " +
		secondaryStyle.Render(fmt.Sprintf("%d/%d", snap.TokensAvailable, snap.TokensMax))

	slotBar := p.renderColoredDotBar(snap.ConcurrentActive, snap.ConcurrentMax, '■', '□', warnStyle, mutedStyle)
	slotSeg := secondaryStyle.Render("SLOTS") + "  " + slotBar + "  " +
		secondaryStyle.Render(fmt.Sprintf("%d/%d", snap.ConcurrentActive, snap.ConcurrentMax))

	var backoffSeg string
	if p.store != nil && p.store.IsThrottled() {
		remaining := snap.BackoffRemaining
		if remaining <= 0 {
			remaining = float64(p.store.ThrottleRetryAfterSecs())
		}
		backoffSeg = errorStyle.Render(fmt.Sprintf("BACKOFF  %.1fs", remaining))
	} else {
		backoffSeg = secondaryStyle.Render("BACKOFF") + "  " + mutedStyle.Render("none")
	}

	var dedupSeg string
	if snap.DedupWaiters > 0 {
		dedupSeg = secondaryStyle.Render("DEDUP") + "  " +
			secondaryStyle.Render(fmt.Sprintf("%d waiting", snap.DedupWaiters))
	} else {
		dedupSeg = secondaryStyle.Render("DEDUP") + "  " + mutedStyle.Render("none")
	}

	sep := mutedStyle.Render("  ·  ")
	content := tokenSeg + sep + slotSeg + sep + backoffSeg + sep + dedupSeg

	return p.renderSubBox("GATEWAY", []string{content}, width, p.theme.PaneBorderRequestFlow())
}
```

- [ ] **Run tests — verify they pass**

```bash
go test ./internal/ui/panes/... -run TestRenderGatewayBanner -v
```

Expected: PASS.

- [ ] **Commit**

```bash
git add internal/ui/panes/requestflow_pane.go internal/ui/panes/requestflow_boxed_test.go
git commit -m "feat(requestflow): add renderGatewayBanner — full-width health strip"
```

---

## Task 6: `buildGatewayBoxLines()` — pure event log

**Files:**
- Modify: `internal/ui/panes/requestflow_boxed.go`
- Modify: `internal/ui/panes/requestflow_pane.go` (`renderGatewayState()` rewrite)
- Modify: `internal/ui/panes/requestflow_boxed_test.go`

Currently `buildGatewayBoxLines()` prepends state metric lines (token bar, slot bar) from `gatewayStateLines()` before the decision log. Remove that prefix so the box is a pure log.

Also rewrite `renderGatewayState()` (used by `viewFlat()`) since `gatewayStateLines()` will be deleted in Task 10.

- [ ] **Write a failing test in `requestflow_boxed_test.go`**

```go
func TestBuildGatewayBoxLines_NoStateMetrics(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	// Inject a decision event.
	injectEventInternal(p, s, domain.GatewayEvent{
		Kind: domain.EventRequestAllowed, Method: "GET", Path: "/v1/me/player",
	})
	lines := p.buildGatewayBoxLines(10)
	for _, l := range lines {
		assert.NotContains(t, l, "tokens", "gateway log box must not contain token metric header")
		assert.NotContains(t, l, "concurrent", "gateway log box must not contain semaphore metric header")
	}
}

func TestBuildGatewayBoxLines_ContainsDecisionEntry(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	injectEventInternal(p, s, domain.GatewayEvent{
		Kind: domain.EventRequestAllowed, Method: "GET", Path: "/v1/me/player",
	})
	lines := p.buildGatewayBoxLines(10)
	combined := strings.Join(lines, "\n")
	assert.Contains(t, combined, "allowed")
}
```

- [ ] **Run tests — verify the state metrics test fails**

```bash
go test ./internal/ui/panes/... -run "TestBuildGatewayBoxLines" -v
```

Expected: `TestBuildGatewayBoxLines_NoStateMetrics` FAIL (token/concurrent found), `TestBuildGatewayBoxLines_ContainsDecisionEntry` PASS.

- [ ] **Rewrite `buildGatewayBoxLines()` in `requestflow_boxed.go`**

Remove the `gatewayStateLines()` call and the state-bar prefix. The function now renders only the decision log:

```go
// buildGatewayBoxLines returns styled content lines for the GATEWAY LOG sub-box.
// Pure event stream — no state metric bars. State is shown in the GATEWAY banner.
// Events are newest-first (most recent at top). Padded to maxRows with empty strings.
func (p *RequestFlowPane) buildGatewayBoxLines(maxRows int) []string {
	if maxRows <= 0 {
		return nil
	}

	successStyle := lipgloss.NewStyle().Foreground(p.theme.Success())
	errorStyle := lipgloss.NewStyle().Foreground(p.theme.Error())
	warnStyle := lipgloss.NewStyle().Foreground(p.theme.Warning())
	secondaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
	primaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextPrimary())

	lines := make([]string, 0, maxRows)
	for i := len(p.displayState.decisions) - 1; i >= 0 && len(lines) < maxRows; i-- {
		d := p.displayState.decisions[i]
		var style lipgloss.Style
		switch d.kind {
		case domain.EventRequestEntered:
			if d.priority == domain.PriorityInteractive {
				style = primaryStyle
			} else {
				style = mutedStyle
			}
		case domain.EventRequestAllowed, domain.EventBackoffExpired,
			domain.EventDedupResolved:
			style = successStyle
		case domain.EventHttpCompleted:
			switch {
			case d.statusCode >= 200 && d.statusCode < 300:
				style = successStyle
			case d.statusCode == 429:
				style = warnStyle
			case d.statusCode >= 500:
				style = errorStyle
			default:
				style = secondaryStyle
			}
		case domain.EventRequestBlocked, domain.EventBackoffStarted:
			style = errorStyle
		case domain.EventDedupJoined:
			style = warnStyle
		case domain.EventTokenConsumed, domain.EventSemaphoreAcquired,
			domain.EventSemaphoreReleased:
			style = secondaryStyle
		case domain.EventTokenRefilled:
			style = mutedStyle
		default:
			style = mutedStyle
		}
		lines = append(lines, style.Render(d.label))
	}

	for len(lines) < maxRows {
		lines = append(lines, "")
	}
	return lines
}
```

Note: `decisionEntry` needs a `priority` field and `statusCode` field to support the color rules above. Check `requestflow_replay.go` — if `decisionEntry` doesn't have these fields, add them:

```go
// In requestflow_replay.go, update decisionEntry:
type decisionEntry struct {
	kind       domain.EventKind
	label      string
	shownAt    time.Time
	priority   domain.RequestPriority // needed for EventRequestEntered color
	statusCode int                    // needed for EventHttpCompleted color
}
```

And update `processNextEvent()` in `requestflow_pane.go` to populate them when appending to `displayState.decisions`:

```go
p.displayState.decisions = append(p.displayState.decisions, decisionEntry{
	kind:       event.Kind,
	label:      formatDecisionLabel(event),
	shownAt:    time.Now(),
	priority:   event.Priority,
	statusCode: event.StatusCode,
})
```

- [ ] **Rewrite `renderGatewayState()` in `requestflow_pane.go`**

This function is used by `viewFlat()` (the narrow-terminal fallback). It currently calls `gatewayStateLines()` which will be deleted in Task 10. Replace with a compact one-liner:

```go
// renderGatewayState renders a compact one-line gateway state summary for the flat layout.
func (p *RequestFlowPane) renderGatewayState() string {
	snap := p.displayState.snapshot
	successStyle := lipgloss.NewStyle().Foreground(p.theme.Success())
	warnStyle := lipgloss.NewStyle().Foreground(p.theme.Warning())
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
	secondaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
	tokenBar := p.renderColoredDotBar(snap.TokensAvailable, snap.TokensMax, '●', '○', successStyle, mutedStyle)
	slotBar := p.renderColoredDotBar(snap.ConcurrentActive, snap.ConcurrentMax, '■', '□', warnStyle, mutedStyle)
	return secondaryStyle.Render("tokens") + " " + tokenBar + "  " +
		secondaryStyle.Render("slots") + " " + slotBar
}
```

- [ ] **Run tests — verify they pass**

```bash
go test ./internal/ui/panes/... -run "TestBuildGatewayBoxLines" -v
```

Expected: PASS.

- [ ] **Full package build**

```bash
go build ./internal/ui/panes/...
```

- [ ] **Commit**

```bash
git add internal/ui/panes/requestflow_boxed.go internal/ui/panes/requestflow_pane.go internal/ui/panes/requestflow_replay.go internal/ui/panes/requestflow_boxed_test.go
git commit -m "refactor(requestflow): buildGatewayBoxLines is pure event log; rewrite renderGatewayState"
```

---

## Task 7: `buildSpotifyBoxLines()` — new format

**Files:**
- Modify: `internal/ui/panes/requestflow_boxed.go`
- Modify: `internal/ui/panes/requestflow_boxed_test.go`

New format per row: `[status]  [method] [path]  [latency]`. Blocked and dedup-joined requests are omitted. No padding to `maxRows`.

- [ ] **Write failing tests in `requestflow_boxed_test.go`**

```go
func TestBuildSpotifyBoxLines_ShowsStatusMethodPath(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	const reqID uint64 = 1
	injectEventInternal(p, s, domain.GatewayEvent{
		Kind: domain.EventRequestEntered, RequestID: reqID,
		Method: "GET", Path: "/v1/me/player",
	})
	injectEventInternal(p, s, domain.GatewayEvent{
		Kind: domain.EventSemaphoreAcquired, RequestID: reqID,
		Method: "GET", Path: "/v1/me/player",
	})
	injectEventInternal(p, s, domain.GatewayEvent{
		Kind: domain.EventHttpCompleted, RequestID: reqID,
		Method: "GET", Path: "/v1/me/player", StatusCode: 200, DurationMs: 43,
	})
	injectEventInternal(p, s, domain.GatewayEvent{
		Kind: domain.EventRequestAllowed, RequestID: reqID,
		Method: "GET", Path: "/v1/me/player", StatusCode: 200,
	})
	lines := p.buildSpotifyBoxLines(10)
	require.NotEmpty(t, lines)
	combined := strings.Join(lines, "\n")
	assert.Contains(t, combined, "200")
	assert.Contains(t, combined, "GET")
	assert.Contains(t, combined, "/player")   // /v1/me stripped
	assert.Contains(t, combined, "43ms")
}

func TestBuildSpotifyBoxLines_OmitsBlockedRequests(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	const reqID uint64 = 2
	injectEventInternal(p, s, domain.GatewayEvent{
		Kind: domain.EventRequestEntered, RequestID: reqID,
		Method: "PUT", Path: "/v1/me/player/volume",
	})
	injectEventInternal(p, s, domain.GatewayEvent{
		Kind: domain.EventRequestBlocked, RequestID: reqID,
		Method: "PUT", Path: "/v1/me/player/volume",
	})
	lines := p.buildSpotifyBoxLines(10)
	combined := strings.Join(lines, "\n")
	assert.NotContains(t, combined, "PUT", "blocked request must not appear in SPOTIFY box")
}

func TestBuildSpotifyBoxLines_OmitsDedupJoined(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	const reqID uint64 = 3
	injectEventInternal(p, s, domain.GatewayEvent{
		Kind: domain.EventRequestEntered, RequestID: reqID,
		Method: "GET", Path: "/v1/me/player",
	})
	injectEventInternal(p, s, domain.GatewayEvent{
		Kind: domain.EventDedupJoined, RequestID: reqID,
		Method: "GET", Path: "/v1/me/player",
	})
	lines := p.buildSpotifyBoxLines(10)
	// Only empty or no lines — dedup-joined request must not appear.
	combined := strings.Join(lines, "\n")
	assert.Empty(t, strings.TrimSpace(combined), "dedup-joined request must not appear in SPOTIFY box")
}

func TestBuildSpotifyBoxLines_InFlightShowsPlaceholder(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	const reqID uint64 = 4
	injectEventInternal(p, s, domain.GatewayEvent{
		Kind: domain.EventRequestEntered, RequestID: reqID,
		Method: "GET", Path: "/v1/me/player",
	})
	injectEventInternal(p, s, domain.GatewayEvent{
		Kind: domain.EventSemaphoreAcquired, RequestID: reqID,
		Method: "GET", Path: "/v1/me/player",
	})
	// No EventHttpCompleted — still in flight.
	lines := p.buildSpotifyBoxLines(10)
	combined := strings.Join(lines, "\n")
	assert.Contains(t, combined, "···", "in-flight request must show placeholder")
}
```

- [ ] **Run tests — verify they fail**

```bash
go test ./internal/ui/panes/... -run "TestBuildSpotifyBoxLines" -v
```

Expected: FAIL — wrong format or blocked/dedup entries present.

- [ ] **Rewrite `buildSpotifyBoxLines()` in `requestflow_boxed.go`**

```go
// buildSpotifyBoxLines returns styled content lines for the SPOTIFY sub-box.
// Only requests that reached Spotify are included — blocked and dedup-joined
// requests are omitted. No padding: the box height reflects actual HTTP traffic.
func (p *RequestFlowPane) buildSpotifyBoxLines(maxRows int) []string {
	if maxRows <= 0 {
		return nil
	}
	anims := p.sortedAnimations()
	lines := make([]string, 0, len(anims))

	secondaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())

	for _, a := range anims {
		if len(lines) >= maxRows {
			break
		}
		// Skip requests that never reached Spotify.
		if a.decision == domain.EventDedupJoined || a.decision == domain.EventRequestBlocked {
			continue
		}

		path := stripAPIPrefix(a.path)
		methodStr := secondaryStyle.Render(a.method)

		if a.phase < phaseInFlight {
			// Request is in-flight — no response yet.
			placeholder := mutedStyle.Render("···")
			lines = append(lines, fmt.Sprintf("%s  %s %s  %s",
				placeholder, methodStr, mutedStyle.Render(path), placeholder))
			continue
		}

		var statusStyle lipgloss.Style
		switch {
		case a.statusCode >= 200 && a.statusCode < 300:
			statusStyle = lipgloss.NewStyle().Foreground(p.theme.Success())
		case a.statusCode == 429:
			statusStyle = lipgloss.NewStyle().Foreground(p.theme.Warning())
		case a.statusCode >= 500:
			statusStyle = lipgloss.NewStyle().Foreground(p.theme.Error())
		default:
			statusStyle = lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
		}

		statusStr := statusStyle.Render(fmt.Sprintf("%d", a.statusCode))
		pathStr := statusStyle.Render(path)
		latStr := secondaryStyle.Render(fmt.Sprintf("%dms", a.durationMs))
		lines = append(lines, fmt.Sprintf("%s  %s %s  %s", statusStr, methodStr, pathStr, latStr))
	}
	return lines
}
```

- [ ] **Run tests — verify they pass**

```bash
go test ./internal/ui/panes/... -run "TestBuildSpotifyBoxLines" -v
```

Expected: PASS.

- [ ] **Commit**

```bash
git add internal/ui/panes/requestflow_boxed.go internal/ui/panes/requestflow_boxed_test.go
git commit -m "feat(requestflow): buildSpotifyBoxLines shows method+path+status, omits blocked/dedup"
```

---

## Task 8: `renderAutoTrafficStrip()`

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go`
- Modify: `internal/ui/panes/requestflow_pane_test.go`

- [ ] **Write failing tests in `requestflow_pane_test.go`**

```go
func TestRenderAutoTrafficStrip_PollingActive(t *testing.T) {
	p := newTestRequestFlowPane()
	p.SetSize(120, 30)
	p.Update(panes.PollingSnapshotMsg{TickIntervalMs: 1000, IsIdle: false})
	out := p.(*panes.RequestFlowPane).View()
	// The auto-traffic strip is inside the pane's View output.
	// Test via View() since renderAutoTrafficStrip is unexported.
	assert.Contains(t, out, "▶")
	assert.Contains(t, out, "1s")
	assert.Contains(t, out, "running")
}

func TestRenderAutoTrafficStrip_PollingIdle(t *testing.T) {
	p := newTestRequestFlowPane()
	p.SetSize(120, 30)
	p.Update(panes.PollingSnapshotMsg{TickIntervalMs: 1000, IsIdle: true, IdleSecs: 45})
	out := p.(*panes.RequestFlowPane).View()
	assert.Contains(t, out, "⏸")
	assert.Contains(t, out, "idle")
	assert.Contains(t, out, "45s")
}
```

Note: `newTestRequestFlowPane()` returns `*panes.RequestFlowPane` — no cast needed. Adjust if the helper returns a different type.

- [ ] **Write an internal test for the cache freshness segment in `requestflow_boxed_test.go`**

```go
func TestRenderAutoTrafficStrip_FreshDomain(t *testing.T) {
	s := state.New()
	p := newInternalTestPaneWithStore(s)
	// Store has zero-value FetchedAt for all domains — shows "never fetched".
	out := p.renderAutoTrafficStrip(120)
	assert.Contains(t, out, "AUTO-TRAFFIC")
}
```

- [ ] **Run tests — verify they fail**

```bash
go test ./internal/ui/panes/... -run "TestRenderAutoTrafficStrip" -v
```

Expected: FAIL — `renderAutoTrafficStrip undefined`.

- [ ] **Add `renderAutoTrafficStrip()` to `requestflow_pane.go`**

```go
// renderAutoTrafficStrip renders the full-width AUTO-TRAFFIC box.
// Explains why background requests appear in the pane: playback polling rate
// and library cache freshness per domain.
func (p *RequestFlowPane) renderAutoTrafficStrip(width int) string {
	ps := p.pollingState
	successStyle := lipgloss.NewStyle().Foreground(p.theme.Success())
	warnStyle := lipgloss.NewStyle().Foreground(p.theme.Warning())
	errorStyle := lipgloss.NewStyle().Foreground(p.theme.Error())
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
	sep := mutedStyle.Render("  ·  ")

	// Polling segment.
	interval := humanInterval(ps.TickIntervalMs)
	var pollSeg string
	if ps.IsIdle {
		pollSeg = warnStyle.Render(fmt.Sprintf("⏸ playback  every %s · idle %ds", interval, ps.IdleSecs))
	} else {
		pollSeg = successStyle.Render(fmt.Sprintf("▶ playback  every %s · running", interval))
	}

	// Cache freshness segments.
	type cacheEntry struct {
		name string
		fa   time.Time
		ttl  time.Duration
	}
	var entries []cacheEntry
	if p.store != nil {
		entries = []cacheEntry{
			{"playlists", p.store.PlaylistsFetchedAt(), state.PlaylistsTTL},
			{"albums", p.store.AlbumsFetchedAt(), state.AlbumsTTL},
			{"liked", p.store.LikedTracksFetchedAt(), state.LikedTracksTTL},
			{"recent", p.store.RecentPlayedFetchedAt(), state.RecentlyPlayedTTL},
		}
	}

	segments := []string{pollSeg}
	for _, e := range entries {
		if e.fa.IsZero() {
			segments = append(segments, mutedStyle.Render(e.name+"  never fetched"))
			continue
		}
		if !state.IsStale(e.fa, e.ttl) {
			segments = append(segments, mutedStyle.Render(e.name+"  fresh"))
			continue
		}
		age := humanAge(e.fa)
		if time.Since(e.fa) >= time.Hour {
			segments = append(segments, errorStyle.Render(fmt.Sprintf("⚠ %s  %s", e.name, age)))
		} else {
			segments = append(segments, warnStyle.Render(fmt.Sprintf("⚠ %s  %s", e.name, age)))
		}
	}

	content := strings.Join(segments, sep)
	return p.renderSubBox("AUTO-TRAFFIC", []string{content}, width, p.theme.ColumnPrimary())
}
```

The `state` package import is already present. Add `"time"` to the import block if not present.

- [ ] **Run tests — verify they pass**

```bash
go test ./internal/ui/panes/... -run "TestRenderAutoTrafficStrip" -v
```

Expected: PASS.

- [ ] **Commit**

```bash
git add internal/ui/panes/requestflow_pane.go internal/ui/panes/requestflow_pane_test.go internal/ui/panes/requestflow_boxed_test.go
git commit -m "feat(requestflow): add renderAutoTrafficStrip — human-readable polling and cache status"
```

---

## Task 9: `viewBoxed()` — four-zone layout

**Files:**
- Modify: `internal/ui/panes/requestflow_pane.go`

Replace the old three-box horizontal layout with: GATEWAY banner + three columns (no arrow columns) + AUTO-TRAFFIC strip.

- [ ] **Verify existing `viewBoxed` tests pass before changing**

```bash
go test ./internal/ui/panes/... -run "TestRequestFlowPane" -v
```

Note which tests exercise the boxed layout so you can verify they still pass after.

- [ ] **Rewrite `viewBoxed()` in `requestflow_pane.go`**

Replace the full function:

```go
// viewBoxed renders the four-zone redesigned layout:
//   1. GATEWAY banner — full-width health summary (3 rows)
//   2. Three-column area — APP | GATEWAY LOG | SPOTIFY
//   3. AUTO-TRAFFIC strip — full-width polling + cache status (3 rows)
func (p *RequestFlowPane) viewBoxed() string {
	const bannerHeight = 3   // border + 1 content + border
	const autoHeight = 3     // border + 1 content + border
	const spacing = 2        // 1 blank row above columns + 1 below

	boxAreaHeight := p.height - bannerHeight - autoHeight - spacing
	if boxAreaHeight < 3 {
		return p.viewFlat()
	}
	innerRows := boxAreaHeight - 2
	if innerRows < 1 {
		return p.viewFlat()
	}

	// Column widths: APP 28% | gap 2% | GATEWAY LOG 42% | gap 3% | SPOTIFY 25%
	appW := p.width * 28 / 100
	gwW := p.width * 42 / 100
	spotifyW := p.width * 25 / 100
	leftGapW := p.width * 2 / 100
	rightGapW := p.width - appW - gwW - spotifyW - leftGapW
	if rightGapW < 1 {
		rightGapW = 1
	}

	if appW < 12 {
		appW = 12
	}
	if gwW < 20 {
		gwW = 20
	}
	if spotifyW < 10 {
		spotifyW = 10
	}
	if appW+leftGapW+gwW+rightGapW+spotifyW > p.width {
		return p.viewFlat()
	}

	// Build content lines for each column.
	appLines := p.buildAppBoxLines(innerRows)
	gwLines := p.buildGatewayBoxLines(innerRows)
	spotifyLines := p.buildSpotifyBoxLines(innerRows)

	// Render the three column boxes with distinct border colors.
	appBox := p.renderSubBox("APP", appLines, appW, p.theme.ColumnPrimary())
	gwBox := p.renderSubBox("GATEWAY LOG", gwLines, gwW, p.theme.PaneBorderRequestFlow())
	spotifyBox := p.renderSubBox("SPOTIFY", spotifyLines, spotifyW, p.theme.Success())

	// Build gap blocks spanning the full column area height.
	leftGap := buildGapBlock(leftGapW, boxAreaHeight)
	rightGap := buildGapBlock(rightGapW, boxAreaHeight)

	columns := lipgloss.JoinHorizontal(lipgloss.Top, appBox, leftGap, gwBox, rightGap, spotifyBox)

	banner := p.renderGatewayBanner(p.width)
	autoTraffic := p.renderAutoTrafficStrip(p.width)

	return banner + "\n" + columns + "\n" + autoTraffic
}

// buildGapBlock returns a multi-line blank-space string for use as a gap column
// in lipgloss.JoinHorizontal. Each line is `width` spaces.
func buildGapBlock(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	line := strings.Repeat(" ", width)
	lines := make([]string, height)
	for i := range lines {
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}
```

- [ ] **Verify the pane renders without panic**

```bash
go test ./internal/ui/panes/... -run "TestNewRequestFlowPane_EmptyDisplayState|TestNewRequestFlowPane_NilStore" -v
```

Expected: PASS.

- [ ] **Run all RequestFlowPane tests**

```bash
go test ./internal/ui/panes/... -run "TestRequestFlowPane|TestRender" -v
```

Fix any failures caused by layout structure changes.

- [ ] **Commit**

```bash
git add internal/ui/panes/requestflow_pane.go
git commit -m "feat(requestflow): viewBoxed — four-zone layout with distinct colored borders"
```

---

## Task 10: Remove dead code

**Files:**
- Modify: `internal/ui/panes/requestflow_boxed.go`
- Modify: `internal/ui/panes/requestflow_pane.go`
- Modify: `internal/ui/panes/requestflow_boxed_test.go`
- Modify: `internal/ui/panes/requestflow_pane_test.go`

Delete all code made obsolete by the redesign. Delete the tests for deleted functions first, then delete the functions.

- [ ] **Delete `renderRightArrow` tests from `requestflow_boxed_test.go`**

Remove these test functions entirely:
- `TestRenderRightArrow_2xx_ContainsAnimatedArrow`
- `TestRenderRightArrow_429_ContainsX`
- `TestRenderRightArrow_500_ContainsArrow`
- `TestRenderRightArrow_StatusZero_ContainsX`
- `TestRenderRightArrow_Blocked_ContainsX`

- [ ] **Delete status strip tests from `requestflow_pane_test.go`**

Remove any test functions that reference `renderStatusStrip`, `renderStoreStatus`, or `renderStalenessStatus`. Search:

```bash
grep -n "StatusStrip\|StoreStatus\|StalenessStatus" internal/ui/panes/requestflow_pane_test.go
```

Delete each matching test function.

- [ ] **Verify tests compile**

```bash
go test ./internal/ui/panes/... -count=0
```

Expected: compile success (no runtime needed — just checking for compile errors).

- [ ] **Delete dead functions from `requestflow_boxed.go`**

Delete these functions entirely:
- `gatewayStateLines()` (lines ~116–165)
- `renderRightArrow()` (lines ~79–111)
- `buildLeftArrowLines()` (lines ~307–341)
- `buildRightArrowLines()` (lines ~343–359)

- [ ] **Delete dead functions from `requestflow_pane.go`**

Delete these functions entirely:
- `renderStatusStrip()` (line ~532)
- `renderStoreStatus()` (line ~560)
- `renderStalenessStatus()` (line ~596)

- [ ] **Update `viewFlat()` — remove `renderStatusStrip()` call**

Find `viewFlat()` and remove the two lines at the end that write the status strip:

```go
// DELETE these two lines:
sb.WriteString(p.renderStatusStrip())
// (and any preceding blank line write)
```

The function should end after `sb.WriteString(p.renderGatewayState())`.

- [ ] **Verify build**

```bash
go build ./internal/ui/panes/...
```

Expected: clean. If any `undefined` errors appear, there are missed references — fix them.

- [ ] **Run all pane tests**

```bash
go test ./internal/ui/panes/... -v
```

Expected: PASS.

- [ ] **Commit**

```bash
git add internal/ui/panes/requestflow_boxed.go internal/ui/panes/requestflow_pane.go internal/ui/panes/requestflow_boxed_test.go internal/ui/panes/requestflow_pane_test.go
git commit -m "chore(requestflow): remove dead code — arrow builders, status strip, gatewayStateLines"
```

---

## Task 11: Final CI gate

**Files:** none

- [ ] **Run full CI**

```bash
make ci
```

Expected: PASS. All lint, test, and coverage checks green.

If coverage drops below 80%, add tests targeting the new functions (`renderGatewayBanner`, `renderAutoTrafficStrip`, `buildSpotifyBoxLines`) until the threshold is met.

- [ ] **Push branch**

```bash
git push origin feat/requestflow-redesign
```

- [ ] **Open PR**

Title: `feat(requestflow): four-zone redesign with distinct colors and human-readable labels`

Body:
```
## Summary
- Replaces three same-colored boxes with four distinct zones (GATEWAY banner, APP / GATEWAY LOG / SPOTIFY columns, AUTO-TRAFFIC strip)
- Each zone has a different border color: blue (app), orange (gateway), green (Spotify)
- GATEWAY LOG is now a pure event stream; health metrics moved to the banner
- SPOTIFY box shows method + path + status + latency; omits blocked/dedup entries
- AUTO-TRAFFIC strip replaces the cryptic status footer with human-readable polling state and cache age

## Test plan
- [ ] Start the app and navigate to Page B (nerd status page)
- [ ] Verify GATEWAY banner shows token/slot bars and updates live
- [ ] Trigger a playback command — verify the Interactive request appears in APP box (bright) and GATEWAY LOG shows "✓ allowed"
- [ ] Wait for a 429 — verify BACKOFF segment in banner turns red
- [ ] Verify SPOTIFY box shows `200 GET /player 43ms` format (not just `200 43ms`)
- [ ] Verify AUTO-TRAFFIC strip shows `▶ playback every 1s · running`
- [ ] Go idle — verify strip shows `⏸ playback ... · idle Xs`
- [ ] `make ci` passes
```

---

## Self-Review Checklist

**Spec coverage:**
- [x] Zone 1 GATEWAY banner → Task 5
- [x] Zone 2 APP box color (`ColumnPrimary`) → Task 9
- [x] Zone 2 GATEWAY LOG color (`PaneBorderRequestFlow`) → Task 9
- [x] Zone 2 SPOTIFY box color (`Success`) → Task 9
- [x] GATEWAY LOG pure event stream → Task 6
- [x] Event kind color rules in GATEWAY LOG → Task 6
- [x] Path `/v1/me` stripping → Task 2
- [x] Label format rewrites → Task 2
- [x] SPOTIFY format `[status] [method] [path] [latency]` → Task 7
- [x] SPOTIFY omits blocked/dedup → Task 7
- [x] SPOTIFY in-flight placeholder `···` → Task 7
- [x] Zone 3 AUTO-TRAFFIC strip → Task 8
- [x] `humanInterval()` → Task 3
- [x] `humanAge()` → Task 3
- [x] No status strip → Task 10
- [x] `renderSubBox` color param → Task 4
- [x] Dead code removal → Task 10
- [x] `viewFlat()` update → Task 10

**Type consistency:**
- `decisionEntry` extended with `priority` and `statusCode` fields in Task 6 — used in `buildGatewayBoxLines()` in Task 6. ✓
- `buildGapBlock()` defined and called in Task 9. ✓
- `stripAPIPrefix()` defined in Task 2, used in Task 7. ✓
- `humanInterval()`, `humanAge()` defined in Task 3, used in Task 8. ✓
