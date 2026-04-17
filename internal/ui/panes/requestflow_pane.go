package panes

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components/viz"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// PollingSnapshotMsg carries app-level polling state to the RequestFlowPane.
// The app sends this on each TickMsg so the pane can display polling diagnostics.
type PollingSnapshotMsg struct {
	// TickIntervalMs is the current playback polling interval in milliseconds.
	TickIntervalMs int
	// IsIdle is true when the user has not pressed a key for idleThresholdSecs.
	IsIdle bool
	// IdleSecs is how long the user has been idle (0 when not idle).
	IdleSecs int
}

// RequestFlowPane visualizes the live APP → GATEWAY → SPOTIFY request pipeline.
// It reads gateway lifecycle events from the store's event journal and replays
// them at human-observable speed (one event per viz.TickMsg, 200ms minimum visibility).
// It does NOT make any Spotify API calls — all data is internal infrastructure state.
type RequestFlowPane struct {
	theme   theme.Theme
	store   state.StateReader
	focused bool
	width   int
	height  int

	// frameIndex is the animation frame counter, advanced on each viz.TickMsg.
	frameIndex int

	// eventCursor is the cursor into the GatewayEventLog; advances as events are drained.
	eventCursor uint64
	// replayQueue holds events waiting to be replayed (one per tick).
	replayQueue []domain.GatewayEvent
	// displayState is what View() reads from; updated by the replay loop.
	displayState replayDisplayState

	// pollingState is the latest app-level polling snapshot.
	pollingState PollingSnapshotMsg
}

// Compile-time check: RequestFlowPane implements layout.Pane.
var _ layout.Pane = &RequestFlowPane{}

// NewRequestFlowPane creates a RequestFlowPane that reads gateway events from
// the store's event log. The pane does not hold a gateway reference — it only
// reads from the store, preserving the ui/ → state/ dependency direction.
func NewRequestFlowPane(s state.StateReader, t theme.Theme) *RequestFlowPane {
	return &RequestFlowPane{
		theme: t,
		store: s,
		displayState: replayDisplayState{
			requests: make(map[uint64]*requestAnimation),
			snapshot: domain.GatewayStateSnapshot{
				TokensAvailable: 10,
				TokensMax:       10,
				ConcurrentMax:   5,
			},
		},
	}
}

// ID returns the PaneID for the RequestFlow slot.
func (p *RequestFlowPane) ID() layout.PaneID { return layout.PaneRequestFlow }

// Title returns the display title shown in the pane border.
func (p *RequestFlowPane) Title() string { return "Request Flow" }

// ToggleKey returns 0 — Page B panes are not individually toggleable.
func (p *RequestFlowPane) ToggleKey() int { return 0 }

// Actions returns nil — the RequestFlowPane has no interactive shortcuts.
func (p *RequestFlowPane) Actions() []layout.Action { return nil }

// SetSize updates the content area dimensions.
func (p *RequestFlowPane) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// SetFocused updates the keyboard focus state.
func (p *RequestFlowPane) SetFocused(focused bool) { p.focused = focused }

// IsFocused returns whether this pane has keyboard focus.
func (p *RequestFlowPane) IsFocused() bool { return p.focused }

// FrameIndex returns the current animation frame index (exported for testing).
func (p *RequestFlowPane) FrameIndex() int { return p.frameIndex }

// Init returns nil — RequestFlowPane has no self-initiated tick loop.
// It reacts to TickMsg (1s) and viz.TickMsg (200ms) from the app.
func (p *RequestFlowPane) Init() tea.Cmd { return nil }

// Update handles incoming messages.
func (p *RequestFlowPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case viz.TickMsg:
		// Advance the arrow animation frame counter.
		p.frameIndex++
		// Drain new events, replay one, age out old entries.
		p.drainEvents()
		p.processNextEvent()
		p.ageOutEntries()
		return p, nil

	case TickMsg:
		// TickMsg also triggers a drain/process cycle and updates polling state.
		p.drainEvents()
		p.processNextEvent()
		p.ageOutEntries()
		return p, nil

	case PollingSnapshotMsg:
		p.pollingState = m
		return p, nil
	}
	return p, nil
}

// View renders the full RequestFlowPane. Pure — no side effects.
// When pane width >= 60, renders three bordered sub-boxes (APP, GATEWAY, SPOTIFY)
// with dual arrow columns. Falls back to flat table layout for narrower terminals.
func (p *RequestFlowPane) View() string {
	if p.width <= 0 || p.height <= 0 {
		return ""
	}

	// Minimum content width for three bordered boxes.
	if p.width < 60 {
		return p.viewFlat()
	}

	return p.viewBoxed()
}

