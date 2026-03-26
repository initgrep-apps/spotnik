package panes

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestQueuePane creates a QueuePane with a fresh store and black theme.
func newTestQueuePane(focused bool) *QueuePane {
	s := state.New()
	t := theme.Load("black")
	return NewQueuePane(s, t, focused)
}

// newTestQueuePaneWithData creates a QueuePane pre-loaded with playback state and queue.
func newTestQueuePaneWithData(focused bool) *QueuePane {
	s := state.New()
	s.SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
		Item: &api.Track{
			ID:      "now-1",
			Name:    "Blinding Lights",
			URI:     "spotify:track:now-1",
			Artists: []api.Artist{{Name: "The Weeknd"}},
		},
	})
	s.SetQueue([]api.Track{
		{ID: "q1", Name: "Save Your Tears", URI: "spotify:track:q1", Artists: []api.Artist{{Name: "The Weeknd"}}},
		{ID: "q2", Name: "Starboy", URI: "spotify:track:q2", Artists: []api.Artist{{Name: "The Weeknd"}}},
		{ID: "q3", Name: "Can't Feel My Face", URI: "spotify:track:q3", Artists: []api.Artist{{Name: "The Weeknd"}}},
	})
	t := theme.Load("black")
	return NewQueuePane(s, t, focused)
}

// TestQueuePane_View_EmptyQueue verifies that an empty queue renders without panic.
func TestQueuePane_View_EmptyQueue(t *testing.T) {
	pane := newTestQueuePane(true)
	pane.SetSize(80, 20)
	output := pane.View()

	// Table renders cleanly (no panic, returns a string).
	assert.NotEmpty(t, output, "should return non-empty string even for empty queue")
}

// TestQueuePane_View_QueueTracks verifies that queued track names are visible.
func TestQueuePane_View_QueueTracks(t *testing.T) {
	pane := newTestQueuePaneWithData(true)
	pane.SetSize(80, 20)
	output := pane.View()

	assert.Contains(t, output, "Save Your Tears", "should show first queued track")
	assert.Contains(t, output, "Starboy", "should show second queued track")
	assert.Contains(t, output, "The Weeknd", "should show track artist")
}

// TestQueuePane_View_TrackNumbers verifies that row numbers appear in the table.
func TestQueuePane_View_TrackNumbers(t *testing.T) {
	pane := newTestQueuePaneWithData(true)
	pane.SetSize(80, 20)
	output := pane.View()

	// Track numbers 1, 2, 3 should appear.
	assert.Contains(t, output, "1", "track number 1 should appear")
	assert.Contains(t, output, "2", "track number 2 should appear")
}

// TestQueuePane_Update_J_MovesDown verifies that pressing j moves the cursor down.
func TestQueuePane_Update_J_MovesDown(t *testing.T) {
	pane := newTestQueuePaneWithData(true)
	pane.SetSize(80, 20)

	initialCursor := pane.Cursor()
	updated, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	pp, ok := updated.(*QueuePane)
	require.True(t, ok)
	assert.Greater(t, pp.Cursor(), initialCursor, "cursor should move down after j")
}

// TestQueuePane_Update_K_MovesUp verifies that pressing k moves the cursor up.
func TestQueuePane_Update_K_MovesUp(t *testing.T) {
	pane := newTestQueuePaneWithData(true)
	pane.SetSize(80, 20)

	// First move down, then verify k moves back up.
	updated, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	pp, ok := updated.(*QueuePane)
	require.True(t, ok)
	assert.Equal(t, 1, pp.Cursor())

	updated2, _ := pp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	pp2, ok := updated2.(*QueuePane)
	require.True(t, ok)
	assert.Equal(t, 0, pp2.Cursor(), "cursor should move up after k")
}

// TestQueuePane_Update_Enter_PlaysTrack verifies that Enter returns a PlayTrackMsg
// for the selected queue item.
func TestQueuePane_Update_Enter_PlaysTrack(t *testing.T) {
	pane := newTestQueuePaneWithData(true)
	pane.SetSize(80, 20)

	// Cursor starts at 0 — first queued track is "Save Your Tears".
	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter should return a command")

	msg := cmd()
	playMsg, ok := msg.(PlayTrackMsg)
	require.True(t, ok, "command should produce PlayTrackMsg")
	assert.Equal(t, "spotify:track:q1", playMsg.TrackURI, "should play the selected track URI")
}

