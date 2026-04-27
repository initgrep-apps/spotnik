package panes

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// maxGatewayLiveRows is the maximum number of events retained in the buffer.
const maxGatewayLiveRows = 500

// Compile-time interface checks.
var _ layout.Pane = &GatewayLivePane{}
var _ layout.FilterablePane = &GatewayLivePane{}
var _ layout.FilterQueryPane = &GatewayLivePane{}

// gatewayLiveRow holds the pre-built display data for a single gateway event.
type gatewayLiveRow struct {
	glyphRole   uikit.GlyphRole
	intent      uikit.Role
	label       string // "HH:MM:SS  <event description>"
	matchString string // pre-built for filter matching
}

// GatewayLivePane displays a 500-entry reverse-chronological gateway event stream
// that is scrollable and filterable. New events are prepended on each tick.
// A committed filter (Enter-to-apply) narrows visible rows; the query appears in
// the pane border via layout.FilterQueryPane.
type GatewayLivePane struct {
	store       state.StateReader
	theme       theme.Theme
	focused     bool
	width       int
	height      int
	eventCursor uint64
	buffer      []gatewayLiveRow // newest-first; capped at maxGatewayLiveRows
	table       *components.Table
	filter      *components.Filter
	activeQuery string // committed filter (Enter-to-apply)
}

// NewGatewayLivePane creates a GatewayLivePane with the given store and theme.
// The table has a single column ("row") with no header.
func NewGatewayLivePane(s state.StateReader, th theme.Theme) *GatewayLivePane {
	columns := []components.ColumnDef{
		// Empty color so the column applies no foreground — ListRow.Render() supplies
		// its own ANSI intent colours, which must not be overridden by the column style.
		{Key: "row", Header: "", FlexFactor: 1, Color: ""},
	}
	t := components.NewTable(components.TableConfig{
		Columns:      columns,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   false,
	})
	t.SetFocused(false)

	return &GatewayLivePane{
		store:  s,
		theme:  th,
		table:  t,
		filter: components.NewFilter(th),
	}
}

// ID returns the PaneGatewayLive identifier.
func (p *GatewayLivePane) ID() layout.PaneID { return layout.PaneGatewayLive }

// Title returns the display title shown in the pane border.
func (p *GatewayLivePane) Title() string { return "Gateway Live" }

// ToggleKey returns 4 — the Page B toggle key for this pane.
func (p *GatewayLivePane) ToggleKey() int { return 4 }

// Actions returns shortcut hints based on filter state.
func (p *GatewayLivePane) Actions() []layout.Action {
	if p.filter.IsActive() {
		return []layout.Action{{Key: "Esc", Label: "cancel"}}
	}
	return []layout.Action{{Key: "f", Label: "filter"}}
}

// IsFocused returns whether this pane has keyboard focus.
func (p *GatewayLivePane) IsFocused() bool { return p.focused }

// SetFocused updates the keyboard focus state.
func (p *GatewayLivePane) SetFocused(f bool) {
	p.focused = f
	p.table.SetFocused(f && !p.filter.IsActive())
}

// Init returns nil — GatewayLivePane reacts to TickMsg from the app.
func (p *GatewayLivePane) Init() tea.Cmd { return nil }

// SetSize updates the content area dimensions.
func (p *GatewayLivePane) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.filter.SetWidth(w)
	p.resizeTable()
}

// SetTheme updates the theme reference and rebuilds the table with new column colors.
func (p *GatewayLivePane) SetTheme(th theme.Theme) {
	p.theme = th
	p.filter = components.NewFilter(th)
	columns := []components.ColumnDef{
		{Key: "row", Header: "", FlexFactor: 1, Color: ""},
	}
	t := components.NewTable(components.TableConfig{
		Columns:      columns,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   false,
	})
	t.SetFocused(p.focused && !p.filter.IsActive())
	p.table = t
	p.resizeTable()
	p.buildTableRows()
}

// HasActiveFilter returns true when the in-pane filter input is capturing keystrokes.
// Satisfies layout.FilterablePane.
func (p *GatewayLivePane) HasActiveFilter() bool { return p.filter.IsActive() }

// ActiveFilterQuery returns the committed filter query for border display.
// Satisfies layout.FilterQueryPane.
func (p *GatewayLivePane) ActiveFilterQuery() string { return p.activeQuery }