// viewBoxed renders the four-zone redesigned layout:
//  1. GATEWAY banner — full-width health summary (3 rows)
//  2. Three-column area — APP | GATEWAY LOG | SPOTIFY (remaining height)
//  3. AUTO-TRAFFIC strip — full-width polling + cache status (3 rows)
func (p *RequestFlowPane) viewBoxed() string {
	const bannerHeight = 3 // border + 1 content + border
	const autoHeight = 3   // border + 1 content + border
	const spacing = 2      // 1 blank row above columns + 1 below

	boxAreaHeight := p.height - bannerHeight - autoHeight - spacing
	if boxAreaHeight < 3 {
		return p.viewFlat()
	}
	innerRows := boxAreaHeight - 2 // subtract top/bottom border of column boxes
	if innerRows < 1 {
		return p.viewFlat()
	}

	// Column widths: APP 28% | gap 2% | GATEWAY LOG (remainder) | gap 2% | SPOTIFY 25%
	// Gaps are equal so the layout is visually balanced. GATEWAY LOG absorbs all rounding.
	appW := p.width * 28 / 100
	spotifyW := p.width * 25 / 100
	leftGapW := p.width * 2 / 100
	rightGapW := leftGapW
	gwW := p.width - appW - spotifyW - leftGapW - rightGapW
	if rightGapW < 1 {
		rightGapW = 1
	}

	// Enforce minimum widths.
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

	// Gap blocks spanning the full column box area height (boxAreaHeight rows).
	leftGap := buildGapBlock(leftGapW, boxAreaHeight)
	rightGap := buildGapBlock(rightGapW, boxAreaHeight)

	columns := lipgloss.JoinHorizontal(lipgloss.Top, appBox, leftGap, gwBox, rightGap, spotifyBox)

	banner := p.renderGatewayBanner(p.width)
	autoTraffic := p.renderAutoTrafficStrip(p.width)

	return banner + "\n" + columns + "\n" + autoTraffic
}

// buildGapBlock returns a blank-space block for use as a gap column
// in lipgloss.JoinHorizontal. Each line is `width` spaces; the block is `height` lines tall.
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

// viewFlat renders the original flat table layout. Used as a fallback for narrow
// terminals (width < 60).
func (p *RequestFlowPane) viewFlat() string {
	var sb strings.Builder

	// Row 1: column headers.
	colWidth := paneMax(p.width/3, 12)
	sb.WriteString(p.renderColumnHeaders(colWidth))
	sb.WriteString("\n")

	// Rows 2+: per-request rows (APP → GATEWAY → SPOTIFY).
	rows := p.renderRequestRows(colWidth)
	for _, row := range rows {
		sb.WriteString(row)
		sb.WriteString("\n")
	}

	// Gateway state block (token bucket, semaphore, backoff, dedup).
	sb.WriteString(p.renderGatewayState())

	return sb.String()
}

// renderColumnHeaders renders the three column headers.
func (p *RequestFlowPane) renderColumnHeaders(colWidth int) string {
	headerStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary()).Bold(true)
	app := padRightVisible(headerStyle.Render("APP"), colWidth)
	gw := padRightVisible(headerStyle.Render("GATEWAY"), colWidth)
	spotify := headerStyle.Render("SPOTIFY")
	return app + gw + spotify
}

// renderRequestRows renders one row per active request in the flat layout.
func (p *RequestFlowPane) renderRequestRows(colWidth int) []string {
	anims := p.sortedAnimations()
	if len(anims) == 0 {
		return nil
	}
	rows := make([]string, 0, len(anims))
	for _, a := range anims {
		appCol := p.renderFlatAppEntry(a, colWidth)
		arrowCol := p.renderFlatArrow(a, colWidth)
		spotifyCol := p.renderFlatSpotifyEntry(a)
		rows = append(rows, appCol+arrowCol+spotifyCol)
	}
	return rows
}

// renderFlatAppEntry renders one line in the APP column (flat layout).
// Interactive priority requests are shown in TextPrimary; Background in TextMuted.
func (p *RequestFlowPane) renderFlatAppEntry(a *requestAnimation, colWidth int) string {
	marker := "⚡"
	if a.priority != domain.PriorityInteractive {
		marker = "◷"
	}
	if a.phase >= phaseCompleted {
		marker = " "
	}
	ep := truncateStr(a.path, colWidth-2)
	text := marker + ep

	var style lipgloss.Style
	if a.phase >= phaseCompleted {
		style = lipgloss.NewStyle().Foreground(p.theme.TextMuted())
	} else if a.priority == domain.PriorityInteractive {
		style = lipgloss.NewStyle().Foreground(p.theme.TextPrimary())
	} else {
		style = lipgloss.NewStyle().Foreground(p.theme.TextMuted())
	}
	return padRightVisible(style.Render(text), colWidth)
}

