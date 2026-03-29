package panes

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// maxNetworkLogRows is the maximum number of completed request rows to retain.
const maxNetworkLogRows = 200

// networkLogRow holds the display data extracted from gateway events for a single
// completed or blocked request.
type networkLogRow struct {
	timestamp  time.Time
	method     string
	path       string
	statusCode int
	durationMs int64
	priority   domain.RequestPriority
	decision   domain.EventKind
}

// NetworkLogPane displays a scrollable, filterable reverse-chronological log of
// all API requests read from the GatewayEventLog via cursor-based reads.
// It does NOT import api/.
type NetworkLogPane struct {
	store             *state.Store
	theme             theme.Theme
	table             *components.Table
	filter            *components.Filter
	focused           bool
	width             int
	height            int
	eventCursor       uint64
	completedRequests []networkLogRow
}

// Compile-time check: NetworkLogPane implements layout.Pane.
var _ layout.Pane = &NetworkLogPane{}

// NewNetworkLogPane creates a NetworkLogPane with the given store and theme.
func NewNetworkLogPane(s *state.Store, th theme.Theme) *NetworkLogPane {
	columns := []components.ColumnDef{
		{Key: "time", Header: "TIME", FlexFactor: 3, Color: th.TextMuted()},
		{Key: "method", Header: "METHOD", FlexFactor: 2, Color: th.TextSecondary()},
		{Key: "endpoint", Header: "ENDPOINT", FlexFactor: 7, Color: th.TextPrimary()},
		{Key: "status", Header: "STATUS", FlexFactor: 2, Color: th.TextPrimary()},
		{Key: "latency", Header: "LATENCY", FlexFactor: 2, Color: th.TextMuted()},
		{Key: "priority", Header: "PRIORITY", FlexFactor: 3, Color: th.TextMuted()},
		{Key: "decision", Header: "DECISION", FlexFactor: 3, Color: th.TextSecondary()},
	}

	t := components.NewTable(components.TableConfig{
		Columns:      columns,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   true,
	})

	p := &NetworkLogPane{
		store:  s,
		theme:  th,
		table:  t,
		filter: components.NewFilter(th),
	}
	t.SetFocused(false)
	p.refreshRows()
	return p
}

// ID returns the PaneNetworkLog identifier.
func (p *NetworkLogPane) ID() layout.PaneID { return layout.PaneNetworkLog }

// Title returns the display title shown in the pane border.
func (p *NetworkLogPane) Title() string { return "Network Log" }

// ToggleKey returns 0 — Page B panes are not individually toggleable.
func (p *NetworkLogPane) ToggleKey() int { return 0 }

// Actions returns shortcut hints based on filter state.
func (p *NetworkLogPane) Actions() []layout.Action {
	if p.filter.IsActive() {
		return []layout.Action{{Key: "Esc", Label: "close"}}
	}
	return []layout.Action{
		{Key: "f", Label: "filter"},
		{Key: "j/k", Label: "scroll"},
	}
}

// SetSize updates the content area dimensions.
func (p *NetworkLogPane) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.filter.SetWidth(width)
	p.resizeTable()
}

// SetFocused updates the keyboard focus state.
func (p *NetworkLogPane) SetFocused(focused bool) {
	p.focused = focused
	p.table.SetFocused(focused && !p.filter.IsActive())
}

// IsFocused returns whether this pane has keyboard focus.
func (p *NetworkLogPane) IsFocused() bool { return p.focused }

// HasActiveFilter returns true when the in-pane filter is capturing keystrokes.
func (p *NetworkLogPane) HasActiveFilter() bool { return p.filter.IsActive() }

// SelectedIndex returns the current table cursor row (0-based).
// Exported for testing.
func (p *NetworkLogPane) SelectedIndex() int { return p.table.SelectedIndex() }

// EventCursor returns the current event cursor position.
// Exported for testing.
func (p *NetworkLogPane) EventCursor() uint64 { return p.eventCursor }

// CompletedRequestsLen returns the number of completed request rows currently buffered.
// Exported for testing.
func (p *NetworkLogPane) CompletedRequestsLen() int { return len(p.completedRequests) }

// Init returns nil — the NetworkLogPane reacts to TickMsg from the app.
func (p *NetworkLogPane) Init() tea.Cmd { return nil }

// Update handles key events and TickMsg to refresh the log.
func (p *NetworkLogPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case TickMsg:
		// Refresh log entries from the event journal on each 1s tick.
		p.refreshRows()
		return p, nil

	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}
		return p.handleKey(m)
	}
	return p, nil
}

