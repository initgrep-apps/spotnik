package panes

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	btable "github.com/evertras/bubble-table/table"
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
// A live filter (typing narrows rows immediately) is toggled with 'f'; the
// current query appears in the pane border via layout.FilterQueryPane.
type GatewayLivePane struct {
	*TableBasedPane

	eventCursor uint64
	buffer      []gatewayLiveRow // newest-first; capped at maxGatewayLiveRows
}

// NewGatewayLivePane creates a GatewayLivePane with the given store and theme.
// The table uses a two-column layout (glyph, event) with no header. Per-row
// glyph foreground is applied via btable.StyledCell so the row-level selection
// background is not interrupted by embedded ANSI resets.
func NewGatewayLivePane(s state.StateReader, th theme.Theme) *GatewayLivePane {
	columns := []components.ColumnDef{
		// FlexFactor 1:30 reserves a narrow glyph column (one Unicode rune plus
		// bubble-table column padding) and gives the remainder to the event text.
		// Color on the glyph column is a fallback; per-row StyledCell overrides it.
		{Key: "glyph", Header: "", FlexFactor: 1, Color: th.TextPrimary()},
		{Key: "event", Header: "", FlexFactor: 30, Color: th.ColumnPrimary()},
	}
	t := components.NewTable(components.TableConfig{
		Columns:      columns,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   false,
	})
	t.SetFocused(false)

	f := components.NewFilter(th)
	return &GatewayLivePane{
		TableBasedPane: NewTableBasedPane(s, th, false, t, f),
	}
}

// ID returns the PaneGatewayLive identifier.
func (p *GatewayLivePane) ID() layout.PaneID { return layout.PaneGatewayLive }

// Title returns the display title shown in the pane border.
func (p *GatewayLivePane) Title() string { return "Gateway Live" }

// ToggleKey returns 4 — the Stats page toggle key for this pane.
func (p *GatewayLivePane) ToggleKey() int { return 4 }

// SetFocused updates the keyboard focus state.
func (p *GatewayLivePane) SetFocused(f bool) {
	p.TableBasedPane.SetFocused(f)
	p.Table().SetFocused(f && !p.Filter().IsActive())
}

// Init returns nil — GatewayLivePane reacts to TickMsg from the app.
func (p *GatewayLivePane) Init() tea.Cmd { return nil }

// SetSize updates the content area dimensions.
func (p *GatewayLivePane) SetSize(w, h int) {
	p.TableBasedPane.SetSize(w, h)
	p.Filter().SetWidth(w)
	p.resizeTable()
}

// SetTheme updates the theme reference and rebuilds the table with new column colors.
func (p *GatewayLivePane) SetTheme(th theme.Theme) {
	p.theme = th
	newFilter := components.NewFilter(th)
	columns := []components.ColumnDef{
		{Key: "glyph", Header: "", FlexFactor: 1, Color: th.TextPrimary()},
		{Key: "event", Header: "", FlexFactor: 30, Color: th.ColumnPrimary()},
	}
	newTable := components.NewTable(components.TableConfig{
		Columns:      columns,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   false,
	})
	newTable.SetFocused(p.focused && !p.Filter().IsActive())
	p.SwapTableAndFilter(newTable, newFilter)
	p.resizeTable()
	p.buildTableRows()
}

// BufferedEventCount returns the number of events in the buffer.
// Exported for testing.
func (p *GatewayLivePane) BufferedEventCount() int { return len(p.buffer) }

// TableCurrentPage returns the current page number (1-indexed) of the inner table.
// Exported for testing the Esc scroll-reset behaviour.
func (p *GatewayLivePane) TableCurrentPage() int { return p.Table().CurrentPage() }

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
	// Delegate filter keys (f, Esc, and forwarding when active) to the shared handler.
	if consumed, cmd := p.HandleFilterKey(m, p.buildTableRows, p.resizeTable); consumed {
		return p, cmd
	}

	// Forward navigation keys to the table.
	cmd := p.Table().Update(m)
	return p, cmd
}

// View renders the gateway live pane. Pure — no side effects.
func (p *GatewayLivePane) View() string {
	if p.width == 0 || p.height == 0 {
		return ""
	}
	var parts []string
	if p.Filter().IsActive() {
		parts = append(parts, p.Filter().View(p.width))
	}
	parts = append(parts, p.Table().View())
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
		label = fmt.Sprintf("%s  Token consumed %s %d", ts, uikit.GlyphFor(uikit.GlyphArrowRight, uikit.ActiveMode()), snap.TokensAvailable)

	case domain.EventTokenRefilled:
		role = uikit.GlyphRepeatAll
		intent = uikit.RoleSuccess
		snap := e.Snapshot
		label = fmt.Sprintf("%s  Tokens refilled %s %d", ts, uikit.GlyphFor(uikit.GlyphArrowRight, uikit.ActiveMode()), snap.TokensAvailable)

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

// buildTableRows converts the buffer to rich table rows, applying the live filter
// query. Per-row glyph foreground is applied via btable.StyledCell so that
// bubble-table's row-level selection background (SelectedBg) is not interrupted
// by embedded ANSI resets. The event column receives a plain string.
//
// No-ops when width is zero to avoid rendering rows with negative label width.
func (p *GatewayLivePane) buildTableRows() {
	if p.width == 0 {
		return
	}
	query := strings.ToLower(p.Filter().Query())
	mode := uikit.ActiveMode()
	rows := make([]map[string]any, 0, len(p.buffer))

	for _, row := range p.buffer {
		if query != "" && !strings.Contains(row.matchString, query) {
			continue
		}
		glyphCell := btable.NewStyledCell(
			uikit.GlyphFor(row.glyphRole, mode),
			lipgloss.NewStyle().Foreground(uikit.ColourFor(row.intent, p.theme)),
		)
		rows = append(rows, map[string]any{
			"glyph": glyphCell,
			"event": row.label,
		})
	}

	p.Table().SetRichRows(rows)
}

// resizeTable adjusts the table height to account for the filter bar when active.
func (p *GatewayLivePane) resizeTable() {
	h := p.height
	if p.Filter().IsActive() {
		h--
	}
	if h < 0 {
		h = 0
	}
	p.Table().SetSize(p.width, h)
}
