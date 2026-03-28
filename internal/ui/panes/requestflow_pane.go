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

// maxRecentReqs is the maximum number of recent requests displayed in the APP column.
const maxRecentReqs = 6

// requestAgeOut is how long a completed request stays visible in the flow view.
const requestAgeOut = 5 * time.Second

// requestDimAge is the age after which a completed request is shown dimmed.
const requestDimAge = 3 * time.Second

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

// RequestCompletedMsg is sent when an API request completes so RequestFlowPane
// can add it to its recent-requests display.
type RequestCompletedMsg struct {
	// Endpoint is the API path (e.g., "/me/player").
	Endpoint string
	// StatusCode is the HTTP response status (200, 204, 429, 500, etc.).
	StatusCode int
	// LatencyMs is the round-trip time in milliseconds.
	LatencyMs int
	// Priority is domain.PriorityInteractive or domain.PriorityBackground.
	Priority domain.RequestPriority
	// GatewayDecision is the gateway routing outcome (allowed/waited/deduped/blocked).
	// Defaults to DecisionAllowed (zero value) for backward compatibility.
	GatewayDecision domain.GatewayDecision
	// CompletedAt is when the request completed. Zero value means time.Now().
	CompletedAt time.Time
}

// reqDisplay holds display state for one recently completed request.
type reqDisplay struct {
	endpoint    string
	statusCode  int
	latencyMs   int
	priority    domain.RequestPriority
	decision    domain.GatewayDecision
	completedAt time.Time
}

// RequestFlowPane visualizes the live APP → GATEWAY → SPOTIFY request pipeline.
// It reads from domain.GatewaySnapshotter (via Snapshot()) and *state.Store. It does NOT make
// any Spotify API calls — all data is internal infrastructure state.
type RequestFlowPane struct {
	theme   theme.Theme
	gateway domain.GatewaySnapshotter
	store   *state.Store
	focused bool
	width   int
	height  int

	// frameIndex is the animation frame counter, advanced on each viz.TickMsg.
	frameIndex int

	// recentReqs stores the last maxRecentReqs completed requests.
	recentReqs []reqDisplay

	// lastSnapshot is the most recent gateway state, refreshed on TickMsg.
	lastSnapshot domain.GatewayState

	// pollingState is the latest app-level polling snapshot.
	pollingState PollingSnapshotMsg
}

// Compile-time check: RequestFlowPane implements layout.Pane.
var _ layout.Pane = &RequestFlowPane{}

// NewRequestFlowPane creates a RequestFlowPane with the given gateway, store, and theme.
func NewRequestFlowPane(gw domain.GatewaySnapshotter, s *state.Store, t theme.Theme) *RequestFlowPane {
	p := &RequestFlowPane{
		theme:   t,
		gateway: gw,
		store:   s,
	}
	if gw != nil {
		p.lastSnapshot = gw.Snapshot()
	}
	return p
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
		return p, nil

	case TickMsg:
		// Refresh gateway snapshot and sync requests from net log.
		if p.gateway != nil {
			p.lastSnapshot = p.gateway.Snapshot()
		}
		p.syncFromNetLog()
		return p, nil

	case PollingSnapshotMsg:
		p.pollingState = m
		return p, nil

	case RequestCompletedMsg:
		at := m.CompletedAt
		if at.IsZero() {
			at = time.Now()
		}
		entry := reqDisplay{
			endpoint:    m.Endpoint,
			statusCode:  m.StatusCode,
			latencyMs:   m.LatencyMs,
			priority:    m.Priority,
			decision:    m.GatewayDecision,
			completedAt: at,
		}
		// Prepend (newest first), cap at maxRecentReqs.
		p.recentReqs = append([]reqDisplay{entry}, p.recentReqs...)
		if len(p.recentReqs) > maxRecentReqs {
			p.recentReqs = p.recentReqs[:maxRecentReqs]
		}
		return p, nil
	}
	return p, nil
}

// syncFromNetLog reads the store's network log and populates recentReqs
// with the most recent entries within the requestAgeOut window.
func (p *RequestFlowPane) syncFromNetLog() {
	if p.store == nil {
		return
	}
	entries := p.store.NetLogEntries()
	cutoff := time.Now().Add(-requestAgeOut)

	// Rebuild from store — newest first, capped at maxRecentReqs.
	p.recentReqs = p.recentReqs[:0]
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if e.Timestamp.Before(cutoff) {
			continue
		}
		p.recentReqs = append(p.recentReqs, reqDisplay{
			endpoint:    e.Path,
			statusCode:  e.StatusCode,
			latencyMs:   int(e.DurationMs),
			priority:    e.Priority,
			decision:    e.GatewayDecision,
			completedAt: e.Timestamp,
		})
		if len(p.recentReqs) >= maxRecentReqs {
			break
		}
	}
}

