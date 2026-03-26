// Package panes — QueuePane displays the current play queue in the right pane.
// It renders a "NOW" section showing the currently playing track and a "NEXT UP"
// section listing upcoming tracks with j/k navigation and Enter to play.
package panes

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// Compile-time check: QueuePane implements layout.Pane.
var _ layout.Pane = &QueuePane{}

// QueuePane is the right-pane Bubble Tea model that shows what is playing
// now and what is coming up next. It never imports api/ directly — all data
// is read from the central store.
type QueuePane struct {
	store   *state.Store
	theme   theme.Theme
	focused bool

	// cursor is the zero-based index into the NEXT UP queue list.
	cursor int

	// scrollOffset is the first visible item index in the NEXT UP list.
	scrollOffset int

	width  int
	height int
}

// NewQueuePane creates a new QueuePane with the given store, theme, and focus state.
func NewQueuePane(store *state.Store, theme theme.Theme, focused bool) *QueuePane {
	return &QueuePane{
		store:   store,
		theme:   theme,
		focused: focused,
	}
}

// ID returns the PaneQueue identifier for this pane slot.
func (q *QueuePane) ID() layout.PaneID { return layout.PaneQueue }

// Title returns the display title shown in the pane border.
func (q *QueuePane) Title() string { return "Queue" }

// ToggleKey returns 2 — the number key for btop-style pane toggling.
func (q *QueuePane) ToggleKey() int { return 2 }

// Actions returns the pane-specific shortcut hints displayed in the border.
// Returns the close action when filter is active; otherwise the default set.
func (q *QueuePane) Actions() []layout.Action {
	return []layout.Action{
		{Key: "f", Label: "filter"},
		{Key: "A", Label: "add"},
	}
}

// Init satisfies tea.Model. The queue pane has no startup command of its own —
// the root app's tick loop drives queue refreshes.
func (q *QueuePane) Init() tea.Cmd {
	return nil
}

// IsFocused returns true when the queue pane has keyboard focus.
func (q *QueuePane) IsFocused() bool {
	return q.focused
}

// SetFocused sets the keyboard focus state.
func (q *QueuePane) SetFocused(focused bool) {
	q.focused = focused
}

// SetSize updates the render dimensions.
func (q *QueuePane) SetSize(width, height int) {
	q.width = width
	q.height = height
}

// Cursor returns the current cursor position in the NEXT UP list.
func (q *QueuePane) Cursor() int {
	return q.cursor
}

// Update handles key events when the pane is focused.
func (q *QueuePane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !q.focused {
		return q, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return q, nil
	}

	queue := q.store.Queue()
	qLen := len(queue)

	switch {
	case keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "j",
		keyMsg.Type == tea.KeyDown:
		if q.cursor < qLen-1 {
			q.cursor++
			q.ensureCursorVisible()
		}

	case keyMsg.Type == tea.KeyRunes && string(keyMsg.Runes) == "k",
		keyMsg.Type == tea.KeyUp:
		if q.cursor > 0 {
			q.cursor--
			q.ensureCursorVisible()
		}

	case keyMsg.Type == tea.KeyEnter:
		if qLen > 0 && q.cursor < qLen {
			uri := queue[q.cursor].URI
			return q, func() tea.Msg {
				return PlayTrackMsg{TrackURI: uri}
			}
		}
	}

	return q, nil
}

