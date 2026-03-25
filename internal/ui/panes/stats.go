// Package panes contains the Bubble Tea pane models for the Spotnik TUI.
// Panes read from the central Store and emit request messages for side effects.
// Panes never call the API directly or import api/ — data flows through messages and store only.
package panes

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// StatsSection identifies the active section in the stats view.
type StatsSection int

const (
	// StatsSectionTopTracks is the top tracks section (focused by default).
	StatsSectionTopTracks StatsSection = iota
	// StatsSectionTopArtists is the top artists section.
	StatsSectionTopArtists
	// StatsSectionRecentlyPlayed is the recently played section.
	StatsSectionRecentlyPlayed
	// StatsSectionNetLog is the network log section.
	StatsSectionNetLog
)

const statsSectionCount = 4

// timeRanges is the cycle order for the f key.
var timeRanges = []string{"short_term", "medium_term", "long_term"}

// timeRangeLabels maps API values to human-readable display labels.
var timeRangeLabels = map[string]string{
	"short_term":  "4wk",
	"medium_term": "6mo",
	"long_term":   "all",
}

// StatsLoadedMsg is returned by the stats fetch command.
// TimeRange identifies which range was fetched.
// TopTracks and TopArtists carry the fetched data on success.
// Err is non-nil on failure. Update() writes data to the store; pane reads from store.
type StatsLoadedMsg struct {
	// TimeRange is the time range that was fetched ("short_term", "medium_term", "long_term").
	TimeRange string
	// TopTracks contains the fetched top tracks for the time range.
	TopTracks []domain.Track
	// TopArtists contains the fetched top artists for the time range.
	TopArtists []domain.FullArtist
	// Err is non-nil if the fetch failed.
	Err error
}

// FetchStatsMsg is a request message emitted by StatsView to ask the root app
// to fetch stats data for the given time range from the API.
type FetchStatsMsg struct {
	// TimeRange is "short_term", "medium_term", or "long_term".
	TimeRange string
}

// StatsView is the Bubble Tea model for the stats dashboard.
// It renders top tracks, top artists, and recently played in a two-pane layout.
// Time range and section focus are pane-local state; data is read from the Store.
// NOTE: StatsView never imports api/ — it reads all data through state.Store.
type StatsView struct {
	store *state.Store
	theme theme.Theme

	// activeSection is the currently focused section.
	activeSection StatsSection

	// timeRange is the currently active time range for top tracks/artists.
	timeRange string

	// cursor is the selection cursor within the active section.
	cursor int

	// scrollOffset is the first visible item index in the active section.
	scrollOffset int

	width  int
	height int
}

// NewStatsView constructs a StatsView with default short_term range and top-tracks focus.
func NewStatsView(store *state.Store, t theme.Theme) *StatsView {
	return &StatsView{
		store:         store,
		theme:         t,
		activeSection: StatsSectionTopTracks,
		timeRange:     "short_term",
	}
}

// SetSize updates the view's rendering dimensions.
func (sv *StatsView) SetSize(w, h int) {
	sv.width = w
	sv.height = h
}

// ActiveSection returns the currently focused section (exported for testing).
func (sv *StatsView) ActiveSection() StatsSection {
	return sv.activeSection
}

// TimeRange returns the active time range string (exported for testing).
func (sv *StatsView) TimeRange() string {
	return sv.timeRange
}

// Cursor returns the current cursor position within the active section (exported for testing).
func (sv *StatsView) Cursor() int {
	return sv.cursor
}

// Init requests short_term data and recently played from the root app on first open.
func (sv *StatsView) Init() tea.Cmd {
	// Emit request messages for the root app to handle — we never call the API directly.
	return tea.Batch(
		func() tea.Msg { return FetchStatsMsg{TimeRange: "short_term"} },
		func() tea.Msg { return FetchRecentlyPlayedRequestMsg{} },
	)
}

// Update handles all messages for the stats view.
func (sv *StatsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case StatsLoadedMsg:
		// Data is now in store; update local time range and reset cursor if range changed.
		if m.TimeRange != "" {
			sv.timeRange = m.TimeRange
		}
		return sv, nil

	case RecentlyPlayedLoadedMsg:
		// Recently played data is now in store; view reads from store on View().
		return sv, nil

	case tea.KeyMsg:
		return sv.handleKey(m)
	}

	return sv, nil
}