// BufferedEventCount returns the number of events in the buffer.
// Exported for testing.
func (p *GatewayLivePane) BufferedEventCount() int { return len(p.buffer) }

// TableCurrentPage returns the current page number (1-indexed) of the inner table.
// Exported for testing the Esc scroll-reset behaviour.
func (p *GatewayLivePane) TableCurrentPage() int { return p.table.CurrentPage() }

// Update handles key events and TickMsg to refresh the live event stream.
func (p *GatewayLivePane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case TickMsg:
		p.drainEvents()
		return p, nil
	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}
		return p.handleKey(m)
	}
	return p, nil
}

// handleKey routes key events for the gateway live pane.
func (p *GatewayLivePane) handleKey(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	// When filter is active, handle Enter/Esc at pane level; forward others to filter.
	if p.filter.IsActive() {
		switch m.Type {
		case tea.KeyEnter:
			// Commit: read query BEFORE Toggle() which clears it.
			p.activeQuery = p.filter.Query()
			p.filter.Toggle() // deactivates and clears filter input
			p.table.SetFocused(true)
			p.resizeTable()
			p.buildTableRows()
			return p, nil
		case tea.KeyEscape:
			// Cancel: deactivate filter without committing.
			p.filter.Toggle()
			p.table.SetFocused(true)
			p.resizeTable()
			p.buildTableRows()
			return p, nil
		default:
			cmd := p.filter.Update(m)
			p.buildTableRows()
			return p, cmd
		}
	}

	// 'f' activates the filter.
	if m.Type == tea.KeyRunes && string(m.Runes) == "f" {
		p.filter.Toggle()
		p.table.SetFocused(false)
		p.resizeTable()
		return p, nil
	}

	// Esc when filter is not open:
	// Mode 1: committed query exists → clear it.
	// Mode 2: no committed query → reset scroll to page 1.
	if m.Type == tea.KeyEscape {
		if p.activeQuery != "" {
			p.activeQuery = ""
			p.buildTableRows()
			return p, nil
		}
		p.table.GotoTop()
		return p, nil
	}

	// Forward navigation keys to the table.
	cmd := p.table.Update(m)
	return p, cmd
}

// View renders the gateway live pane. Pure — no side effects.
func (p *GatewayLivePane) View() string {
	if p.width == 0 || p.height == 0 {
		return ""
	}
	var parts []string
	if p.filter.IsActive() {
		parts = append(parts, p.filter.View(p.width))
	}
	parts = append(parts, p.table.View())
	return strings.Join(parts, "\n")
}

// drainEvents reads new gateway events from the store, converts them to rows
// (newest-first), prepends to the buffer, and trims to maxGatewayLiveRows.
func (p *GatewayLivePane) drainEvents() {
	if p.store == nil {
		return
	}
	newCursor, events := p.store.ReadEventsFrom(p.eventCursor)
	p.eventCursor = newCursor

	if len(events) == 0 {
		return
	}

	// Build rows from new events; events arrive chronologically (oldest first).
	newRows := make([]gatewayLiveRow, 0, len(events))
	for _, e := range events {
		row, ok := buildGatewayLiveRow(e)
		if !ok {
			continue
		}
		newRows = append(newRows, row)
	}

	if len(newRows) == 0 {
		// All new events were skipped (e.g. all EventBackoffExpired); table unchanged.
		return
	}

	// Reverse so newest event is first.
	for i, j := 0, len(newRows)-1; i < j; i, j = i+1, j-1 {
		newRows[i], newRows[j] = newRows[j], newRows[i]
	}

	// Prepend to buffer (newest at front).
	p.buffer = append(newRows, p.buffer...)

	// Trim to cap.
	if len(p.buffer) > maxGatewayLiveRows {
		p.buffer = p.buffer[:maxGatewayLiveRows]
	}

	p.buildTableRows()
}

