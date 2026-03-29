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
	store   *state.Store
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
func NewRequestFlowPane(s *state.Store, t theme.Theme) *RequestFlowPane {
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

// viewBoxed renders the three bordered sub-boxes layout.
// Box proportions (approximate):
//
//	APP ~25% | left arrow ~8% | GATEWAY ~26% | right arrow ~8% | SPOTIFY ~20%
func (p *RequestFlowPane) viewBoxed() string {
	contentWidth := p.width
	statusStripHeight := 1
	// boxAreaHeight: subtract status strip and 1 blank separator row.
	boxAreaHeight := p.height - statusStripHeight - 1
	// Need at least 3 rows for a meaningful box (top border + 1 content row + bottom border).
	if boxAreaHeight < 3 {
		return p.viewFlat()
	}

	// Column widths (proportional to pane width).
	appBoxW := contentWidth * 25 / 100
	arrowW := contentWidth * 8 / 100
	gwBoxW := contentWidth * 26 / 100
	spotifyBoxW := contentWidth * 20 / 100

	// Enforce minimum widths so boxes are always meaningful.
	if appBoxW < 10 {
		appBoxW = 10
	}
	if arrowW < 7 {
		arrowW = 7
	}
	if gwBoxW < 12 {
		gwBoxW = 12
	}
	if spotifyBoxW < 10 {
		spotifyBoxW = 10
	}

	// Guard: if minimums push total beyond pane width, fall back to flat layout.
	if appBoxW+arrowW+gwBoxW+arrowW+spotifyBoxW > contentWidth {
		return p.viewFlat()
	}

	// Inner row count = box height minus top/bottom border rows.
	innerRows := boxAreaHeight - 2
	if innerRows < 1 {
		innerRows = 1
	}

	// Build content lines for each box.
	appLines := p.buildAppBoxLines(innerRows)
	gwLines := p.buildGatewayBoxLines(innerRows)
	spotifyLines := p.buildSpotifyBoxLines(innerRows)

	// Build arrow columns (one line per content row).
	leftArrows := p.buildLeftArrowLines(innerRows, arrowW)
	rightArrows := p.buildRightArrowLines(innerRows, arrowW)

	// Render bordered sub-boxes.
	appBox := p.renderSubBox("APP", appLines, appBoxW)
	gwBox := p.renderSubBox("GATEWAY", gwLines, gwBoxW)
	spotifyBox := p.renderSubBox("SPOTIFY", spotifyLines, spotifyBoxW)

	// Arrow blocks: pad with a blank line above and below to align
	// arrow rows with box content rows (offset by border rows).
	blankArrow := strings.Repeat(" ", arrowW)
	leftBlock := blankArrow + "\n" + strings.Join(leftArrows, "\n") + "\n" + blankArrow
	rightBlock := blankArrow + "\n" + strings.Join(rightArrows, "\n") + "\n" + blankArrow

	// Compose horizontally: APP | left arrows | GATEWAY | right arrows | SPOTIFY
	composite := lipgloss.JoinHorizontal(lipgloss.Top,
		appBox, leftBlock, gwBox, rightBlock, spotifyBox)

	return composite + "\n" + p.renderStatusStrip()
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
	sb.WriteString("\n")

	// Bottom status strip.
	sb.WriteString(p.renderStatusStrip())

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
	marker := "▶ "
	if a.phase >= phaseCompleted {
		marker = "  "
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
	case domain.EventRequestWaited:
		arrow = "── wait ──"
		style = lipgloss.NewStyle().Foreground(p.theme.Warning())
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

// renderGatewayState renders the GATEWAY column details (token bucket, semaphore, backoff).
// Delegates to gatewayStateLines() defined in requestflow_boxed.go for DRY reuse.
func (p *RequestFlowPane) renderGatewayState() string {
	return strings.Join(p.gatewayStateLines(), "\n")
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
		kind:    event.Kind,
		label:   formatDecisionLabel(event),
		shownAt: time.Now(),
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
	case domain.EventRequestWaited:
		anim.phase = phaseAtGateway
		anim.decision = domain.EventRequestWaited
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
	completedCutoff := now.Add(-5 * time.Second)
	for id, anim := range p.displayState.requests {
		if anim.phase >= phaseCompleted && anim.enteredAt.Before(completedCutoff) {
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

// formatDecisionLabel builds the display string for a decision log entry.
func formatDecisionLabel(e domain.GatewayEvent) string {
	switch e.Kind {
	case domain.EventRequestEntered:
		tag := "bg"
		if e.Priority == domain.PriorityInteractive {
			tag = "int"
		}
		return fmt.Sprintf("→ %s %s entered [%s]", e.Method, e.Path, tag)
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
		return fmt.Sprintf("⏳ backoff started %.0fs", e.Snapshot.BackoffRemaining)
	case domain.EventBackoffExpired:
		return "✓ backoff cleared"
	case domain.EventRequestAllowed:
		return fmt.Sprintf("✓ %s %s allowed", e.Method, e.Path)
	case domain.EventRequestWaited:
		return fmt.Sprintf("⧖ %s %s waited", e.Method, e.Path)
	case domain.EventRequestBlocked:
		return fmt.Sprintf("✗ %s %s blocked", e.Method, e.Path)
	case domain.EventDedupJoined:
		return fmt.Sprintf("⧖ %s %s dedup", e.Method, e.Path)
	case domain.EventDedupResolved:
		return fmt.Sprintf("✓ dedup resolved %d", e.StatusCode)
	case domain.EventHttpCompleted:
		return fmt.Sprintf("✓ %d %dms", e.StatusCode, e.DurationMs)
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

// renderStatusStrip renders the bottom polling + store status line.
func (p *RequestFlowPane) renderStatusStrip() string {
	ps := p.pollingState
	labelStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())

	// Polling section.
	stateLabel := "active"
	idlePart := ""
	if ps.IsIdle {
		stateLabel = "idle"
		idlePart = mutedStyle.Render(fmt.Sprintf("  idle: %ds", ps.IdleSecs))
	}
	intervalMs := ps.TickIntervalMs
	if intervalMs <= 0 {
		intervalMs = 1000
	}
	pollingPart := labelStyle.Render("POLLING") + mutedStyle.Render(fmt.Sprintf("  tick: %dms  state: %s", intervalMs, stateLabel)) + idlePart

	// Store section.
	storePart := p.renderStoreStatus()

	if storePart != "" {
		return pollingPart + "    " + storePart
	}
	return pollingPart
}

// renderStoreStatus renders the STORE section of the status strip.
// Shows active fetches and, when present, stale data domains.
func (p *RequestFlowPane) renderStoreStatus() string {
	if p.store == nil {
		return ""
	}
	labelStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())

	var fetching []string
	if p.store.PlaylistsFetching() {
		fetching = append(fetching, "playlists")
	}
	if p.store.AlbumsFetching() {
		fetching = append(fetching, "albums")
	}
	if p.store.LikedFetching() {
		fetching = append(fetching, "liked")
	}
	if p.store.RecentFetching() {
		fetching = append(fetching, "recent")
	}

	result := labelStyle.Render("STORE")
	if len(fetching) > 0 {
		result += mutedStyle.Render(fmt.Sprintf("  fetching: [%s]", strings.Join(fetching, ", ")))
	}

	stalePart := p.renderStalenessStatus()
	if stalePart != "" {
		result += "  " + stalePart
	}

	return result
}

// renderStalenessStatus builds the "stale: domain(Xs), ..." segment.
// Only non-zero FetchedAt values that exceed their TTL are shown.
// Returns empty string when no data is stale.
func (p *RequestFlowPane) renderStalenessStatus() string {
	if p.store == nil {
		return ""
	}
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())

	var stale []string
	if fa := p.store.PlaylistsFetchedAt(); !fa.IsZero() && state.IsStale(fa, state.PlaylistsTTL) {
		stale = append(stale, fmt.Sprintf("playlists(%ds)", int(time.Since(fa).Seconds())))
	}
	if fa := p.store.AlbumsFetchedAt(); !fa.IsZero() && state.IsStale(fa, state.AlbumsTTL) {
		stale = append(stale, fmt.Sprintf("albums(%ds)", int(time.Since(fa).Seconds())))
	}
	if fa := p.store.LikedTracksFetchedAt(); !fa.IsZero() && state.IsStale(fa, state.LikedTracksTTL) {
		stale = append(stale, fmt.Sprintf("liked(%ds)", int(time.Since(fa).Seconds())))
	}
	if fa := p.store.RecentPlayedFetchedAt(); !fa.IsZero() && state.IsStale(fa, state.RecentlyPlayedTTL) {
		stale = append(stale, fmt.Sprintf("recent(%ds)", int(time.Since(fa).Seconds())))
	}
	if len(stale) == 0 {
		return ""
	}
	return mutedStyle.Render(fmt.Sprintf("stale: %s", strings.Join(stale, ", ")))
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
