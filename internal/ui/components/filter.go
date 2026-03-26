package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
)

// filterCharLimit is the maximum number of characters allowed in the filter query.
const filterCharLimit = 50

// Filter provides in-pane text filtering using bubbles/textinput.
// It is not a tea.Model itself — the pane Update() calls Filter.Update()
// and the pane View() embeds Filter.View() in its rendered output.
type Filter struct {
	input  textinput.Model
	active bool
	query  string
	theme  theme.Theme
}

// NewFilter creates a Filter configured with the given theme. The filter
// starts in an inactive state; call Toggle() to activate it.
func NewFilter(t theme.Theme) *Filter {
	ti := textinput.New()
	ti.Placeholder = "type to filter..."
	ti.CharLimit = filterCharLimit
	ti.Prompt = "> "
	ti.TextStyle = lipgloss.NewStyle().Foreground(t.TextPrimary())
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(t.TextMuted())

	return &Filter{
		input: ti,
		theme: t,
	}
}

// Toggle activates or deactivates the filter.
// Activating focuses the text input. Deactivating blurs the input and
// clears the query.
func (f *Filter) Toggle() {
	f.active = !f.active
	if f.active {
		f.input.Focus()
	} else {
		f.input.Blur()
		f.input.Reset()
		f.query = ""
	}
}

// IsActive returns whether the filter is currently accepting input.
func (f *Filter) IsActive() bool {
	return f.active
}

// Query returns the current filter text (the last committed or in-progress query).
func (f *Filter) Query() string {
	return f.query
}

// Matches returns true if text contains the filter query as a case-insensitive
// substring. Returns true unconditionally when the query is empty (including
// when the filter is inactive).
func (f *Filter) Matches(text string) bool {
	if f.query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(text), strings.ToLower(f.query))
}

// MatchesAny returns true if at least one of the provided strings satisfies Matches.
func (f *Filter) MatchesAny(texts ...string) bool {
	for _, t := range texts {
		if f.Matches(t) {
			return true
		}
	}
	return false
}

// Update handles input events when the filter is active.
//   - Esc: deactivates the filter and clears the query
//   - Enter: deactivates the filter but preserves the current query
//   - All other keys: forwarded to the internal textinput
//
// Returns a tea.Cmd if the textinput produced one (e.g. cursor blink).
// Returns nil when the filter is inactive.
func (f *Filter) Update(msg tea.Msg) tea.Cmd {
	if !f.active {
		return nil
	}

	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.Type {
		case tea.KeyEsc:
			f.input.Blur()
			f.input.Reset()
			f.query = ""
			f.active = false
			return nil
		case tea.KeyEnter:
			// Preserve the current query but deactivate the input.
			f.query = f.input.Value()
			f.input.Blur()
			f.active = false
			return nil
		}
	}

	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	f.query = f.input.Value()
	return cmd
}

// SetWidth updates the visible width of the filter input bar.
// Call this from the pane's SetSize so View() remains side-effect-free.
func (f *Filter) SetWidth(width int) {
	// input.Width is the number of visible columns for the text portion;
	// subtract prompt (2) and outer padding (2) so the bar fits in the pane.
	f.input.Width = width - 4
}

// View renders the filter input bar at the given width.
// Returns an empty string when the filter is inactive.
func (f *Filter) View(width int) string {
	if !f.active {
		return ""
	}

	barStyle := lipgloss.NewStyle().
		Background(f.theme.SurfaceAlt()).
		Width(width).
		Padding(0, 1)

	return barStyle.Render(f.input.View())
}

// BorderLabel returns the text to embed in the pane border when the filter has
// an active query. Returns an empty string when inactive or when the query is empty.
// e.g., `filtering: "rock"`.
func (f *Filter) BorderLabel() string {
	if f.query == "" {
		return ""
	}
	return fmt.Sprintf("filtering: %q", f.query)
}
