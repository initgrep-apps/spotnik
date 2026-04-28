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
	requestID  uint64
	timestamp  time.Time
	method     string
	path       string
	statusCode int
	durationMs int64
	priority   domain.RequestPriority
	// decision is zero when EventHttpCompleted arrives before EventRequestAllowed
	// (production order). The backfill sweep in refreshRows updates it once the
	// decision event arrives on a later tick.
	decision domain.EventKind
}

// NetworkLogPane displays a scrollable, filterable reverse-chronological log of
// all API requests read from the GatewayEventLog via cursor-based reads.
// It does NOT import api/.
type NetworkLogPane struct {
	*TableBasedPane

	eventCursor       uint64
	completedRequests []networkLogRow
	// pendingDecisions accumulates decision events (allowed/blocked/dedupJoined)
	// across ticks so the Decision column is correctly populated when
	// EventHttpCompleted arrives on a later tick than the decision event.
	// Entries are consumed (and deleted) when EventHttpCompleted is processed.
	pendingDecisions map[uint64]domain.EventKind
}

// Compile-time check: NetworkLogPane implements layout.Pane.
var _ layout.Pane = &NetworkLogPane{}

// Compile-time check: NetworkLogPane implements layout.FilterQueryPane.
var _ layout.FilterQueryPane = &NetworkLogPane{}

// NewNetworkLogPane creates a NetworkLogPane with the given store and theme.
func NewNetworkLogPane(s state.StateReader, th theme.Theme) *NetworkLogPane {
	columns := []components.ColumnDef{
		{Key: "time", Header: "Time", FlexFactor: 3, Color: th.ColumnIndex()},
		{Key: "method", Header: "Method", FlexFactor: 2, Color: th.ColumnSecondary()},
		{Key: "endpoint", Header: "Endpoint", FlexFactor: 7, Color: th.ColumnPrimary()},
		{Key: "status", Header: "Status", FlexFactor: 2, Color: th.ColumnTertiary()},
		{Key: "latency", Header: "Latency", FlexFactor: 2, Color: th.ColumnTertiary()},
		{Key: "priority", Header: "Priority", FlexFactor: 3, Color: th.ColumnIndex()},
		{Key: "decision", Header: "Decision", FlexFactor: 3, Color: th.ColumnSecondary()},
	}

	t := components.NewTable(components.TableConfig{
		Columns:      columns,
		Theme:        th,
		PlayingIndex: -1,
		ShowHeader:   true,
	})

	p := &NetworkLogPane{
		TableBasedPane:   NewTableBasedPane(s, th, false, t, components.NewFilter(th)),
		pendingDecisions: make(map[uint64]domain.EventKind),
	}
	t.SetFocused(false)
	p.refreshRows()
	return p
}

// ID returns the PaneNetworkLog identifier.
func (p *NetworkLogPane) ID() layout.PaneID { return layout.PaneNetworkLog }

// Title returns the display title shown in the pane border.
func (p *NetworkLogPane) Title() string { return "Network Log" }

// ToggleKey returns 5 — the Page B toggle key for the Network Log pane.
func (p *NetworkLogPane) ToggleKey() int { return 5 }

// SetSize updates the content area dimensions.
func (p *NetworkLogPane) SetSize(width, height int) {
	p.TableBasedPane.SetSize(width, height)
	p.Filter().SetWidth(width)
	p.resizeTable()
}

// SetFocused updates the keyboard focus state.
func (p *NetworkLogPane) SetFocused(focused bool) {
	p.TableBasedPane.SetFocused(focused)
	p.Table().SetFocused(focused && !p.Filter().IsActive())
}

// SelectedIndex returns the current table cursor row (0-based).
// Exported for testing.
func (p *NetworkLogPane) SelectedIndex() int { return p.Table().SelectedIndex() }

// TableCurrentPage returns the current page number (1-indexed) of the inner table.
// Exported for testing the Esc scroll-reset behaviour (story 173).
func (p *NetworkLogPane) TableCurrentPage() int { return p.Table().CurrentPage() }

// EventCursor returns the current event cursor position.
// Exported for testing.
func (p *NetworkLogPane) EventCursor() uint64 { return p.eventCursor }

// CompletedRequestsLen returns the number of completed request rows currently buffered.
// Exported for testing.
func (p *NetworkLogPane) CompletedRequestsLen() int { return len(p.completedRequests) }

// PendingDecisionsLen returns the number of entries in the pendingDecisions map.
// Exported for testing to verify the map stays bounded.
func (p *NetworkLogPane) PendingDecisionsLen() int { return len(p.pendingDecisions) }

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
		// Delegate filter-key routing to the base.
		if consumed, cmd := p.HandleFilterKey(m, p.buildTableRows, p.resizeTable); consumed {
			return p, cmd
		}
		// Forward j/k and other navigation to the table.
		cmd := p.Table().Update(m)
		return p, cmd
	}
	return p, nil
}

// View renders the network log pane. Pure — no side effects.
func (p *NetworkLogPane) View() string {
	var parts []string
	if p.Filter().IsActive() {
		parts = append(parts, p.Filter().View(p.width))
	}
	parts = append(parts, p.Table().View())
	return strings.Join(parts, "\n")
}