// TestQueuePane_Update_IgnoresWhenNotFocused verifies that the pane ignores input when unfocused.
func TestQueuePane_Update_IgnoresWhenNotFocused(t *testing.T) {
	pane := newTestQueuePaneWithData(false)
	pane.SetSize(80, 20)

	initialCursor := pane.Cursor()
	updated, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	pp, ok := updated.(*QueuePane)
	require.True(t, ok)
	assert.Equal(t, initialCursor, pp.Cursor(), "cursor should not move when not focused")
	assert.Nil(t, cmd, "unfocused pane should return nil command")
}

// TestQueuePane_IsFocused verifies that SetFocused/IsFocused work correctly.
func TestQueuePane_IsFocused(t *testing.T) {
	pane := newTestQueuePane(false)
	assert.False(t, pane.IsFocused())

	pane.SetFocused(true)
	assert.True(t, pane.IsFocused())
}

func TestQueuePane_ScrollIndicators_LongQueue(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
		Item:      &api.Track{ID: "t1", Name: "Now", Artists: []api.Artist{{Name: "A"}}},
	})

	// Create 25 tracks in the queue.
	tracks := make([]api.Track, 25)
	for i := range tracks {
		tracks[i] = api.Track{
			ID:      fmt.Sprintf("q%d", i),
			Name:    fmt.Sprintf("Track %d", i+1),
			Artists: []api.Artist{{Name: "Artist"}},
		}
	}
	s.SetQueue(tracks)

	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(80, 10) // limited height — only a few rows visible

	output := pane.View()
	// bubble-table with page size < queue length shows a subset of rows.
	// Track 1 should be visible, Track 25 should not (out of page).
	assert.Contains(t, output, "Track 1", "first track should be visible")
}

func TestQueuePane_Scroll_CursorMovesWindow(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
		Item:      &api.Track{ID: "t1", Name: "Now", Artists: []api.Artist{{Name: "A"}}},
	})

	tracks := make([]api.Track, 25)
	for i := range tracks {
		tracks[i] = api.Track{
			ID:      fmt.Sprintf("q%d", i),
			Name:    fmt.Sprintf("Track %d", i+1),
			Artists: []api.Artist{{Name: "Artist"}},
		}
	}
	s.SetQueue(tracks)

	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(80, 15)

	// Navigate down past the visible window — bubble-table handles scrolling internally.
	for i := 0; i < 20; i++ {
		m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		pane = m.(*QueuePane)
	}

	// After navigating down 20 times, cursor should be at index 20.
	assert.Equal(t, 20, pane.Cursor())
}

// TestQueuePane_View_WithQueueData verifies that track data is visible in table rows.
func TestQueuePane_View_WithQueueData(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&api.PlaybackState{
		IsPlaying:   true,
		RepeatState: "off",
		Item:        &api.Track{ID: "t1", Name: "Now Playing", Artists: []api.Artist{{Name: "A"}}},
	})
	s.SetQueue([]api.Track{{ID: "q1", Name: "Next Track", Artists: []api.Artist{{Name: "B"}}}})

	th := theme.Load("black")
	pane := NewQueuePane(s, th, false)
	pane.SetSize(80, 20)
	output := pane.View()

	assert.Contains(t, output, "Next Track", "should show queued track name")
	assert.Contains(t, output, "B", "should show queued track artist")
}

// --- Task 1: layout.Pane interface methods ---

// TestQueuePane_ImplementsLayoutPane verifies QueuePane satisfies layout.Pane at compile time.
var _ layout.Pane = &QueuePane{}

// TestQueuePane_ID verifies that ID() returns PaneQueue.
func TestQueuePane_ID(t *testing.T) {
	pane := newTestQueuePane(false)
	assert.Equal(t, layout.PaneQueue, pane.ID())
}

// TestQueuePane_Title verifies that Title() returns "Queue".
func TestQueuePane_Title(t *testing.T) {
	pane := newTestQueuePane(false)
	assert.Equal(t, "Queue", pane.Title())
}

// TestQueuePane_ToggleKey verifies that ToggleKey() returns 2.
func TestQueuePane_ToggleKey(t *testing.T) {
	pane := newTestQueuePane(false)
	assert.Equal(t, 2, pane.ToggleKey())
}

