package panes

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time check: RecentlyPlayedPane implements layout.Pane.
var _ layout.Pane = &RecentlyPlayedPane{}

// newTestRecentlyPlayedPane creates a pane wired to a store pre-populated with test data.
func newTestRecentlyPlayedPane() (*RecentlyPlayedPane, *state.Store) {
	st := state.New()
	now := time.Now()
	st.SetRecentlyPlayed([]domain.PlayHistory{
		{
			Track:    domain.Track{ID: "t1", Name: "Track One", URI: "spotify:track:t1", Artists: []domain.Artist{{Name: "Artist A"}}},
			PlayedAt: now.Add(-2 * time.Minute).Format(time.RFC3339),
		},
		{
			Track:    domain.Track{ID: "t2", Name: "Track Two", URI: "spotify:track:t2", Artists: []domain.Artist{{Name: "Artist B"}}},
			PlayedAt: now.Add(-1 * time.Hour).Format(time.RFC3339),
		},
		{
			Track:    domain.Track{ID: "t3", Name: "Another Song", URI: "spotify:track:t3", Artists: []domain.Artist{{Name: "Band C"}}},
			PlayedAt: now.Add(-3 * 24 * time.Hour).Format(time.RFC3339),
		},
	})
	th := theme.Load("black")
	pane := NewRecentlyPlayedPane(st, th, false)
	pane.SetSize(120, 20)
	return pane, st
}

// TestRecentlyPlayedPane_ImplementsLayoutPane verifies the compile-time check.
func TestRecentlyPlayedPane_ImplementsLayoutPane(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	assert.Equal(t, layout.PaneRecentlyPlayed, pane.ID())
}

func TestRecentlyPlayedPane_Metadata(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	assert.Equal(t, layout.PaneRecentlyPlayed, pane.ID())
	assert.Equal(t, "Recently Played", pane.Title())
	assert.Equal(t, 6, pane.ToggleKey())
}

func TestRecentlyPlayedPane_Actions_Default(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	actions := pane.Actions()
	require.Len(t, actions, 1)
	assert.Equal(t, "f", actions[0].Key)
}

func TestRecentlyPlayedPane_Actions_FilterActive(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	pane.SetFocused(true)
	pane.SetSize(120, 20)
	// Toggle filter on
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck
	actions := pane.Actions()
	require.Len(t, actions, 1)
	assert.Equal(t, "Esc", actions[0].Key)
}

func TestRecentlyPlayedPane_RendersTrackNames(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	pane.SetFocused(true)
	view := pane.View()
	assert.Contains(t, view, "Track One")
	assert.Contains(t, view, "Track Two")
	assert.Contains(t, view, "Another Song")
}

func TestRecentlyPlayedPane_RendersRelativeTime(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	view := pane.View()
	// Track One played 2 min ago
	assert.Contains(t, view, "min ago")
	// Track Two played 1 hr ago
	assert.Contains(t, view, "hr ago")
	// Track Three played 3 days ago
	assert.Contains(t, view, "days ago")
}

func TestRecentlyPlayedPane_EnterEmitsPlayTrackMsg(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	pane.SetFocused(true)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	playMsg, ok := msg.(PlayTrackMsg)
	require.True(t, ok)
	assert.Equal(t, "spotify:track:t1", playMsg.TrackURI)
}

func TestRecentlyPlayedPane_FilterByTrackName(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	pane.SetFocused(true)

	// Activate filter
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck

	// Type "another"
	for _, r := range "another" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	view := pane.View()
	assert.Contains(t, view, "Another Song")
	assert.NotContains(t, strings.ToLower(view), "track one")
}

func TestRecentlyPlayedPane_FilterByArtistName(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	pane.SetFocused(true)

	// Activate filter
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}) //nolint:errcheck

	// Type "Artist A"
	for _, r := range "Artist A" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) //nolint:errcheck
	}

	view := pane.View()
	assert.Contains(t, view, "Track One")
	assert.NotContains(t, view, "Track Two")
}

func TestRecentlyPlayedPane_EmptyData(t *testing.T) {
	st := state.New()
	th := theme.Load("black")
	pane := NewRecentlyPlayedPane(st, th, false)
	pane.SetSize(120, 20)
	// Should not panic and should show empty state message.
	view := pane.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "No recently played tracks")
}

func TestRecentlyPlayedPane_RefreshRows(t *testing.T) {
	pane, st := newTestRecentlyPlayedPane()
	now := time.Now()
	// Update store directly
	st.SetRecentlyPlayed([]domain.PlayHistory{
		{
			Track:    domain.Track{ID: "t99", Name: "New Track", URI: "spotify:track:t99", Artists: []domain.Artist{{Name: "New Artist"}}},
			PlayedAt: now.Add(-10 * time.Minute).Format(time.RFC3339),
		},
	})
	pane.RefreshRows()
	view := pane.View()
	assert.Contains(t, view, "New Track")
}

func TestRecentlyPlayedPane_RecentlyPlayedLoadedMsg(t *testing.T) {
	pane, st := newTestRecentlyPlayedPane()
	now := time.Now()
	// Simulate a RecentlyPlayedLoadedMsg (app writes store then sends msg)
	st.SetRecentlyPlayed([]domain.PlayHistory{
		{
			Track:    domain.Track{ID: "t55", Name: "Loaded Track", URI: "spotify:track:t55", Artists: []domain.Artist{{Name: "Loaded Artist"}}},
			PlayedAt: now.Add(-5 * time.Minute).Format(time.RFC3339),
		},
	})
	pane.Update(RecentlyPlayedLoadedMsg{}) //nolint:errcheck
	view := pane.View()
	assert.Contains(t, view, "Loaded Track")
}

func TestRecentlyPlayedPane_NotFocusedIgnoresKeys(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	// pane is not focused — Enter should not emit a command
	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
}

func TestRecentlyPlayedPane_SetFocused(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	assert.False(t, pane.IsFocused())
	pane.SetFocused(true)
	assert.True(t, pane.IsFocused())
	pane.SetFocused(false)
	assert.False(t, pane.IsFocused())
}

func TestRecentlyPlayedPane_Init(t *testing.T) {
	pane, _ := newTestRecentlyPlayedPane()
	cmd := pane.Init()
	assert.Nil(t, cmd)
}