// handleKey routes keyboard events for the stats view.
func (sv *StatsView) handleKey(m tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.Type == tea.KeyTab:
		sv.activeSection = (sv.activeSection + 1) % statsSectionCount
		sv.scrollOffset = 0
		// Auto-scroll: when entering NetLog, jump cursor to newest entry.
		if sv.activeSection == StatsSectionNetLog {
			entries := sv.store.NetLogEntries()
			if len(entries) > 0 {
				sv.cursor = len(entries) - 1
				sv.ensureCursorVisible()
			} else {
				sv.cursor = 0
			}
		} else {
			sv.cursor = 0
		}
		return sv, nil

	case m.Type == tea.KeyRunes && string(m.Runes) == "j",
		m.Type == tea.KeyDown:
		sv.moveCursorDown()
		return sv, nil

	case m.Type == tea.KeyRunes && string(m.Runes) == "k",
		m.Type == tea.KeyUp:
		sv.moveCursorUp()
		return sv, nil

	case m.Type == tea.KeyEnter:
		return sv.handleEnter()

	case m.Type == tea.KeyRunes && string(m.Runes) == "f":
		return sv.handleCycleRange()
	}

	return sv, nil
}

// moveCursorDown moves the cursor down within the active section, bounded by section length.
func (sv *StatsView) moveCursorDown() {
	max := sv.activeSectionLen() - 1
	if max < 0 {
		max = 0
	}
	if sv.cursor < max {
		sv.cursor++
		sv.ensureCursorVisible()
	}
}

// moveCursorUp moves the cursor up within the active section, bounded at 0.
func (sv *StatsView) moveCursorUp() {
	if sv.cursor > 0 {
		sv.cursor--
		sv.ensureCursorVisible()
	}
}

// visibleItemCount returns the number of items that fit in the visible area
// for a section. The stats layout splits the available height across sections,
// so we use a conservative estimate: half the height for top sections
// (tracks/artists share space horizontally), full height minus header for others.
// NOTE: This is a per-section calculation, not a global one.
func (sv *StatsView) visibleItemCount() int {
	if sv.height <= 0 {
		return 10 // safe default for tests with no size set
	}
	switch sv.activeSection {
	case StatsSectionTopTracks, StatsSectionTopArtists:
		// Top row shares height: header (1) + range toggle (1) + items, roughly half the page.
		available := sv.height/2 - 3
		if available <= 0 {
			return 1
		}
		return available
	default:
		// Bottom sections: subtract header rows (section header + divider).
		available := sv.height - sv.height/2 - 3
		if available <= 0 {
			return 1
		}
		return available
	}
}

// ensureCursorVisible adjusts scrollOffset so the cursor is within the visible window.
func (sv *StatsView) ensureCursorVisible() {
	visible := sv.visibleItemCount()
	if sv.cursor < sv.scrollOffset {
		sv.scrollOffset = sv.cursor
	}
	if sv.cursor >= sv.scrollOffset+visible {
		sv.scrollOffset = sv.cursor - visible + 1
	}
	if sv.scrollOffset < 0 {
		sv.scrollOffset = 0
	}

}

// ScrollOffset returns the current scroll offset (exported for testing).
func (sv *StatsView) ScrollOffset() int {
	return sv.scrollOffset
}

// activeSectionLen returns the number of items in the currently focused section.
func (sv *StatsView) activeSectionLen() int {
	switch sv.activeSection {
	case StatsSectionTopTracks:
		return len(sv.store.TopTracks(sv.timeRange))
	case StatsSectionTopArtists:
		return len(sv.store.TopArtists(sv.timeRange))
	case StatsSectionRecentlyPlayed:
		return len(sv.store.RecentlyPlayed())
	case StatsSectionNetLog:
		return len(sv.store.NetLogEntries())
	}
	return 0
}

// handleEnter plays the currently selected item.
func (sv *StatsView) handleEnter() (tea.Model, tea.Cmd) {
	switch sv.activeSection {
	case StatsSectionTopTracks:
		tracks := sv.store.TopTracks(sv.timeRange)
		if sv.cursor < len(tracks) {
			uri := tracks[sv.cursor].URI
			return sv, func() tea.Msg {
				return PlayTrackMsg{TrackURI: uri}
			}
		}

	case StatsSectionTopArtists:
		artists := sv.store.TopArtists(sv.timeRange)
		if sv.cursor < len(artists) {
			uri := artists[sv.cursor].URI
			return sv, func() tea.Msg {
				return PlayContextMsg{ContextURI: uri}
			}
		}

	case StatsSectionRecentlyPlayed:
		items := sv.store.RecentlyPlayed()
		if sv.cursor < len(items) {
			uri := items[sv.cursor].Track.URI
			return sv, func() tea.Msg {
				return PlayTrackMsg{TrackURI: uri}
			}
		}
	}

	return sv, nil
}

