package panes

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// NetLogView renders a scrollable table of API call logs from the Store's NetLog.
// It is embedded inside StatsView as the 4th section.
type NetLogView struct {
	store  *state.Store
	theme  theme.Theme
	scroll int // scroll offset (0 = show newest at top)
	width  int
	height int
}

// NewNetLogView creates a NetLogView that reads from the given store.
func NewNetLogView(store *state.Store, t theme.Theme) *NetLogView {
	return &NetLogView{
		store: store,
		theme: t,
	}
}

// SetSize updates the rendering dimensions.
func (v *NetLogView) SetSize(w, h int) {
	v.width = w
	v.height = h
}

// ScrollDown moves the scroll position down (shows older entries).
func (v *NetLogView) ScrollDown() {
	entries := v.store.NetLogEntries()
	maxScroll := len(entries) - v.visibleRows()
	if maxScroll < 0 {
		maxScroll = 0
	}
	if v.scroll < maxScroll {
		v.scroll++
	}
}

// ScrollUp moves the scroll position up (shows newer entries).
func (v *NetLogView) ScrollUp() {
	if v.scroll > 0 {
		v.scroll--
	}
}

// Scroll returns the current scroll offset (exported for testing).
func (v *NetLogView) Scroll() int {
	return v.scroll
}

// visibleRows returns how many log rows fit in the available height.
// Reserves 1 row for the header.
func (v *NetLogView) visibleRows() int {
	rows := v.height - 1
	if rows < 1 {
		return 1
	}
	return rows
}

// View renders the network log table.
func (v *NetLogView) View() string {
	entries := v.store.NetLogEntries()

	var sb strings.Builder

	// Header row.
	header := fmt.Sprintf("  %-8s %-6s %-36s %6s %8s",
		"TIME", "METHOD", "PATH", "STATUS", "DURATION")
	sb.WriteString(lipgloss.NewStyle().
		Foreground(v.theme.SectionHeader()).
		Bold(true).
		Render(header))
	sb.WriteString("\n")

	if len(entries) == 0 {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(v.theme.TextMuted()).
			Render("  No API calls recorded yet"))
		return sb.String()
	}

	// Show entries newest-first.
	visible := v.visibleRows()
	// Reverse entries for newest-first display.
	reversed := make([]state.NetLogEntry, len(entries))
	for i, e := range entries {
		reversed[len(entries)-1-i] = e
	}

	// Apply scroll offset.
	start := v.scroll
	if start >= len(reversed) {
		start = len(reversed) - 1
	}
	end := start + visible
	if end > len(reversed) {
		end = len(reversed)
	}

	for _, entry := range reversed[start:end] {
		timeStr := entry.Timestamp.Format("15:04:05")
		row := fmt.Sprintf("  %-8s %-6s %-36s %6d %6dms",
			timeStr, entry.Method, truncate(entry.Path, 36),
			entry.StatusCode, entry.DurationMs)

		style := lipgloss.NewStyle()
		if entry.StatusCode >= 400 {
			style = style.Foreground(v.theme.Error())
		} else if entry.StatusCode >= 200 && entry.StatusCode < 300 {
			style = style.Foreground(v.theme.PlayingIndicator())
		} else {
			style = style.Foreground(v.theme.TextPrimary())
		}

		sb.WriteString(style.Render(row))
		sb.WriteString("\n")
	}

	return sb.String()
}
