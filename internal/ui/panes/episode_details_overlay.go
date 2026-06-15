// Package panes — EpisodeDetailsOverlay is the floating overlay that shows full
// episode details when the user presses 'i' while an episode is playing.
// It displays the episode name, show name, publisher, duration, release date,
// and a scrollable description rendered from html_description.
// Esc or 'q' closes the overlay; j/k and ↑/↓ scroll the description.
package panes

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
)

// EpisodeDetailsOverlay is the floating overlay that displays full episode details.
// Pressing Esc or 'q' emits EpisodeDetailsClosedMsg; j/k and ↑/↓ scroll the
// description when it exceeds the visible area. All other keys are consumed (modal).
type EpisodeDetailsOverlay struct {
	store     state.StateReader
	theme     theme.Theme
	width     int
	height    int
	scrollY   int
	maxScroll int
}

// NewEpisodeDetailsOverlay creates an overlay using the given store and theme.
// The store is used to read PlaybackState for episode data in View().
func NewEpisodeDetailsOverlay(store state.StateReader, t theme.Theme) *EpisodeDetailsOverlay {
	return &EpisodeDetailsOverlay{
		store: store,
		theme: t,
	}
}

// SetSize updates the render dimensions for the overlay.
func (o *EpisodeDetailsOverlay) SetSize(width, height int) {
	o.width = width
	o.height = height
	o.clampScroll()
}

// SetTheme updates the overlay's theme reference for runtime theme switching.
func (o *EpisodeDetailsOverlay) SetTheme(th theme.Theme) {
	o.theme = th
}

// Init satisfies tea.Model; no startup command needed.
func (o *EpisodeDetailsOverlay) Init() tea.Cmd { return nil }

// Update handles keyboard input for the episode details overlay.
// Esc and 'q' close it; j/k and ↑/↓ scroll the description.
// All other keys are consumed with nil cmd (modal).
func (o *EpisodeDetailsOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return o, nil
	}
	switch {
	case keyMsg.Type == tea.KeyEsc:
		return o, func() tea.Msg { return EpisodeDetailsClosedMsg{} }
	case keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "q":
		return o, func() tea.Msg { return EpisodeDetailsClosedMsg{} }
	case keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "j",
		keyMsg.Type == tea.KeyDown:
		if o.scrollY < o.maxScroll {
			o.scrollY++
		}
		return o, nil
	case keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "k",
		keyMsg.Type == tea.KeyUp:
		if o.scrollY > 0 {
			o.scrollY--
		}
		return o, nil
	}
	return o, nil
}

// clampScroll ensures scrollY stays within [0, maxScroll].
func (o *EpisodeDetailsOverlay) clampScroll() {
	if o.scrollY < 0 {
		o.scrollY = 0
	}
	if o.maxScroll < 0 {
		o.maxScroll = 0
	}
	if o.scrollY > o.maxScroll {
		o.scrollY = o.maxScroll
	}
}

// overlayWidth returns the fixed overlay width (80), capped to the terminal width.
func (o *EpisodeDetailsOverlay) overlayWidth() int {
	const fixedWidth = 80
	if o.width > 0 && fixedWidth > o.width {
		return o.width
	}
	return fixedWidth
}

// View renders the episode details overlay with a centered btop-style border.
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

	// Title line
	headerLines = append(headerLines, titleStyle.Width(innerW).Render(ep.Name))

	// Metadata line: show name · duration · release date
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

	// Publisher line
	if ep.Show != nil && ep.Show.Publisher != "" {
		headerLines = append(headerLines,
			publisherStyle.Width(innerW).Render("Published by: "+ep.Show.Publisher))
	}

	// Blank separator
	headerLines = append(headerLines, "")

	// Description
	desc := o.renderDescription(ep)
	descLines := strings.Split(desc, "\n")

	// Compute visible height
	availInner := o.availableInnerHeight()
	reservedLines := len(headerLines) + 1 // +1 for keybar
	visibleDesc := availInner - reservedLines
	if visibleDesc < 3 {
		visibleDesc = 3
	}
	if visibleDesc > len(descLines) {
		visibleDesc = len(descLines)
	}

	o.maxScroll = max(0, len(descLines)-visibleDesc)
	o.clampScroll()

	// Scroll the description lines
	start := o.scrollY
	end := o.scrollY + visibleDesc
	if end > len(descLines) {
		end = len(descLines)
	}
	if start > len(descLines) {
		start = len(descLines)
	}
	scrolledDesc := descLines[start:end]

	mutedStyle := lipgloss.NewStyle().Foreground(o.theme.TextMuted())
	var descRendered []string
	for _, l := range scrolledDesc {
		descRendered = append(descRendered, mutedStyle.Width(innerW).Render(l))
	}

	// Keybar
	keyHintStyle := lipgloss.NewStyle().Foreground(o.theme.KeyHint()).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(o.theme.TextMuted())
	scrollHint := ""
	if o.maxScroll > 0 {
		scrollHint = "  " + dimStyle.Render(keyHintStyle.Render("j/k")+" scroll")
	}
	keybarLine := dimStyle.Render(keyHintStyle.Render("Esc")+" close"+scrollHint)

	// Compose
	allLines := append(headerLines, descRendered...)
	paddedContent := lipgloss.NewStyle().Padding(1, 2).Render(strings.Join(allLines, "\n"))
	paddedKeybar := lipgloss.NewStyle().Padding(0, 2, 1, 2).Width(innerW).MaxWidth(innerW).Render(keybarLine)

	composite := lipgloss.JoinVertical(lipgloss.Left, paddedContent, paddedKeybar)
	height := strings.Count(composite, "\n") + 1 + 2 // +1 last line, +2 borders

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
		rendered, err := renderMarkdown(md, o.overlayWidth()-6)
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

// availableInnerHeight returns how many inner lines are available
// given the current terminal height. Falls back to a sensible minimum.
func (o *EpisodeDetailsOverlay) availableInnerHeight() int {
	if o.height <= 0 {
		return 20
	}
	// Terminal height minus 2 margin rows minus 2 border rows = inner height
	avail := o.height - 4
	if avail < 10 {
		avail = 10
	}
	return avail
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