// handleCycleRange advances to the next time range, checking the store cache first.
// On a cache hit, renders immediately with no fetch. On a miss, emits FetchStatsMsg.
func (sv *StatsView) handleCycleRange() (tea.Model, tea.Cmd) {
	// Find current index in the cycle.
	currentIdx := 0
	for i, r := range timeRanges {
		if r == sv.timeRange {
			currentIdx = i
			break
		}
	}
	nextRange := timeRanges[(currentIdx+1)%len(timeRanges)]
	sv.timeRange = nextRange
	sv.cursor = 0

	// Check if data for this range is already cached in the store.
	if sv.store.TopTracks(nextRange) != nil && sv.store.TopArtists(nextRange) != nil {
		// Cache hit — data is in store, render immediately with no fetch command.
		return sv, nil
	}

	// Cache miss — emit a request for the root app to fetch from the API.
	timeRange := nextRange
	return sv, func() tea.Msg { return FetchStatsMsg{TimeRange: timeRange} }
}

// View renders the stats dashboard. It is pure — no external calls.
// NOTE: only app.go renders the status bar — no pane-level hint bar here.
// Errors are routed through toast notifications (app.go); store.StatsError()
// is preserved for retry logic but no longer read in View().
func (sv *StatsView) View() string {
	return sv.renderDashboard()
}

// renderDashboard renders the full stats layout.
func (sv *StatsView) renderDashboard() string {
	var sb strings.Builder

	// Top row: TOP TRACKS | TOP ARTISTS (50/50 split).
	topTracksView := sv.renderTopTracksSection()
	topArtistsView := sv.renderTopArtistsSection()

	// Use lipgloss for side-by-side layout if we have width.
	if sv.width > 10 {
		halfWidth := sv.width / 2
		tracksStyle := lipgloss.NewStyle().Width(halfWidth)
		artistsStyle := lipgloss.NewStyle().Width(sv.width - halfWidth)
		topRow := lipgloss.JoinHorizontal(
			lipgloss.Top,
			tracksStyle.Render(topTracksView),
			artistsStyle.Render(topArtistsView),
		)
		sb.WriteString(topRow)
	} else {
		sb.WriteString(topTracksView)
		sb.WriteString("\n")
		sb.WriteString(topArtistsView)
	}

	sb.WriteString("\n")
	sb.WriteString(sv.renderRecentlyPlayedSection())

	sb.WriteString("\n")
	sb.WriteString(sv.renderNetLogSection())

	return sb.String()
}