// View renders the queue pane content. It is pure — reads store and returns a string.
func (q *QueuePane) View() string {
	headerStyle := lipgloss.NewStyle().
		Foreground(q.theme.SectionHeader()).
		Bold(true)

	dividerStyle := lipgloss.NewStyle().
		Foreground(q.theme.TextMuted())

	mutedStyle := lipgloss.NewStyle().
		Foreground(q.theme.TextMuted())

	primaryStyle := lipgloss.NewStyle().
		Foreground(q.theme.TextPrimary())

	secondaryStyle := lipgloss.NewStyle().
		Foreground(q.theme.TextSecondary())

	playingStyle := lipgloss.NewStyle().
		Foreground(q.theme.PlayingIndicator())

	selectedBgStyle := lipgloss.NewStyle().
		Background(q.theme.SelectedBg()).
		Foreground(q.theme.SelectedFg())

	divider := dividerStyle.Render(strings.Repeat("─", q.dividerWidth()))

	ps := q.store.PlaybackState()
	queue := q.store.Queue()

	var lines []string

	// Header — show repeat indicator when repeat-track is active.
	headerText := "QUEUE"
	if ps != nil && ps.RepeatState == "track" {
		headerText = "QUEUE [repeat track]"
	} else if ps != nil && ps.RepeatState == "context" {
		headerText = "QUEUE [repeat]"
	}
	lines = append(lines, headerStyle.Render(headerText))
	lines = append(lines, divider)
	lines = append(lines, "")

	// Empty state
	if ps == nil || ps.Item == nil {
		emptyMsg := mutedStyle.Render("Queue is empty")
		lines = append(lines, emptyMsg)
		return strings.Join(lines, "\n")
	}

	// NOW section
	lines = append(lines, playingStyle.Render("▶ NOW"))
	lines = append(lines, "  "+primaryStyle.Render(ps.Item.Name))
	artistName := ""
	if len(ps.Item.Artists) > 0 {
		artistName = ps.Item.Artists[0].Name
	}
	lines = append(lines, "  "+secondaryStyle.Render(artistName))
	lines = append(lines, "")
	lines = append(lines, divider)

	// NEXT UP section
	lines = append(lines, headerStyle.Render("NEXT UP"))
	lines = append(lines, "")

	if len(queue) == 0 {
		lines = append(lines, "  "+mutedStyle.Render("Queue is empty"))
	} else {
		visibleCount := q.visibleTrackCount()
		endIdx := q.scrollOffset + visibleCount
		if endIdx > len(queue) {
			endIdx = len(queue)
		}

		// Scroll up indicator
		if q.scrollOffset > 0 {
			lines = append(lines, "  "+mutedStyle.Render("▲ more above"))
		}

		for i := q.scrollOffset; i < endIdx; i++ {
			track := queue[i]
			numStr := fmt.Sprintf("%d", i+1)
			trackName := track.Name
			artistStr := ""
			if len(track.Artists) > 0 {
				artistStr = track.Artists[0].Name
			}

			numPadded := fmt.Sprintf("%-3s", numStr)
			trackLine := numPadded + trackName
			artistLine := "   " + artistStr

			if i == q.cursor {
				lines = append(lines, selectedBgStyle.Render(trackLine))
				lines = append(lines, selectedBgStyle.Render(artistLine))
			} else {
				lines = append(lines, "  "+primaryStyle.Render(fmt.Sprintf("%s %s", strings.TrimSpace(numPadded), trackName)))
				lines = append(lines, "   "+secondaryStyle.Render(artistStr))
			}
		}

		// Scroll down indicator
		if endIdx < len(queue) {
			lines = append(lines, "  "+mutedStyle.Render("▼ more below"))
		}
	}

	lines = append(lines, "")
	lines = append(lines, divider)

	// Footer: item count
	countMsg := fmt.Sprintf("%d tracks remaining", len(queue))
	lines = append(lines, mutedStyle.Render(countMsg))
	lines = append(lines, "")

	return strings.Join(lines, "\n")
}

// visibleTrackCount returns the number of tracks that fit in the visible area.
// Each track takes 2 lines (name + artist). Header/footer/NOW takes ~10 lines.
func (q *QueuePane) visibleTrackCount() int {
	if q.height <= 0 {
		return 10 // default for tests
	}
	available := q.height - 14 // header + NOW + dividers + footer
	if available <= 0 {
		return 3
	}
	return available / 2 // 2 lines per track
}

// ensureCursorVisible adjusts scrollOffset so the cursor is within the visible window.
func (q *QueuePane) ensureCursorVisible() {
	visible := q.visibleTrackCount()
	if q.cursor < q.scrollOffset {
		q.scrollOffset = q.cursor
	}
	if q.cursor >= q.scrollOffset+visible {
		q.scrollOffset = q.cursor - visible + 1
	}
	if q.scrollOffset < 0 {
		q.scrollOffset = 0
	}
}

// dividerWidth returns the width to use for divider lines.
func (q *QueuePane) dividerWidth() int {
	if q.width > 4 {
		return q.width - 4
	}
	return 20
}
