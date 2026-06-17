// Package panes — EpisodeDetailsOverlay is the floating overlay that shows full
// episode details when the user presses 'i' while an episode is playing.
// It displays the episode name, show name, publisher, duration, release date,
// and a scrollable description rendered from html_description.
// Esc or 'q' closes the overlay; description scrolls via viewport (j/k, ↑/↓, pgup/pgdn, home/end, mouse wheel).
package panes

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// EpisodeDetailsOverlay is the floating overlay that displays full episode details.
// Pressing Esc or 'q' emits EpisodeDetailsClosedMsg. Description scrolling is
// delegated to a bubbles viewport.Model (j/k/↑/↓/pgup/pgdn/home/end/mouse wheel).
// All other keys are consumed (modal).
type EpisodeDetailsOverlay struct {
	store     state.StateReader
	theme     theme.Theme
	width     int
	height    int
	viewport  viewport.Model
	vpContent string // last content set on viewport; guards against redundant SetContent calls
}

// NewEpisodeDetailsOverlay creates an overlay using the given store and theme.
// The store is used to read PlaybackState for episode data in View().
func NewEpisodeDetailsOverlay(store state.StateReader, t theme.Theme) *EpisodeDetailsOverlay {
	return &EpisodeDetailsOverlay{
		store: store,
		theme: t,
	}
}

// SetSize updates the render dimensions for the overlay and resizes the viewport.
func (o *EpisodeDetailsOverlay) SetSize(width, height int) {
	o.width = width
	o.height = height
	o.resizeViewport()
}

// resizeViewport computes the viewport dimensions from current overlay size.
func (o *EpisodeDetailsOverlay) resizeViewport() {
	vpW := o.overlayWidth() - 2
	if vpW < 2 {
		vpW = 2
	}
	vpH := min(o.height-12, 40) // -5 header, -2 border, -1 keybar, -4 padding
	if vpH < 3 {
		vpH = 3
	}
	if o.viewport.Width != vpW || o.viewport.Height != vpH {
		o.viewport.Width = vpW
		o.viewport.Height = vpH
	}
	o.viewport.MouseWheelEnabled = true
}

// SetTheme updates the overlay's theme reference for runtime theme switching.
func (o *EpisodeDetailsOverlay) SetTheme(th theme.Theme) {
	o.theme = th
}

// Init satisfies tea.Model; no startup command needed.
func (o *EpisodeDetailsOverlay) Init() tea.Cmd { return nil }

// Update handles keyboard input for the episode details overlay.
// Esc and 'q' close it; Home/End jump to top/bottom; all other messages
// are delegated to the viewport for scroll handling.
func (o *EpisodeDetailsOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEsc || (keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "q") {
			return o, func() tea.Msg { return EpisodeDetailsClosedMsg{} }
		}
		if keyMsg.Type == tea.KeyHome {
			o.viewport.GotoTop()
			return o, nil
		}
		if keyMsg.Type == tea.KeyEnd {
			o.viewport.GotoBottom()
			return o, nil
		}
	}
	var cmd tea.Cmd
	o.viewport, cmd = o.viewport.Update(msg)
	return o, cmd
}

// overlayWidth returns the fixed overlay width (80), capped to the terminal width.
func (o *EpisodeDetailsOverlay) overlayWidth() int {
	const fixedWidth = 80
	if o.width > 0 && fixedWidth > o.width {
		return o.width
	}
	return fixedWidth
}