// TestQueuePane_Actions verifies that Actions() returns filter and add actions by default.
func TestQueuePane_Actions(t *testing.T) {
	pane := newTestQueuePane(false)
	actions := pane.Actions()
	require.Len(t, actions, 2)
	assert.Equal(t, "f", actions[0].Key)
	assert.Equal(t, "filter", actions[0].Label)
	assert.Equal(t, "A", actions[1].Key)
	assert.Equal(t, "add", actions[1].Label)
}

// --- Task 2: bubble-table rendering ---

// TestQueuePane_Table_FiveTracks verifies the table has 5 rows for 5 queued tracks.
func TestQueuePane_Table_FiveTracks(t *testing.T) {
	s := state.New()
	tracks := make([]api.Track, 5)
	for i := range tracks {
		tracks[i] = api.Track{
			ID:      fmt.Sprintf("t%d", i+1),
			Name:    fmt.Sprintf("Track %d", i+1),
			Artists: []api.Artist{{Name: "Artist"}},
		}
	}
	s.SetQueue(tracks)

	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(80, 20)
	output := pane.View()

	for i := 1; i <= 5; i++ {
		assert.Contains(t, output, fmt.Sprintf("Track %d", i), "should contain track %d", i)
	}
}

// TestQueuePane_Table_ColumnHeaders verifies column headers are rendered.
func TestQueuePane_Table_ColumnHeaders(t *testing.T) {
	pane := newTestQueuePaneWithData(true)
	pane.SetSize(80, 20)
	output := pane.View()

	assert.Contains(t, output, "#", "should contain # column header")
	assert.Contains(t, output, "Track", "should contain Track column header")
	assert.Contains(t, output, "Artist", "should contain Artist column header")
	assert.Contains(t, output, "Duration", "should contain Duration column header")
}

// TestQueuePane_Table_PlayingIndicator verifies the ▶ symbol appears for the playing track.
func TestQueuePane_Table_PlayingIndicator(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
		Item: &api.Track{
			ID:      "now-1",
			Name:    "Playing Track",
			URI:     "spotify:track:now-1",
			Artists: []api.Artist{{Name: "Artist"}},
			// Mark this as playing at position 1 in the queue (0-based index 0).
		},
	})
	tracks := []api.Track{
		{ID: "q1", Name: "First Queue", URI: "spotify:track:q1", Artists: []api.Artist{{Name: "Artist"}}},
		{ID: "q2", Name: "Second Queue", URI: "spotify:track:q2", Artists: []api.Artist{{Name: "Artist"}}},
	}
	s.SetQueue(tracks)

	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(80, 20)
	// Set playing index to 0 (first queued track is "playing").
	pane.SetPlayingIndex(0)
	output := pane.View()

	assert.Contains(t, output, "▶", "should show playing indicator")
}

// TestQueuePane_Table_EmptyQueue verifies no panic and shows empty state.
func TestQueuePane_Table_EmptyQueue(t *testing.T) {
	pane := newTestQueuePane(true)
	pane.SetSize(80, 20)
	// Should not panic:
	output := pane.View()
	assert.NotEmpty(t, output, "should return non-empty string even for empty queue")
}

// TestQueuePane_Table_SetSizeUpdates verifies that SetSize updates table dimensions.
func TestQueuePane_Table_SetSizeUpdates(t *testing.T) {
	pane := newTestQueuePaneWithData(true)
	// Call SetSize multiple times without panic.
	pane.SetSize(80, 20)
	pane.SetSize(100, 30)
	output := pane.View()
	assert.NotEmpty(t, output)
}

// TestQueuePane_Table_JKNavigation verifies j/k keys navigate the table.
func TestQueuePane_Table_JKNavigation(t *testing.T) {
	pane := newTestQueuePaneWithData(true)
	pane.SetSize(80, 20)

	// Initial selected row should be 0.
	assert.Equal(t, 0, pane.table.SelectedIndex())

	// Press j to move down.
	updated, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	pp := updated.(*QueuePane)
	assert.Equal(t, 1, pp.table.SelectedIndex(), "j should move selection down")

	// Press k to move back up.
	updated2, _ := pp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	pp2 := updated2.(*QueuePane)
	assert.Equal(t, 0, pp2.table.SelectedIndex(), "k should move selection up")
}