// View renders the full RequestFlowPane. Pure — no side effects.
func (p *RequestFlowPane) View() string {
	if p.width <= 0 || p.height <= 0 {
		return ""
	}

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

// renderRequestRows renders one row per recent request showing APP → GATEWAY → SPOTIFY.
func (p *RequestFlowPane) renderRequestRows(colWidth int) []string {
	if len(p.recentReqs) == 0 {
		return nil
	}

	rows := make([]string, 0, len(p.recentReqs))
	for _, r := range p.recentReqs {
		appCol := p.renderAppEntry(r, colWidth)
		arrowCol := p.renderArrow(r, colWidth)
		spotifyCol := p.renderSpotifyEntry(r)
		rows = append(rows, appCol+arrowCol+spotifyCol)
	}
	return rows
}

// renderAppEntry renders one line in the APP column.
// Interactive priority requests are shown in TextPrimary; Background in TextMuted.
// Requests older than requestDimAge are always dimmed regardless of priority.
func (p *RequestFlowPane) renderAppEntry(r reqDisplay, colWidth int) string {
	age := time.Since(r.completedAt)
	marker := "  "
	if age < requestDimAge {
		marker = "▶ "
	}
	ep := truncateStr(r.endpoint, colWidth-2)
	text := marker + ep

	var style lipgloss.Style
	if age >= requestDimAge {
		style = lipgloss.NewStyle().Foreground(p.theme.TextMuted())
	} else if r.priority == domain.PriorityInteractive {
		style = lipgloss.NewStyle().Foreground(p.theme.TextPrimary())
	} else {
		style = lipgloss.NewStyle().Foreground(p.theme.TextMuted())
	}
	return padRightVisible(style.Render(text), colWidth)
}

// renderArrow renders the connecting arrow between APP and GATEWAY columns.
// The arrow style reflects the gateway decision:
//   - DecisionAllowed: animated flowing arrow (Success color); ╳ if HTTP 429
//   - DecisionWaited:  "── wait ──" (Warning color)
//   - DecisionDeduped: "──→ dedup" (TextSecondary color)
//   - DecisionBlocked: "── ╳ ──"  (Error color — Background blocked by backoff)
func (p *RequestFlowPane) renderArrow(r reqDisplay, colWidth int) string {
	frames := []string{"──→──", "───→─", "────→"}

	var arrow string
	var style lipgloss.Style

	switch r.decision {
	case domain.DecisionWaited:
		arrow = "── wait ──"
		style = lipgloss.NewStyle().Foreground(p.theme.Warning())
	case domain.DecisionDeduped:
		arrow = "──→ dedup"
		style = lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
	case domain.DecisionBlocked:
		arrow = "── ╳ ──"
		style = lipgloss.NewStyle().Foreground(p.theme.Error())
	default:
		// DecisionAllowed: animate unless the HTTP response was 429.
		if r.statusCode == 429 {
			arrow = "── ╳ ─"
			style = lipgloss.NewStyle().Foreground(p.theme.Warning())
		} else {
			arrow = frames[p.frameIndex%3]
			style = lipgloss.NewStyle().Foreground(p.theme.Success())
		}
	}

	return padRightVisible(style.Render(arrow), colWidth)
}

// renderSpotifyEntry renders the SPOTIFY column for one request.
// Status codes are color-coded: 2xx=Success, 429=Warning, 5xx=Error, 0=TextMuted.
func (p *RequestFlowPane) renderSpotifyEntry(r reqDisplay) string {
	latencyStr := fmt.Sprintf("%dms", r.latencyMs)

	var statusStyle lipgloss.Style
	suffix := ""
	switch {
	case r.statusCode == 0:
		statusStyle = lipgloss.NewStyle().Foreground(p.theme.TextMuted())
	case r.statusCode >= 200 && r.statusCode < 300:
		statusStyle = lipgloss.NewStyle().Foreground(p.theme.Success())
	case r.statusCode == 429:
		statusStyle = lipgloss.NewStyle().Foreground(p.theme.Warning())
		suffix = " ⚠"
	case r.statusCode >= 500:
		statusStyle = lipgloss.NewStyle().Foreground(p.theme.Error())
	default:
		statusStyle = lipgloss.NewStyle().Foreground(p.theme.TextSecondary())
	}

	statusStr := statusStyle.Render(fmt.Sprintf("%-6d", r.statusCode))
	return fmt.Sprintf("%s %-8s%s", statusStr, latencyStr, suffix)
}

// renderGatewayState renders the GATEWAY column details (token bucket, semaphore, backoff).
func (p *RequestFlowPane) renderGatewayState() string {
	snap := p.lastSnapshot

	successStyle := lipgloss.NewStyle().Foreground(p.theme.Success())
	warnStyle := lipgloss.NewStyle().Foreground(p.theme.Warning())
	errorStyle := lipgloss.NewStyle().Foreground(p.theme.Error())
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.TextMuted())
	secondaryStyle := lipgloss.NewStyle().Foreground(p.theme.TextSecondary())

	// Token bucket bar: ● (Success) for available, ○ (muted) for consumed.
	tokenBar := p.renderColoredDotBar(snap.TokensAvailable, snap.TokensMax, '●', '○', successStyle, mutedStyle)
	tokenLine := fmt.Sprintf("tokens  %s %d/%d", tokenBar, snap.TokensAvailable, snap.TokensMax)

	// Semaphore bar: ■ (Warning) for in-use, □ (muted) for available.
	semBar := p.renderColoredDotBar(snap.ConcurrentActive, snap.ConcurrentMax, '■', '□', warnStyle, mutedStyle)
	semLine := fmt.Sprintf("conc    %s %d/%d", semBar, snap.ConcurrentActive, snap.ConcurrentMax)

	lines := []string{tokenLine, semLine}

	// Backoff timer: only show when store is throttled.
	if p.store != nil && p.store.IsThrottled() {
		remaining := snap.BackoffRemaining
		if remaining <= 0 {
			remaining = float64(p.store.ThrottleRetryAfterSecs())
		}
		lines = append(lines, errorStyle.Render(fmt.Sprintf("⏳ backoff %.1fs", remaining)))
	}

	// Dedup waiters: only show when active.
	if snap.DedupWaiters > 0 {
		lines = append(lines, secondaryStyle.Render(fmt.Sprintf("dedup  %d in-flight", snap.DedupWaiters)))
	}

	return strings.Join(lines, "\n")
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