// renderTopTracksSection renders the TOP TRACKS section with height capping.
func (sv *StatsView) renderTopTracksSection() string {
	focused := sv.activeSection == StatsSectionTopTracks
	var sb strings.Builder

	sb.WriteString(sv.renderSectionHeader("TOP TRACKS", focused))
	sb.WriteString("\n")
	sb.WriteString(sv.renderTimeRangeToggle(focused))
	sb.WriteString("\n")

	tracks := sv.store.TopTracks(sv.timeRange)
	if len(tracks) == 0 {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(sv.theme.TextMuted()).
			Render("  No listening data for this period"))
		return sb.String()
	}

	// Calculate scroll window for top-tracks section.
	scrollOffset := 0
	if focused {
		scrollOffset = sv.scrollOffset
	}
	visibleCount := sv.visibleItemCount()
	endIdx := scrollOffset + visibleCount
	if endIdx > len(tracks) {
		endIdx = len(tracks)
	}

	// Scroll-up indicator.
	if scrollOffset > 0 {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(sv.theme.TextMuted()).
			Render("  ▲"))
		sb.WriteString("\n")
	}

	for i := scrollOffset; i < endIdx; i++ {
		track := tracks[i]
		artistName := ""
		if len(track.Artists) > 0 {
			artistName = track.Artists[0].Name
		}
		row := fmt.Sprintf("  %2d  %-24s  %s", i+1, truncate(track.Name, 24), artistName)

		if focused && i == sv.cursor {
			sb.WriteString(lipgloss.NewStyle().
				Background(sv.theme.SelectedBg()).
				Foreground(sv.theme.TextPrimary()).
				Render(row))
		} else {
			sb.WriteString(lipgloss.NewStyle().
				Foreground(sv.theme.TextPrimary()).
				Render(row))
		}
		sb.WriteString("\n")
	}

	// Scroll-down indicator.
	if endIdx < len(tracks) {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(sv.theme.TextMuted()).
			Render("  ▼"))
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderTopArtistsSection renders the TOP ARTISTS section with height capping.
func (sv *StatsView) renderTopArtistsSection() string {
	focused := sv.activeSection == StatsSectionTopArtists
	var sb strings.Builder

	sb.WriteString(sv.renderSectionHeader("TOP ARTISTS", focused))
	sb.WriteString("\n")
	sb.WriteString(sv.renderTimeRangeToggle(focused))
	sb.WriteString("\n")

	artists := sv.store.TopArtists(sv.timeRange)
	if len(artists) == 0 {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(sv.theme.TextMuted()).
			Render("  No listening data for this period"))
		return sb.String()
	}

	// Calculate scroll window for top-artists section.
	scrollOffset := 0
	if focused {
		scrollOffset = sv.scrollOffset
	}
	visibleCount := sv.visibleItemCount()
	endIdx := scrollOffset + visibleCount
	if endIdx > len(artists) {
		endIdx = len(artists)
	}

	// Scroll-up indicator.
	if scrollOffset > 0 {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(sv.theme.TextMuted()).
			Render("  ▲"))
		sb.WriteString("\n")
	}

	for i := scrollOffset; i < endIdx; i++ {
		artist := artists[i]
		row := fmt.Sprintf("  %2d  %s", i+1, artist.Name)

		if focused && i == sv.cursor {
			sb.WriteString(lipgloss.NewStyle().
				Background(sv.theme.SelectedBg()).
				Foreground(sv.theme.TextPrimary()).
				Render(row))
		} else {
			sb.WriteString(lipgloss.NewStyle().
				Foreground(sv.theme.TextPrimary()).
				Render(row))
		}
		sb.WriteString("\n")
	}

	// Scroll-down indicator.
	if endIdx < len(artists) {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(sv.theme.TextMuted()).
			Render("  ▼"))
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderRecentlyPlayedSection renders the RECENTLY PLAYED section with height capping.
func (sv *StatsView) renderRecentlyPlayedSection() string {
	focused := sv.activeSection == StatsSectionRecentlyPlayed
	var sb strings.Builder

	sb.WriteString(sv.renderSectionHeader("RECENTLY PLAYED", focused))
	sb.WriteString("\n")

	items := sv.store.RecentlyPlayed()
	if len(items) == 0 {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(sv.theme.TextMuted()).
			Render("  No recent listening history"))
		return sb.String()
	}

	// Calculate scroll window for this section.
	scrollOffset := 0
	if focused {
		scrollOffset = sv.scrollOffset
	}
	visibleCount := sv.visibleItemCount()
	endIdx := scrollOffset + visibleCount
	if endIdx > len(items) {
		endIdx = len(items)
	}

	// Scroll-up indicator.
	if scrollOffset > 0 {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(sv.theme.TextMuted()).
			Render("  ▲"))
		sb.WriteString("\n")
	}

	for i := scrollOffset; i < endIdx; i++ {
		item := items[i]
		artistName := ""
		if len(item.Track.Artists) > 0 {
			artistName = item.Track.Artists[0].Name
		}
		relTime := formatPlayedAt(item.PlayedAt)
		row := fmt.Sprintf("  %-20s  ·  %-16s  ·  %-16s  %s",
			truncate(item.Track.Name, 20),
			truncate(artistName, 16),
			truncate(item.Track.Album.Name, 16),
			relTime,
		)

		if focused && i == sv.cursor {
			sb.WriteString(lipgloss.NewStyle().
				Background(sv.theme.SelectedBg()).
				Foreground(sv.theme.TextPrimary()).
				Render(row))
		} else {
			sb.WriteString(lipgloss.NewStyle().
				Foreground(sv.theme.TextPrimary()).
				Render(row))
		}
		sb.WriteString("\n")
	}

	// Scroll-down indicator.
	if endIdx < len(items) {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(sv.theme.TextMuted()).
			Render("  ▼"))
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderSectionHeader renders a section title using the SectionHeader theme token.
func (sv *StatsView) renderSectionHeader(title string, focused bool) string {
	color := sv.theme.SectionHeader()
	if focused {
		color = sv.theme.PlayingIndicator()
	}
	return lipgloss.NewStyle().
		Foreground(color).
		Bold(true).
		Render("  " + title)
}

