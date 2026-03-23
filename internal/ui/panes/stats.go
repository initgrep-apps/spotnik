package panes

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/api"
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
)

// timeRanges is the cycle order for the f key.
var timeRanges = []string{"short_term", "medium_term", "long_term"}

// timeRangeLabels maps API values to human-readable display labels.
var timeRangeLabels = map[string]string{
	"short_term":  "4wk",
	"medium_term": "6mo",
	"long_term":   "all",
}

// StatsLoadedMsg is sent when top tracks and top artists have been fetched
// and written to the store for a given time range.
type StatsLoadedMsg struct {
	TopTracks  []api.Track
	TopArtists []api.FullArtist
	TimeRange  string
}

// statsTimeRangeChangedMsg is an internal message sent when the user
// switches the time range and uncached data needs to be fetched.
type statsTimeRangeChangedMsg struct{ timeRange string }

// FetchStatsMsg is a request message emitted by StatsView to ask the root app
// to fetch stats data for the given time range from the API.
type FetchStatsMsg struct {
	// TimeRange is "short_term", "medium_term", or "long_term".
	TimeRange string
}

// StatsView is the Bubble Tea model for the stats dashboard.
// It renders top tracks, top artists, and recently played in a two-pane layout.
// Time range and section focus are pane-local state; data is read from the Store.
type StatsView struct {
	store   *state.Store
	theme   theme.Theme
	userAPI *api.UserClient

	// activeSection is the currently focused section.
	activeSection StatsSection

	// timeRange is the currently active time range for top tracks/artists.
	timeRange string

	// cursor is the selection cursor within the active section.
	cursor int

	// loading is true while data is being fetched.
	loading bool

	// topTracks holds the currently displayed top tracks (for the active time range).
	topTracks []api.Track

	// topArtists holds the currently displayed top artists (for the active time range).
	topArtists []api.FullArtist

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

// SetUserAPI injects the Spotify user API client so Init() can fetch data.
// Called by the root app after construction.
func (sv *StatsView) SetUserAPI(client *api.UserClient) {
	sv.userAPI = client
}

// SetSize updates the view's rendering dimensions.
func (sv *StatsView) SetSize(w, h int) {
	sv.width = w
	sv.height = h
}

// ActiveSection returns the currently focused section (for testing).
func (sv *StatsView) ActiveSection() StatsSection {
	return sv.activeSection
}

// TimeRange returns the active time range string (for testing).
func (sv *StatsView) TimeRange() string {
	return sv.timeRange
}

// Cursor returns the current cursor position within the active section (for testing).
func (sv *StatsView) Cursor() int {
	return sv.cursor
}

// Init fetches short_term top tracks, top artists, and recently played on first open.
func (sv *StatsView) Init() tea.Cmd {
	sv.loading = true
	return tea.Batch(
		sv.fetchStatsCmd("short_term"),
		sv.fetchRecentlyPlayedCmd(),
	)
}

// fetchStatsCmd returns a command that fetches top tracks and artists for the given time range.
// It writes results to the store and returns a StatsLoadedMsg.
func (sv *StatsView) fetchStatsCmd(timeRange string) tea.Cmd {
	userAPI := sv.userAPI
	store := sv.store
	return func() tea.Msg {
		if userAPI == nil {
			return StatsLoadedMsg{TimeRange: timeRange}
		}
		ctx := context.Background()
		tracks, err := userAPI.GetTopTracks(ctx, timeRange, 25)
		if err != nil {
			return StatsLoadedMsg{TimeRange: timeRange}
		}
		artists, err := userAPI.GetTopArtists(ctx, timeRange, 25)
		if err != nil {
			return StatsLoadedMsg{TimeRange: timeRange}
		}
		store.SetTopTracks(timeRange, tracks)
		store.SetTopArtists(timeRange, artists)
		return StatsLoadedMsg{
			TopTracks:  tracks,
			TopArtists: artists,
			TimeRange:  timeRange,
		}
	}
}

// fetchRecentlyPlayedCmd returns a command that fetches recently played tracks.
func (sv *StatsView) fetchRecentlyPlayedCmd() tea.Cmd {
	userAPI := sv.userAPI
	store := sv.store
	return func() tea.Msg {
		if userAPI == nil {
			return RecentlyPlayedLoadedMsg{}
		}
		items, err := userAPI.GetRecentlyPlayed(context.Background(), 50)
		if err != nil {
			return RecentlyPlayedLoadedMsg{}
		}
		store.SetRecentlyPlayed(items)
		return RecentlyPlayedLoadedMsg{}
	}
}

// Update handles all messages for the stats view.
func (sv *StatsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case StatsLoadedMsg:
		sv.loading = false
		sv.topTracks = m.TopTracks
		sv.topArtists = m.TopArtists
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
		sv.activeSection = (sv.activeSection + 1) % 3
		sv.cursor = 0
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
	}
}

