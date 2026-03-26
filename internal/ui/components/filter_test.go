package components_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/stretchr/testify/assert"
)

func TestNewFilter_StartsInactive(t *testing.T) {
	f := components.NewFilter(testTheme())
	assert.False(t, f.IsActive())
}

func TestFilter_QueryEmptyInitially(t *testing.T) {
	f := components.NewFilter(testTheme())
	assert.Equal(t, "", f.Query())
}

func TestFilter_ToggleActivates(t *testing.T) {
	f := components.NewFilter(testTheme())
	f.Toggle()
	assert.True(t, f.IsActive())
}

func TestFilter_ToggleTwiceDeactivates(t *testing.T) {
	f := components.NewFilter(testTheme())
	f.Toggle()
	f.Toggle()
	assert.False(t, f.IsActive())
}

func TestFilter_MatchesCaseInsensitive(t *testing.T) {
	f := components.NewFilter(testTheme())
	f.Toggle()

	// Simulate typing via Update
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b', 'l', 'i', 'n', 'd'}})

	assert.True(t, f.Matches("Blinding Lights"))
}

func TestFilter_MatchesFalseForNonMatching(t *testing.T) {
	f := components.NewFilter(testTheme())
	f.Toggle()
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x', 'y', 'z'}})

	assert.False(t, f.Matches("Blinding Lights"))
}

func TestFilter_MatchesTrueWhenInactive(t *testing.T) {
	f := components.NewFilter(testTheme())
	// Filter is inactive, should always match
	assert.True(t, f.Matches("Blinding Lights"))
	assert.True(t, f.Matches("anything"))
}

func TestFilter_MatchesAnyReturnsTrueIfAnyMatches(t *testing.T) {
	f := components.NewFilter(testTheme())
	f.Toggle()
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a', 'r', 't', 'i', 's', 't'}})

	// "artist" matches "Jazz Artist" but not "Rock Song"
	assert.True(t, f.MatchesAny("Rock Song", "Jazz Artist"))
}

func TestFilter_MatchesAnyReturnsFalseIfNoneMatch(t *testing.T) {
	f := components.NewFilter(testTheme())
	f.Toggle()
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x', 'x', 'x'}})

	assert.False(t, f.MatchesAny("Rock Song", "Jazz Artist"))
}

func TestFilter_BorderLabelWhenActive(t *testing.T) {
	f := components.NewFilter(testTheme())
	f.Toggle()

	// Simulate typing "rock"
	for _, r := range []rune{'r', 'o', 'c', 'k'} {
		f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	label := f.BorderLabel()
	assert.Equal(t, `filtering: "rock"`, label)
}

func TestFilter_BorderLabelEmptyWhenInactive(t *testing.T) {
	f := components.NewFilter(testTheme())
	assert.Equal(t, "", f.BorderLabel())
}

func TestFilter_ViewEmptyWhenInactive(t *testing.T) {
	f := components.NewFilter(testTheme())
	assert.Equal(t, "", f.View(40))
}

func TestFilter_ViewNonEmptyWhenActive(t *testing.T) {
	f := components.NewFilter(testTheme())
	f.Toggle()
	f.SetWidth(40)
	view := f.View(40)
	assert.NotEmpty(t, view)
}

func TestFilter_EscDeactivatesFilter(t *testing.T) {
	f := components.NewFilter(testTheme())
	f.Toggle()
	assert.True(t, f.IsActive())

	// Type some query text
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r', 'o', 'c', 'k'}})

	// Press Esc
	f.Update(tea.KeyMsg{Type: tea.KeyEsc})

	assert.False(t, f.IsActive())
	// Query should be cleared
	assert.Equal(t, "", f.Query())
}

func TestFilter_EnterDeactivatesButPreservesQuery(t *testing.T) {
	f := components.NewFilter(testTheme())
	f.Toggle()

	// Type "rock"
	for _, r := range []rune{'r', 'o', 'c', 'k'} {
		f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Press Enter
	f.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Filter is deactivated but query is preserved
	assert.False(t, f.IsActive())
	assert.Equal(t, "rock", f.Query())
}

func TestFilter_EmptyQueryMatchesEverything(t *testing.T) {
	f := components.NewFilter(testTheme())
	f.Toggle()
	// No query typed; empty query should match anything
	assert.True(t, f.Matches("anything"))
	assert.True(t, f.Matches(""))
}

func TestFilter_MatchesAnyEmptyQueryReturnsTrue(t *testing.T) {
	f := components.NewFilter(testTheme())
	// Inactive filter with empty query — MatchesAny must return true
	// even with zero arguments, consistent with Matches() contract.
	assert.True(t, f.MatchesAny())
	assert.True(t, f.MatchesAny("anything"))
}

func TestFilter_MatchesAnyZeroArgsWithQueryReturnsFalse(t *testing.T) {
	f := components.NewFilter(testTheme())
	f.Toggle()
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r', 'o', 'c', 'k'}})
	// Active query with zero args — nothing can match
	assert.False(t, f.MatchesAny())
}