// renderTimeRangeToggle renders the [4wk] [6mo] [all] range toggle row.
// The active range is highlighted using the ActiveBorder token.
func (sv *StatsView) renderTimeRangeToggle(sectionFocused bool) string {
	var parts []string
	for _, r := range timeRanges {
		label := timeRangeLabels[r]
		bracket := "[" + label + "]"
		if r == sv.timeRange && sectionFocused {
			parts = append(parts, lipgloss.NewStyle().
				Foreground(sv.theme.ActiveBorder()).
				Bold(true).
				Render(bracket))
		} else if r == sv.timeRange {
			parts = append(parts, lipgloss.NewStyle().
				Foreground(sv.theme.TextPrimary()).
				Bold(true).
				Render(bracket))
		} else {
			parts = append(parts, lipgloss.NewStyle().
				Foreground(sv.theme.TextMuted()).
				Render(bracket))
		}
	}
	return "  " + strings.Join(parts, " ")
}

// FormatRelativeTime returns a human-readable relative timestamp per the spec:
//
//	< 1 min   → "just now"
//	1–59 min  → "{n} min ago"
//	1–23 hr   → "{n} hr ago"
//	1–6 days  → "{n} days ago"
//	>= 7 days → "Mar 12" short date
func FormatRelativeTime(t time.Time) string {
	elapsed := time.Since(t)

	if elapsed < time.Minute {
		return "just now"
	}
	if elapsed < time.Hour {
		mins := int(elapsed.Minutes())
		return fmt.Sprintf("%d min ago", mins)
	}
	if elapsed < 24*time.Hour {
		hours := int(elapsed.Hours())
		return fmt.Sprintf("%d hr ago", hours)
	}
	days := int(elapsed.Hours() / 24)
	if days < 7 {
		return fmt.Sprintf("%d days ago", days)
	}
	return t.Format("Jan 2")
}

// formatPlayedAt parses an ISO 8601 played_at timestamp and returns a relative time string.
func formatPlayedAt(playedAt string) string {
	t, err := time.Parse(time.RFC3339, playedAt)
	if err != nil {
		return ""
	}
	return FormatRelativeTime(t)
}

// renderNetLogSection renders the NETWORK LOG section with cursor-based highlighting
// and height capping.
func (sv *StatsView) renderNetLogSection() string {
	focused := sv.activeSection == StatsSectionNetLog
	var sb strings.Builder

	sb.WriteString(sv.renderSectionHeader("NETWORK LOG", focused))
	sb.WriteString("\n")

	// Column header.
	header := fmt.Sprintf("  %-8s %-6s %-36s %6s %8s",
		"TIME", "METHOD", "PATH", "STATUS", "DURATION")
	sb.WriteString(lipgloss.NewStyle().
		Foreground(sv.theme.SectionHeader()).
		Bold(true).
		Render(header))
	sb.WriteString("\n")

	entries := sv.store.NetLogEntries()
	if len(entries) == 0 {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(sv.theme.TextMuted()).
			Render("  No API calls recorded yet"))
		return sb.String()
	}

	// Calculate scroll window for net log section.
	scrollOffset := 0
	if focused {
		scrollOffset = sv.scrollOffset
	}
	visibleCount := sv.visibleItemCount()
	endIdx := scrollOffset + visibleCount
	if endIdx > len(entries) {
		endIdx = len(entries)
	}

	// Scroll-up indicator.
	if scrollOffset > 0 {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(sv.theme.TextMuted()).
			Render("  ▲"))
		sb.WriteString("\n")
	}

	// Chronological order (oldest first), cursor highlights current row.
	for i := scrollOffset; i < endIdx; i++ {
		entry := entries[i]
		timeStr := entry.Timestamp.Format("15:04:05")
		row := fmt.Sprintf("  %-8s %-6s %-36s %6d %6dms",
			timeStr, entry.Method, truncate(entry.Path, 36),
			entry.StatusCode, entry.DurationMs)

		if focused && i == sv.cursor {
			// Highlighted cursor row.
			sb.WriteString(lipgloss.NewStyle().
				Background(sv.theme.SelectedBg()).
				Foreground(sv.theme.TextPrimary()).
				Render(row))
		} else {
			// Color-code by status: 2xx green, 4xx+ red, others default.
			style := lipgloss.NewStyle()
			if entry.StatusCode >= 400 {
				style = style.Foreground(sv.theme.Error())
			} else if entry.StatusCode >= 200 && entry.StatusCode < 300 {
				style = style.Foreground(sv.theme.PlayingIndicator())
			} else {
				style = style.Foreground(sv.theme.TextPrimary())
			}
			sb.WriteString(style.Render(row))
		}
		sb.WriteString("\n")
	}

	// Scroll-down indicator.
	if endIdx < len(entries) {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(sv.theme.TextMuted()).
			Render("  ▼"))
		sb.WriteString("\n")
	}

	return sb.String()
}

// NOTE: truncate is defined in search.go and reused here across the panes package.