// buildGatewayLiveRow converts a single GatewayEvent into a gatewayLiveRow.
// Returns (zero, false) for events that should not be displayed (EventBackoffExpired).
func buildGatewayLiveRow(e domain.GatewayEvent) (gatewayLiveRow, bool) {
	ts := e.Timestamp.Format("15:04:05")
	path := strings.TrimPrefix(e.Path, "/v1/me")

	var role uikit.GlyphRole
	var intent uikit.Role
	var label string

	switch e.Kind {
	case domain.EventRequestEntered:
		if e.Priority == domain.PriorityInteractive {
			role = uikit.GlyphRunning
			intent = uikit.RolePlain
		} else {
			role = uikit.GlyphDeadline
			intent = uikit.RoleMuted
		}
		label = fmt.Sprintf("%s  %s %s", ts, e.Method, path)

	case domain.EventTokenConsumed:
		role = uikit.GlyphWarning
		intent = uikit.RoleWarning
		snap := e.Snapshot
		label = fmt.Sprintf("%s  Token consumed → %d", ts, snap.TokensAvailable)

	case domain.EventTokenRefilled:
		role = uikit.GlyphRepeatAll
		intent = uikit.RoleSuccess
		snap := e.Snapshot
		label = fmt.Sprintf("%s  Tokens refilled → %d", ts, snap.TokensAvailable)

	case domain.EventSemaphoreAcquired:
		role = uikit.GlyphFilledSquare
		intent = uikit.RoleInfo
		snap := e.Snapshot
		label = fmt.Sprintf("%s  Semaphore acquired  %d/%d", ts, snap.ConcurrentActive, snap.ConcurrentMax)

	case domain.EventSemaphoreReleased:
		role = uikit.GlyphEmptySquare
		intent = uikit.RoleMuted
		snap := e.Snapshot
		label = fmt.Sprintf("%s  Semaphore released  %d/%d", ts, snap.ConcurrentActive, snap.ConcurrentMax)

	case domain.EventRequestAllowed:
		role = uikit.GlyphSuccess
		intent = uikit.RoleSuccess
		label = fmt.Sprintf("%s  %s %s  allowed", ts, e.Method, path)

	case domain.EventRequestBlocked:
		role = uikit.GlyphError
		intent = uikit.RoleError
		label = fmt.Sprintf("%s  %s %s  blocked", ts, e.Method, path)

	case domain.EventDedupJoined:
		role = uikit.GlyphRateLimit
		intent = uikit.RoleInfo
		label = fmt.Sprintf("%s  %s %s  dedup joined", ts, e.Method, path)

	case domain.EventDedupResolved:
		role = uikit.GlyphSuccess
		intent = uikit.RoleSuccess
		snap := e.Snapshot
		label = fmt.Sprintf("%s  Dedup resolved  %d", ts, snap.DedupWaiters)

	case domain.EventBackoffStarted:
		role = uikit.GlyphBlocked
		intent = uikit.RoleError
		snap := e.Snapshot
		label = fmt.Sprintf("%s  Backoff started  (retry in %.1fs)", ts, snap.BackoffRemaining)

	case domain.EventHttpCompleted:
		role = uikit.GlyphSuccess
		intent = uikit.RoleSuccess
		label = fmt.Sprintf("%s  %d  %dms", ts, e.StatusCode, e.DurationMs)

	case domain.EventBackoffExpired:
		// Silently skip — not displayed in the live stream.
		return gatewayLiveRow{}, false

	default:
		return gatewayLiveRow{}, false
	}

	return gatewayLiveRow{
		glyphRole:   role,
		intent:      intent,
		label:       label,
		matchString: strings.ToLower(label),
	}, true
}

// buildTableRows converts the buffer to table rows, applying the committed filter.
// No-ops when width is zero to avoid rendering rows with negative label width.
func (p *GatewayLivePane) buildTableRows() {
	if p.width == 0 {
		return
	}
	query := strings.ToLower(p.activeQuery)
	rows := make([]map[string]string, 0, len(p.buffer))

	for _, row := range p.buffer {
		if query != "" && !strings.Contains(row.matchString, query) {
			continue
		}
		rendered := uikit.ListRow{
			Glyph:  row.glyphRole,
			Label:  row.label,
			Intent: row.intent,
			Theme:  p.theme,
		}.Render(p.width - 2)
		rows = append(rows, map[string]string{"row": rendered})
	}

	p.table.SetRows(rows)
}

// resizeTable adjusts the table height to account for the filter bar when active.
func (p *GatewayLivePane) resizeTable() {
	h := p.height
	if p.filter.IsActive() {
		h--
	}
	if h < 0 {
		h = 0
	}
	p.table.SetSize(p.width, h)
}