// handleKey routes key events for the network log pane.
func (p *NetworkLogPane) handleKey(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	// When filter is active, forward all keys to the filter.
	if p.filter.IsActive() {
		cmd := p.filter.Update(m)
		if !p.filter.IsActive() {
			p.table.SetFocused(true)
			p.resizeTable()
		}
		p.buildTableRows()
		return p, cmd
	}

	// 'f' activates the filter.
	if m.Type == tea.KeyRunes && string(m.Runes) == "f" {
		p.filter.Toggle()
		p.table.SetFocused(false)
		p.resizeTable()
		return p, nil
	}

	// Forward j/k and other navigation to the table.
	cmd := p.table.Update(m)
	return p, cmd
}

// View renders the network log pane. Pure — no side effects.
func (p *NetworkLogPane) View() string {
	var parts []string
	if p.filter.IsActive() {
		parts = append(parts, p.filter.View(p.width))
	}
	parts = append(parts, p.table.View())
	return strings.Join(parts, "\n")
}

// refreshRows drains new events from the GatewayEventLog since the last cursor
// position, extracts completed and blocked requests, and rebuilds the table.
func (p *NetworkLogPane) refreshRows() {
	// Drain new events since last cursor.
	newCursor, events := p.store.ReadEventsFrom(p.eventCursor)
	p.eventCursor = newCursor

	// Build a map of RequestID → decision kind for lookups.
	// Decision events (allowed/blocked/waited/dedupJoined) precede EventHttpCompleted
	// in the event stream for the same request.
	decisions := make(map[uint64]domain.EventKind)
	for _, e := range events {
		switch e.Kind {
		case domain.EventRequestAllowed, domain.EventRequestBlocked,
			domain.EventRequestWaited, domain.EventDedupJoined:
			decisions[e.RequestID] = e.Kind
		}
	}

	for _, e := range events {
		switch e.Kind {
		case domain.EventHttpCompleted:
			row := networkLogRow{
				timestamp:  e.Timestamp,
				method:     e.Method,
				path:       e.Path,
				statusCode: e.StatusCode,
				durationMs: e.DurationMs,
				priority:   e.Priority,
				decision:   decisions[e.RequestID],
			}
			p.completedRequests = append(p.completedRequests, row)

		case domain.EventRequestBlocked:
			// Blocked requests never reach HTTP — show them with status 0.
			row := networkLogRow{
				timestamp:  e.Timestamp,
				method:     e.Method,
				path:       e.Path,
				statusCode: 0,
				durationMs: 0,
				priority:   e.Priority,
				decision:   domain.EventRequestBlocked,
			}
			p.completedRequests = append(p.completedRequests, row)
		}
	}

	// Cap at max entries (newest kept).
	if len(p.completedRequests) > maxNetworkLogRows {
		p.completedRequests = p.completedRequests[len(p.completedRequests)-maxNetworkLogRows:]
	}

	p.buildTableRows()
}

// buildTableRows iterates completedRequests in reverse (newest-first), applies
// the filter, and sets the table rows.
func (p *NetworkLogPane) buildTableRows() {
	query := p.filter.Query()
	rows := make([]map[string]string, 0, len(p.completedRequests))

	// Iterate newest-first.
	for i := len(p.completedRequests) - 1; i >= 0; i-- {
		row := p.completedRequests[i]

		statusStr := strconv.Itoa(row.statusCode)
		latencyStr := fmt.Sprintf("%dms", row.durationMs)

		pri := "bg"
		if row.priority == domain.PriorityInteractive {
			pri = "int"
		}

		dec := ""
		switch row.decision {
		case domain.EventRequestAllowed:
			dec = "allowed"
		case domain.EventRequestBlocked:
			dec = "blocked"
		case domain.EventRequestWaited:
			dec = "waited"
		case domain.EventDedupJoined:
			dec = "dedup"
		}

		// Apply filter: match on endpoint path, status code, priority, or decision.
		if query != "" {
			if !p.filter.MatchesAny(row.path, statusStr, pri, dec) {
				continue
			}
		}

		rows = append(rows, map[string]string{
			"time":     row.timestamp.Format("15:04:05"),
			"method":   row.method,
			"endpoint": row.path,
			"status":   statusStr,
			"latency":  latencyStr,
			"priority": pri,
			"decision": dec,
		})
	}
	p.table.SetRows(rows)
}

// resizeTable adjusts table height to account for the filter bar when active.
func (p *NetworkLogPane) resizeTable() {
	h := p.height
	if p.filter.IsActive() {
		h--
	}
	if h < 0 {
		h = 0
	}
	p.table.SetSize(p.width, h)
}
