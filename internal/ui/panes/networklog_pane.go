package panes

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// latencyBarMax is the response time (ms) at which the latency bar is full (10 bars).
const latencyBarMax = 200

// latencyBarSteps is the number of █ chars in a full latency bar.
const latencyBarSteps = 10

// NetworkLogPane displays a scrollable, filterable reverse-chronological log of
// all API requests stored in state.Store.NetLogEntries(). It does NOT import api/.
type NetworkLogPane struct {
	store   *state.Store
	theme   theme.Theme
	table   *components.Table
	filter  *components.Filter
	focused bool
	width   int
	height  int
}

// Compile-time check: NetworkLogPane implements layout.Pane.
var _ layout.Pane = &NetworkLogPane{}

// NewNetworkLogPane creates a NetworkLogPane with the given store and theme.
func NewNetworkLogPane(s *state.Store, th theme.Theme) *NetworkLogPane {
	columns := []components.ColumnDef{
		{Key: "time", Header: "TIME", FlexFactor: 3, Color: th.TextMuted()},
		{Key: "method", Header: "METHOD", FlexFactor: 2, Color: th.TextSecondary()},
		{Key: "endpoint", Header: "ENDPOINT", FlexFactor: 8, Color: th.TextPrimary()},
		{Key: "status", Header: "STATUS", FlexFactor: 2, Color: th.TextPrimary()},
		{Key: "latency", Header: "LATENCY", FlexFactor: 2, Color: th.TextMuted()},
		{Key: "notes", Header: "NOTES", FlexFactor: 4, Color: th.TextMuted()},
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

// SelectedIndex returns the current table cursor row (0-based).
// Exported for testing.
func (p *NetworkLogPane) SelectedIndex() int { return p.table.SelectedIndex() }

// Init returns nil — the NetworkLogPane reacts to TickMsg from the app.
func (p *NetworkLogPane) Init() tea.Cmd { return nil }

// Update handles key events and TickMsg to refresh the log.
func (p *NetworkLogPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case TickMsg:
		// Refresh log entries from store on each 1s tick.
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
		p.refreshRows()
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

// refreshRows re-reads the store's net log and rebuilds the table rows.
// Entries are sorted newest-first before display.
func (p *NetworkLogPane) refreshRows() {
	entries := p.store.NetLogEntries()

	// Sort newest-first (Entries() returns oldest-first).
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	query := p.filter.Query()
	rows := make([]map[string]string, 0, len(entries))
	for _, e := range entries {
		statusStr := strconv.Itoa(e.StatusCode)
		latencyStr := fmt.Sprintf("%dms", e.DurationMs)

		// Apply filter: match on endpoint path or status code.
		if query != "" {
			if !p.filter.MatchesAny(e.Path, statusStr) {
				continue
			}
		}

		notes := latencyBar(e.DurationMs)
		if e.StatusCode == 429 {
			notes += " ⚠"
		}

		rows = append(rows, map[string]string{
			"time":     e.Timestamp.Format("15:04:05"),
			"method":   e.Method,
			"endpoint": e.Path,
			"status":   statusStr,
			"latency":  latencyStr,
			"notes":    notes,
		})
	}
	p.table.SetRows(rows)
}

// latencyBar returns a string of █ characters proportional to durationMs.
// Full bar (10 chars) at latencyBarMax ms.
func latencyBar(durationMs int64) string {
	if durationMs <= 0 {
		return "█"
	}
	bars := int(durationMs * latencyBarSteps / latencyBarMax)
	if bars > latencyBarSteps {
		bars = latencyBarSteps
	}
	if bars < 1 {
		bars = 1
	}
	return strings.Repeat("█", bars)
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