// renderFlatArrow renders the left arrow (APP→GATEWAY) for the flat layout.
// Arrow style reflects the gateway decision based on requestAnimation fields.
func (p *RequestFlowPane) renderFlatArrow(a *requestAnimation, colWidth int) string {
	frames := []string{"──→──", "───→─", "────→"}

	var arrow string
	var style lipgloss.Style

	switch a.decision {
	case domain.EventDedupJoined:
		arrow = "──→ dedup"
		style = lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
	case domain.EventRequestBlocked:
		arrow = "── ╳ ──"
		style = lipgloss.NewStyle().Foreground(p.theme.Error())
	default:
		if a.statusCode == 429 {
			arrow = "── ╳ ─"
			style = lipgloss.NewStyle().Foreground(p.theme.Warning())
		} else {
			arrow = frames[p.frameIndex%3]
			style = lipgloss.NewStyle().Foreground(p.theme.Success())
		}
	}

	return padRightVisible(style.Render(arrow), colWidth)
}

// renderFlatSpotifyEntry renders the SPOTIFY column for one request (flat layout).
// Status codes are color-coded: 2xx=Success, 429=Warning, 5xx=Error, 0=TextMuted.
func (p *RequestFlowPane) renderFlatSpotifyEntry(a *requestAnimation) string {
	latencyStr := fmt.Sprintf("%dms", a.durationMs)

	var statusStyle lipgloss.Style
	suffix := ""
	switch {
	case a.statusCode == 0:
		statusStyle = lipgloss.NewStyle().Foreground(p.theme.TextMuted())
	case a.statusCode >= 200 && a.statusCode < 300:
		statusStyle = lipgloss.NewStyle().Foreground(p.theme.Success())
	case a.statusCode == 429:
		statusStyle = lipgloss.NewStyle().Foreground(p.theme.Warning())
		suffix = " ⚠"
	case a.statusCode >= 500:
		statusStyle = lipgloss.NewStyle().Foreground(p.theme.Error())
	default:
		statusStyle = lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
	}

	statusStr := statusStyle.Render(fmt.Sprintf("%-6d", a.statusCode))
	return fmt.Sprintf("%s %-8s%s", statusStr, latencyStr, suffix)
}

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

// renderAutoTrafficStrip renders a full-width AUTO-TRAFFIC box explaining why
// background requests appear. It shows the current playback polling state and
// library cache freshness for playlists, albums, liked tracks, and recent plays.
func (p *RequestFlowPane) renderAutoTrafficStrip(width int) string {
	successStyle := lipgloss.NewStyle().Foreground(p.theme.Success())
	warnStyle := lipgloss.NewStyle().Foreground(p.theme.Warning())
	errorStyle := lipgloss.NewStyle().Foreground(p.theme.Error())
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
	sep := mutedStyle.Render("  ·  ")

	// Polling segment.
	interval := humanInterval(p.pollingState.TickIntervalMs)
	var pollSeg string
	if p.pollingState.IsIdle {
		pollSeg = warnStyle.Render(fmt.Sprintf("⏸ playback  every %s · idle %ds", interval, p.pollingState.IdleSecs))
	} else {
		pollSeg = successStyle.Render(fmt.Sprintf("▶ playback  every %s · running", interval))
	}

	// Cache freshness segments.
	segments := []string{pollSeg}
	if p.store != nil {
		type cacheEntry struct {
			name string
			fa   time.Time
			ttl  time.Duration
		}
		entries := []cacheEntry{
			{"playlists", p.store.PlaylistsFetchedAt(), state.PlaylistsTTL},
			{"albums", p.store.AlbumsFetchedAt(), state.AlbumsTTL},
			{"liked", p.store.LikedTracksFetchedAt(), state.LikedTracksTTL},
			{"recent", p.store.RecentPlayedFetchedAt(), state.RecentlyPlayedTTL},
		}
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
	}

	content := strings.Join(segments, sep)
	return p.renderSubBox("AUTO-TRAFFIC", []string{content}, width, p.theme.ColumnPrimary())
}

// --- Replay engine ---

// drainEvents reads new events from the store's event log and appends
// them to the replay queue.
func (p *RequestFlowPane) drainEvents() {
	if p.store == nil {
		return
	}
	newCursor, events := p.store.ReadEventsFrom(p.eventCursor)
	p.eventCursor = newCursor
	p.replayQueue = append(p.replayQueue, events...)
}

// processNextEvent pops one event from the replay queue and updates
// the display state. One event per tick = 200ms minimum visibility.
func (p *RequestFlowPane) processNextEvent() {
	if len(p.replayQueue) == 0 {
		return
	}
	event := p.replayQueue[0]
	p.replayQueue = p.replayQueue[1:]

	// Update snapshot to this event's state.
	p.displayState.snapshot = event.Snapshot

	// Process request-scoped events.
	if event.RequestID > 0 {
		p.processRequestEvent(event)
	}

	// Append to decision log.
	p.displayState.decisions = append(p.displayState.decisions, decisionEntry{
		kind:       event.Kind,
		label:      formatDecisionLabel(event),
		shownAt:    time.Now(),
		priority:   event.Priority,
		statusCode: event.StatusCode,
	})
}