// moveCursorUp moves the cursor up within the active section, bounded at 0.
func (sv *StatsView) moveCursorUp() {
	if sv.cursor > 0 {
		sv.cursor--
	}
}

// activeSectionLen returns the number of items in the currently focused section.
func (sv *StatsView) activeSectionLen() int {
	switch sv.activeSection {
	case StatsSectionTopTracks:
		return len(sv.topTracks)
	case StatsSectionTopArtists:
		return len(sv.topArtists)
	case StatsSectionRecentlyPlayed:
		return len(sv.store.RecentlyPlayed())
	}
	return 0
}

// handleEnter plays the currently selected item.
func (sv *StatsView) handleEnter() (tea.Model, tea.Cmd) {
	switch sv.activeSection {
	case StatsSectionTopTracks:
		if sv.cursor < len(sv.topTracks) {
			track := sv.topTracks[sv.cursor]
			return sv, func() tea.Msg {
				return PlayTrackMsg{TrackURI: track.URI}
			}
		}

	case StatsSectionTopArtists:
		if sv.cursor < len(sv.topArtists) {
			artist := sv.topArtists[sv.cursor]
			return sv, func() tea.Msg {
				return PlayContextMsg{ContextURI: artist.URI}
			}
		}

	case StatsSectionRecentlyPlayed:
		items := sv.store.RecentlyPlayed()
		if sv.cursor < len(items) {
			track := items[sv.cursor].Track
			return sv, func() tea.Msg {
				return PlayTrackMsg{TrackURI: track.URI}
			}
		}
	}

	return sv, nil
}

// handleCycleRange advances to the next time range, checking the cache first.
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

	// Check if data for this range is already cached.
	if sv.store.TopTracks(nextRange) != nil && sv.store.TopArtists(nextRange) != nil {
		// Cache hit — load from store immediately, no fetch.
		sv.topTracks = sv.store.TopTracks(nextRange)
		sv.topArtists = sv.store.TopArtists(nextRange)
		return sv, nil
	}

	// Cache miss — trigger fetch.
	sv.loading = true
	return sv, sv.fetchStatsCmd(nextRange)
}

// View renders the stats dashboard. It is pure — no external calls.
func (sv *StatsView) View() string {
	if sv.loading && len(sv.topTracks) == 0 && len(sv.topArtists) == 0 {
		return sv.renderLoading()
	}
	return sv.renderDashboard()
}

// renderLoading renders a loading spinner/message.
func (sv *StatsView) renderLoading() string {
	header := sv.renderSectionHeader("STATS", false)
	spinner := lipgloss.NewStyle().
		Foreground(sv.theme.TextMuted()).
		Render("  Loading stats...")
	return strings.Join([]string{header, spinner}, "\n")
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
	sb.WriteString(sv.renderHelpBar())

	return sb.String()
}

// renderTopTracksSection renders the TOP TRACKS section.
func (sv *StatsView) renderTopTracksSection() string {
	focused := sv.activeSection == StatsSectionTopTracks
	var sb strings.Builder

	sb.WriteString(sv.renderSectionHeader("TOP TRACKS", focused))
	sb.WriteString("\n")
	sb.WriteString(sv.renderTimeRangeToggle(focused))
	sb.WriteString("\n")

	if len(sv.topTracks) == 0 {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(sv.theme.TextMuted()).
			Render("  No listening data for this period"))
		return sb.String()
	}

	for i, track := range sv.topTracks {
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

	return sb.String()
}

// renderTopArtistsSection renders the TOP ARTISTS section.
func (sv *StatsView) renderTopArtistsSection() string {
	focused := sv.activeSection == StatsSectionTopArtists
	var sb strings.Builder

	sb.WriteString(sv.renderSectionHeader("TOP ARTISTS", focused))
	sb.WriteString("\n")
	sb.WriteString(sv.renderTimeRangeToggle(focused))
	sb.WriteString("\n")

	if len(sv.topArtists) == 0 {
		sb.WriteString(lipgloss.NewStyle().
			Foreground(sv.theme.TextMuted()).
			Render("  No listening data for this period"))
		return sb.String()
	}

	for i, artist := range sv.topArtists {
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

	return sb.String()
}

// renderRecentlyPlayedSection renders the RECENTLY PLAYED section.
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

	for i, item := range items {
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

// renderHelpBar renders the keybinding hint line at the bottom.
func (sv *StatsView) renderHelpBar() string {
	style := lipgloss.NewStyle().Foreground(sv.theme.TextMuted())
	keyStyle := lipgloss.NewStyle().Foreground(sv.theme.KeyHint()).Bold(true)

	hints := []string{
		keyStyle.Render("Tab") + " next section",
		keyStyle.Render("j/k") + " move",
		keyStyle.Render("Enter") + " play",
		keyStyle.Render("f") + " cycle time range",
		keyStyle.Render("1") + " library view",
	}
	return style.Render("  " + strings.Join(hints, "   "))
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

// NOTE: truncate is defined in search.go and reused here across the panes package.
