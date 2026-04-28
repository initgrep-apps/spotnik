package panes

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// Compile-time check: PollingTrafficPane implements layout.Pane.
var _ layout.Pane = &PollingTrafficPane{}

// PollingTrafficPane shows the current polling cadence for playback and library cache freshness.
// The playback row is driven by PollingSnapshotMsg; library rows read store TTL sentinel methods.
type PollingTrafficPane struct {
	store        state.StateReader
	theme        theme.Theme
	focused      bool
	width        int
	height       int
	pollSnapshot PollingSnapshotMsg
}

// NewPollingTrafficPane creates a PollingTrafficPane with the given store and theme.
func NewPollingTrafficPane(s state.StateReader, th theme.Theme) *PollingTrafficPane {
	return &PollingTrafficPane{store: s, theme: th}
}

// ID returns the PanePollingTraffic identifier.
func (p *PollingTrafficPane) ID() layout.PaneID { return layout.PanePollingTraffic }

// Title returns the display title shown in the pane border.
func (p *PollingTrafficPane) Title() string { return "Polling Traffic" }

// ToggleKey returns 3 — the Page B toggle key for this pane.
func (p *PollingTrafficPane) ToggleKey() int { return 3 }

// Actions returns nil — this pane has no interactive shortcuts.
func (p *PollingTrafficPane) Actions() []layout.Action { return nil }

// IsFocused returns whether this pane has keyboard focus.
func (p *PollingTrafficPane) IsFocused() bool { return p.focused }

// SetFocused updates the keyboard focus state.
func (p *PollingTrafficPane) SetFocused(f bool) { p.focused = f }

// Init returns nil — PollingTrafficPane reacts to PollingSnapshotMsg from the app.
func (p *PollingTrafficPane) Init() tea.Cmd { return nil }

// SetSize updates the content area dimensions.
func (p *PollingTrafficPane) SetSize(w, h int) { p.width = w; p.height = h }

// SetTheme updates the theme reference for runtime theme switching.
func (p *PollingTrafficPane) SetTheme(th theme.Theme) { p.theme = th }

// Update stores PollingSnapshotMsg when received.
func (p *PollingTrafficPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m, ok := msg.(PollingSnapshotMsg); ok {
		p.pollSnapshot = m
	}
	return p, nil
}

// View renders the 5-row polling traffic grid. Pure — no side effects.
func (p *PollingTrafficPane) View() string {
	if p.width == 0 || p.height == 0 {
		return ""
	}

	th := p.theme
	mode := uikit.ActiveMode()
	const labelWidth = 10

	mutedStyle := lipgloss.NewStyle().Foreground(th.TextMuted())

	renderTypeIcon := func(role uikit.GlyphRole) string {
		return mutedStyle.Render(uikit.GlyphFor(role, mode))
	}

	// Playback row
	var playRow string
	{
		icon := renderTypeIcon(uikit.GlyphMusicNote)
		label := mutedStyle.Render(uikit.PadOrTruncate("Playback", labelWidth))
		if p.pollSnapshot.IsIdle {
			glyph := lipgloss.NewStyle().Foreground(th.Warning()).Render(uikit.GlyphFor(uikit.GlyphPaused, mode))
			intervalStr := pollingHumanInterval(p.pollSnapshot.TickIntervalMs)
			status := lipgloss.NewStyle().Foreground(th.Warning()).Render(fmt.Sprintf("idle · %s", intervalStr))
			playRow = icon + "  " + label + "  " + glyph + " " + status
		} else {
			glyph := lipgloss.NewStyle().Foreground(th.Success()).Render(uikit.GlyphFor(uikit.GlyphPlaying, mode))
			intervalStr := pollingHumanInterval(p.pollSnapshot.TickIntervalMs)
			status := lipgloss.NewStyle().Foreground(th.Success()).Render(fmt.Sprintf("%s · running", intervalStr))
			playRow = icon + "  " + label + "  " + glyph + " " + status
		}
	}

	// Library cache rows: Playlists, Albums, Liked, Recent
	type cacheRow struct {
		typeRole  uikit.GlyphRole
		label     string
		fetchedAt time.Time
		ttl       time.Duration
	}
	cacheRows := []cacheRow{
		{uikit.GlyphQueue, "Playlists", p.store.PlaylistsFetchedAt(), state.PlaylistsTTL},
		{uikit.GlyphDoubleNote, "Albums", p.store.AlbumsFetchedAt(), state.AlbumsTTL},
		{uikit.GlyphPinned, "Liked", p.store.LikedTracksFetchedAt(), state.LikedTracksTTL},
		{uikit.GlyphDeadline, "Recent", p.store.RecentPlayedFetchedAt(), state.RecentlyPlayedTTL},
	}

	renderedRows := make([]string, 0, 5)
	renderedRows = append(renderedRows, playRow)

	for _, cr := range cacheRows {
		icon := renderTypeIcon(cr.typeRole)
		label := mutedStyle.Render(uikit.PadOrTruncate(cr.label, labelWidth))
		var statusStr string
		if cr.fetchedAt.IsZero() {
			statusStr = mutedStyle.Render("never fetched")
		} else if !state.IsStale(cr.fetchedAt, cr.ttl) {
			freshGlyph := lipgloss.NewStyle().Foreground(th.TextMuted()).Render(uikit.GlyphFor(uikit.GlyphAvailable, mode))
			statusStr = freshGlyph + " " + mutedStyle.Render("fresh")
		} else {
			age := cacheAge(cr.fetchedAt)
			d := time.Since(cr.fetchedAt)
			var staleColor lipgloss.Color
			if d >= time.Hour {
				staleColor = th.Error()
			} else {
				staleColor = th.Warning()
			}
			warnGlyph := lipgloss.NewStyle().Foreground(staleColor).Render(uikit.GlyphFor(uikit.GlyphWarning, mode))
			statusStr = warnGlyph + " " + lipgloss.NewStyle().Foreground(staleColor).Render(age+" stale")
		}
		renderedRows = append(renderedRows, icon+"  "+label+"  "+statusStr)
	}

	return lipgloss.NewStyle().PaddingLeft(1).Render(strings.Join(renderedRows, "\n"))
}

// pollingHumanInterval converts milliseconds to a human-readable interval string.
// >= 1000ms → "Xs"; < 1000ms → "Xms"; <= 0 → "?".
func pollingHumanInterval(ms int) string {
	if ms <= 0 {
		return "?"
	}
	if ms >= 1000 {
		return fmt.Sprintf("%ds", ms/1000)
	}
	return fmt.Sprintf("%dms", ms)
}

// cacheAge returns a human-readable duration since t (e.g. "3m", "2h 15m"),
// without an "ago" suffix so the caller can append context-specific text
// such as " stale".
func cacheAge(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}