// processRequestEvent updates the requestAnimation for the given event.
func (p *RequestFlowPane) processRequestEvent(event domain.GatewayEvent) {
	anim, exists := p.displayState.requests[event.RequestID]
	if !exists {
		anim = &requestAnimation{
			requestID: event.RequestID,
			method:    event.Method,
			path:      event.Path,
			priority:  event.Priority,
			phase:     phaseEntered,
			enteredAt: time.Now(),
		}
		p.displayState.requests[event.RequestID] = anim
	}

	switch event.Kind {
	case domain.EventRequestEntered:
		anim.phase = phaseEntered
	case domain.EventSemaphoreAcquired:
		anim.phase = phaseAtGateway
	case domain.EventDedupJoined:
		anim.phase = phaseAtGateway
		anim.decision = domain.EventDedupJoined
	case domain.EventHttpCompleted:
		anim.phase = phaseInFlight
		anim.statusCode = event.StatusCode
		anim.durationMs = event.DurationMs
	case domain.EventRequestAllowed:
		anim.phase = phaseCompleted
		if anim.decision == 0 {
			anim.decision = domain.EventRequestAllowed
		}
	case domain.EventRequestBlocked:
		anim.phase = phaseCompleted
		anim.decision = domain.EventRequestBlocked
	case domain.EventDedupResolved:
		anim.phase = phaseCompleted
		anim.statusCode = event.StatusCode
		anim.decision = domain.EventDedupJoined
	}
}

// ageOutEntries removes old decisions and completed requests.
func (p *RequestFlowPane) ageOutEntries() {
	now := time.Now()
	// Age out decisions older than 3s.
	cutoff := now.Add(-3 * time.Second)
	filtered := p.displayState.decisions[:0]
	for _, d := range p.displayState.decisions {
		if d.shownAt.After(cutoff) {
			filtered = append(filtered, d)
		}
	}
	p.displayState.decisions = filtered

	// Age out completed requests older than 5s.
	// Also force-remove any request stuck in an incomplete phase for more than
	// 30s — this covers the ring-buffer overflow case where EventRequestAllowed
	// was overwritten before drainEvents() could read it, leaving the animation
	// permanently below phaseCompleted.
	completedCutoff := now.Add(-5 * time.Second)
	staleCutoff := now.Add(-30 * time.Second)
	for id, anim := range p.displayState.requests {
		if anim.phase >= phaseCompleted && anim.enteredAt.Before(completedCutoff) {
			delete(p.displayState.requests, id)
		} else if anim.enteredAt.Before(staleCutoff) {
			delete(p.displayState.requests, id)
		}
	}
}

// sortedAnimations returns active request animations sorted by enteredAt, newest first.
func (p *RequestFlowPane) sortedAnimations() []*requestAnimation {
	if len(p.displayState.requests) == 0 {
		return nil
	}
	out := make([]*requestAnimation, 0, len(p.displayState.requests))
	for _, a := range p.displayState.requests {
		out = append(out, a)
	}
	// Sort newest-first (by enteredAt descending).
	for i := 0; i < len(out)-1; i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].enteredAt.After(out[i].enteredAt) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// stripAPIPrefix removes the "/v1/me" prefix common to all Spotify API paths.
func stripAPIPrefix(path string) string {
	return strings.TrimPrefix(path, "/v1/me")
}

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

// renderColoredDotBar renders a progress bar using filled/empty rune characters
// with distinct lipgloss styles for the filled and empty portions.
func (p *RequestFlowPane) renderColoredDotBar(filled, total int, filledRune, emptyRune rune, filledStyle, emptyStyle lipgloss.Style) string {
	if total <= 0 {
		return ""
	}
	var sb strings.Builder
	for i := 0; i < total; i++ {
		if i < filled {
			sb.WriteString(filledStyle.Render(string(filledRune)))
		} else {
			sb.WriteString(emptyStyle.Render(string(emptyRune)))
		}
	}
	return sb.String()
}

// padRightVisible pads s with spaces to visible width w using lipgloss.Width()
// to measure the string's visible character count, correctly ignoring ANSI
// escape sequences. Use this for styled strings that contain ANSI codes.
func padRightVisible(s string, w int) string {
	visibleWidth := lipgloss.Width(s)
	if visibleWidth >= w {
		return s
	}
	return s + strings.Repeat(" ", w-visibleWidth)
}

// SetTheme updates the theme reference for runtime theme switching.
func (p *RequestFlowPane) SetTheme(th theme.Theme) {
	p.theme = th
}

// truncateStr truncates s to at most max runes.
func truncateStr(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return string(runes[:max-1]) + "…"
}