// refreshRows drains new events from the GatewayEventLog since the last cursor
// position, extracts completed and blocked requests, and rebuilds the table.
//
// Gateway emission order for primary callers: EventHttpCompleted fires FIRST,
// then EventRequestAllowed fires. This means EventHttpCompleted can arrive on
// the same tick before EventRequestAllowed is in the map. To handle this,
// completed rows store their requestID and a backfill sweep at the end of each
// tick fills in any decision that arrived in the same batch but was processed
// after the row was created.
//
// Dedup waiters emit EventDedupJoined then EventDedupResolved; they never emit
// EventHttpCompleted. Clearing the pendingDecisions entry on EventDedupResolved
// prevents unbounded map growth for waiter request IDs.
func (p *NetworkLogPane) refreshRows() {
	// Drain new events since last cursor.
	newCursor, events := p.store.ReadEventsFrom(p.eventCursor)
	p.eventCursor = newCursor

	// First pass: accumulate decision events into the persistent map. Decision events
	// (allowed/blocked/dedupJoined) may arrive on a different tick than the
	// corresponding EventHttpCompleted, so we keep pendingDecisions across ticks.
	// EventDedupResolved clears the waiter's entry — dedup waiters never emit
	// EventHttpCompleted, so without this the map would grow without bound.
	for _, e := range events {
		switch e.Kind {
		case domain.EventRequestAllowed, domain.EventRequestBlocked,
			domain.EventDedupJoined:
			p.pendingDecisions[e.RequestID] = e.Kind
		case domain.EventDedupResolved:
			// Waiter's lifecycle is complete — clear its pendingDecisions entry.
			delete(p.pendingDecisions, e.RequestID)
		}
	}

	// Second pass: build rows for completed and blocked events.
	for _, e := range events {
		switch e.Kind {
		case domain.EventHttpCompleted:
			// Consume and delete the decision to prevent unbounded map growth.
			// If EventRequestAllowed has not arrived yet (production order: HttpCompleted
			// first, then Allowed), decision will be zero and the backfill sweep below
			// will fill it in once the Allowed event is processed in this same batch.
			decision := p.pendingDecisions[e.RequestID]
			delete(p.pendingDecisions, e.RequestID)

			row := networkLogRow{
				requestID:  e.RequestID,
				timestamp:  e.Timestamp,
				method:     e.Method,
				path:       e.Path,
				statusCode: e.StatusCode,
				durationMs: e.DurationMs,
				priority:   e.Priority,
				decision:   decision,
			}
			p.completedRequests = append(p.completedRequests, row)

		case domain.EventRequestBlocked:
			// Blocked requests never reach HTTP — show them with status 0.
			// The decision is already known (EventRequestBlocked implies blocked).
			delete(p.pendingDecisions, e.RequestID)
			row := networkLogRow{
				requestID:  e.RequestID,
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

	// Backfill sweep: when EventHttpCompleted arrived before EventRequestAllowed in
	// the same batch (production order), the row was created with decision==0. Now
	// that all decision events have been added to pendingDecisions, fill in any rows
	// that still have a zero decision. This covers the same-tick (in-batch) ordering
	// case where Allowed is processed in the first pass after the row was created in
	// the second pass. We also handle cross-tick backfill: if a row carries a zero
	// decision from a previous tick and its Allowed event arrives now, fill it in.
	for i := range p.completedRequests {
		if p.completedRequests[i].decision == 0 {
			if d, ok := p.pendingDecisions[p.completedRequests[i].requestID]; ok {
				p.completedRequests[i].decision = d
				delete(p.pendingDecisions, p.completedRequests[i].requestID)
			}
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
	query := p.Filter().Query()
	rows := make([]map[string]string, 0, len(p.completedRequests))

	// Iterate newest-first.
	for i := len(p.completedRequests) - 1; i >= 0; i-- {
		row := p.completedRequests[i]

		statusStr := strconv.Itoa(row.statusCode)
		latencyStr := fmt.Sprintf("%dms", row.durationMs)

		pri := "◷ background"
		if row.priority == domain.PriorityInteractive {
			pri = "⚡ interactive"
		}

		dec := ""
		switch row.decision {
		case domain.EventRequestAllowed:
			dec = "allowed"
		case domain.EventRequestBlocked:
			dec = "blocked"
		case domain.EventDedupJoined:
			dec = "dedup"
		}

		// Apply filter: match on endpoint path, status code, priority, or decision.
		if query != "" {
			if !p.Filter().MatchesAny(row.path, statusStr, pri, dec) {
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
	p.Table().SetRows(rows)
}

// SetTheme updates the theme reference and rebuilds the table with new column colors.
// Called when the user switches themes at runtime.
// NOTE: rebuilding Filter resets the active state and query — existing behaviour.
func (p *NetworkLogPane) SetTheme(th theme.Theme) {
	p.theme = th
	columns := []components.ColumnDef{
		{Key: "time", Header: "Time", FlexFactor: 3, Color: th.ColumnIndex()},
		{Key: "method", Header: "Method", FlexFactor: 2, Color: th.ColumnSecondary()},
		{Key: "endpoint", Header: "Endpoint", FlexFactor: 7, Color: th.ColumnPrimary()},
		{Key: "status", Header: "Status", FlexFactor: 2, Color: th.ColumnTertiary()},
		{Key: "latency", Header: "Latency", FlexFactor: 2, Color: th.ColumnTertiary()},
		{Key: "priority", Header: "Priority", FlexFactor: 3, Color: th.ColumnIndex()},
		{Key: "decision", Header: "Decision", FlexFactor: 3, Color: th.ColumnSecondary()},
	}
	newTable, newFilter := components.RebuildTableTheme(th, columns, p.Table().Rows(), p.focused)
	p.SwapTableAndFilter(newTable, newFilter)
	p.resizeTable()
	p.buildTableRows()
}

// resizeTable adjusts table height to account for the filter bar when active.
func (p *NetworkLogPane) resizeTable() {
	h := p.height
	if p.Filter().IsActive() {
		h--
	}
	if h < 0 {
		h = 0
	}
	p.Table().SetSize(p.width, h)
}