// View renders the episode details overlay with a viewport for scrollable description.
func (o *EpisodeDetailsOverlay) View() string {
	ps := o.store.PlaybackState()
	if ps == nil || ps.Episode == nil {
		return o.renderEmptyChrome("No episode playing",
			"Press i when an episode is playing to see details")
	}

	ep := ps.Episode
	totalW := o.overlayWidth()
	innerW := totalW - 2
	if innerW < 2 {
		innerW = 2
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(o.theme.TextPrimary()).
		Bold(true)
	metaStyle := lipgloss.NewStyle().
		Foreground(o.theme.TextSecondary())
	publisherStyle := lipgloss.NewStyle().
		Foreground(o.theme.TextSecondary())

	var headerLines []string

	headerLines = append(headerLines, titleStyle.Width(innerW).Render(ep.Name))

	var metaParts []string
	if ep.Show != nil && ep.Show.Name != "" {
		metaParts = append(metaParts, ep.Show.Name)
	}
	if ep.DurationMs > 0 {
		metaParts = append(metaParts, formatDuration(ep.DurationMs))
	}
	if ep.ReleaseDate != "" {
		metaParts = append(metaParts, ep.ReleaseDate)
	}
	if len(metaParts) > 0 {
		headerLines = append(headerLines,
			metaStyle.Width(innerW).Render(strings.Join(metaParts, " "+uikit.GlyphFor(uikit.GlyphSeparator, uikit.ActiveMode())+" ")))
	}

	if ep.Show != nil && ep.Show.Publisher != "" {
		headerLines = append(headerLines,
			publisherStyle.Width(innerW).Render("Published by: "+ep.Show.Publisher))
	}

	headerLines = append(headerLines, "")

	desc := o.renderDescription(ep)

	if desc != o.vpContent {
		o.viewport.SetContent(desc)
		o.vpContent = desc
	}

	keyHintStyle := lipgloss.NewStyle().Foreground(o.theme.KeyHint()).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(o.theme.TextMuted())
	scrollPct := fmt.Sprintf("%.0f%%", o.viewport.ScrollPercent()*100)
	keybarLine := dimStyle.Render(keyHintStyle.Render("Esc") + " close  " + scrollPct)

	vpStr := o.viewport.View()
	vpLines := strings.Split(vpStr, "\n")
	allLines := append(headerLines, vpLines...)
	paddedContent := lipgloss.NewStyle().Padding(1, 2).Render(strings.Join(allLines, "\n"))
	paddedKeybar := lipgloss.NewStyle().Padding(0, 2, 1, 2).Width(innerW).MaxWidth(innerW).Render(keybarLine)

	composite := lipgloss.JoinVertical(lipgloss.Left, paddedContent, paddedKeybar)
	height := strings.Count(composite, "\n") + 1 + 2

	chrome := uikit.OverlayChrome{
		Width:  totalW,
		Height: height,
		Title:  "Episode Details",
		Theme:  o.theme,
	}
	return chrome.Render(composite)
}

// renderEmptyChrome renders the overlay chrome around an EmptyState.
func (o *EpisodeDetailsOverlay) renderEmptyChrome(text, hint string) string {
	totalW := o.overlayWidth()
	innerW := totalW - 2
	if innerW < 2 {
		innerW = 2
	}
	const emptyStateHeight = 6
	inner := uikit.EmptyState{
		Text:   text,
		Hint:   hint,
		Width:  innerW,
		Height: emptyStateHeight - 2,
		Theme:  o.theme,
	}.Render()
	chrome := uikit.OverlayChrome{
		Width:  totalW,
		Title:  "Episode Details",
		Height: emptyStateHeight,
		Theme:  o.theme,
	}
	return chrome.Render(inner)
}

// renderDescription returns the description text for the episode, using
// html_description (preferred), description (fallback), or a muted placeholder.
func (o *EpisodeDetailsOverlay) renderDescription(ep *domain.Episode) string {
	if ep.HTMLDescription != "" {
		md := htmlToMarkdown(ep.HTMLDescription)
		rendered, err := renderMarkdown(md, o.viewport.Width)
		if err == nil && rendered != "" {
			return rendered
		}
		if md != "" {
			return md
		}
	}
	if ep.Description != "" {
		return ep.Description
	}
	return "No description available."
}

// formatDuration converts milliseconds to a human-readable duration string.
func formatDuration(ms int) string {
	if ms <= 0 {
		return ""
	}
	totalSec := ms / 1000
	hours := totalSec / 3600
	mins := (totalSec % 3600) / 60
	secs := totalSec % 60
	if hours > 0 {
		return fmt.Sprintf("%dh %02dm", hours, mins)
	}
	if mins > 0 {
		return fmt.Sprintf("%dm %02ds", mins, secs)
	}
	return fmt.Sprintf("%ds", secs)
}
