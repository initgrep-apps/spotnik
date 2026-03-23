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
//
// Display order is oldest-at-top, newest-at-bottom (chronological).
// By default the view auto-scrolls to keep the newest entry visible.
// When the user scrolls up (k), auto-scroll disables so they can browse history.
// Scrolling back to the bottom re-enables auto-scroll.
type NetLogView struct {
	store  *state.Store
	theme  theme.Theme
	scroll int // index of the first visible entry (in chronological order)
	pinned bool // true = auto-scroll to newest; false = user has scrolled up
	width  int
	height int
}

// NewNetLogView creates a NetLogView that reads from the given store.
func NewNetLogView(store *state.Store, t theme.Theme) *NetLogView {
	return &NetLogView{
		store:  store,
		theme:  t,
		pinned: true,
	}
}

// SetSize updates the rendering dimensions.
func (v *NetLogView) SetSize(w, h int) {
	v.width = w
	v.height = h
}

// ScrollDown moves the viewport down (shows newer entries).
func (v *NetLogView) ScrollDown() {
	entries := v.store.NetLogEntries()
	maxScroll := len(entries) - v.visibleRows()
	if maxScroll < 0 {
		maxScroll = 0
	}
	if v.scroll < maxScroll {
		v.scroll++
	}
	// Re-enable auto-scroll if we've reached the bottom.
	if v.scroll >= maxScroll {
		v.pinned = true
	}
}

// ScrollUp moves the viewport up (shows older entries).
func (v *NetLogView) ScrollUp() {
	if v.scroll > 0 {
		v.scroll--
		v.pinned = false
	}
}

// Scroll returns the current scroll offset (exported for testing).
func (v *NetLogView) Scroll() int {
	return v.scroll
}

// Pinned returns true if auto-scroll is active (exported for testing).
func (v *NetLogView) Pinned() bool {
	return v.pinned
}

// visibleRows returns how many log rows fit in the available height.
// Reserves 1 row for the column header.
func (v *NetLogView) visibleRows() int {
	rows := v.height - 1
	if rows < 1 {
		return 1
	}
	return rows
}

// View renders the network log table in chronological order (oldest first).
// When pinned, the viewport stays at the bottom showing the newest entries.
func (v *NetLogView) View() string {
	entries := v.store.NetLogEntries()

	var sb strings.Builder

	// Column header row.
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

	visible := v.visibleRows()

	// Auto-scroll: pin viewport to the bottom so newest entries are always visible.
	if v.pinned {
		v.scroll = len(entries) - visible
		if v.scroll < 0 {
			v.scroll = 0
		}
	}

	// Clamp scroll bounds.
	maxScroll := len(entries) - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if v.scroll > maxScroll {
		v.scroll = maxScroll
	}

	start := v.scroll
	end := start + visible
	if end > len(entries) {
		end = len(entries)
	}

	for _, entry := range entries[start:end] {
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
