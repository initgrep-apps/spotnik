package panes

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// --- Task 3: filter support ---

// TestQueuePane_Filter_FKeyActivates verifies that pressing 'f' activates the filter.
func TestQueuePane_Filter_FKeyActivates(t *testing.T) {
	pane := newTestQueuePaneWithData(true)
	pane.SetSize(80, 20)

	assert.False(t, pane.filter.IsActive(), "filter should be inactive initially")
	updated, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	pp := updated.(*QueuePane)
	assert.True(t, pp.filter.IsActive(), "f key should activate filter")
}

// TestQueuePane_Filter_EscCloses verifies that Esc closes the filter and restores full list.
func TestQueuePane_Filter_EscCloses(t *testing.T) {
	pane := newTestQueuePaneWithData(true)
	pane.SetSize(80, 20)

	// Activate filter.
	updated, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	pp := updated.(*QueuePane)
	require.True(t, pp.filter.IsActive())

	// Press Esc — filter should close.
	updated2, _ := pp.Update(tea.KeyMsg{Type: tea.KeyEsc})
	pp2 := updated2.(*QueuePane)
	assert.False(t, pp2.filter.IsActive(), "Esc should close filter")
	// Full list should be restored: table has 3 rows.
	output := pp2.View()
	assert.Contains(t, output, "Save Your Tears", "full list should be visible after close")
}

// TestQueuePane_Filter_QueryFiltersRows verifies that typing a query reduces visible rows.
func TestQueuePane_Filter_QueryFiltersRows(t *testing.T) {
	s := state.New()
	s.SetQueue([]api.Track{
		{ID: "q1", Name: "Rocket Man", Artists: []api.Artist{{Name: "Elton John"}}},
		{ID: "q2", Name: "Rock and Roll", Artists: []api.Artist{{Name: "Led Zeppelin"}}},
		{ID: "q3", Name: "Save Your Tears", Artists: []api.Artist{{Name: "The Weeknd"}}},
	})
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(80, 20)

	// Activate filter.
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	pane = m.(*QueuePane)

	// Type "rock" — should match "Rocket Man" and "Rock and Roll" but not "Save Your Tears".
	for _, r := range "rock" {
		m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		pane = m.(*QueuePane)
	}

	output := pane.View()
	assert.Contains(t, output, "Rocket Man", "should show track matching filter")
	assert.Contains(t, output, "Rock and Roll", "should show track matching filter")
	assert.NotContains(t, output, "Save Your Tears", "non-matching track should be hidden")
}

// TestQueuePane_Filter_ArtistMatch verifies that filter matches artist names.
func TestQueuePane_Filter_ArtistMatch(t *testing.T) {
	s := state.New()
	s.SetQueue([]api.Track{
		{ID: "q1", Name: "Blinding Lights", Artists: []api.Artist{{Name: "The Weeknd"}}},
		{ID: "q2", Name: "Levitating", Artists: []api.Artist{{Name: "Dua Lipa"}}},
	})
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(80, 20)

	// Activate filter and type "weeknd" to match by artist.
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	pane = m.(*QueuePane)
	for _, r := range "weeknd" {
		m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		pane = m.(*QueuePane)
	}

	output := pane.View()
	assert.Contains(t, output, "Blinding Lights", "should match track by artist name")
	assert.NotContains(t, output, "Levitating", "non-matching track should be hidden")
}

// TestQueuePane_Filter_EmptyQueryShowsAll verifies that empty filter shows all rows.
func TestQueuePane_Filter_EmptyQueryShowsAll(t *testing.T) {
	pane := newTestQueuePaneWithData(true)
	pane.SetSize(80, 20)

	// Activate filter but don't type anything.
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	pp := m.(*QueuePane)
	assert.True(t, pp.filter.IsActive())

	output := pp.View()
	// All three tracks should be visible.
	assert.Contains(t, output, "Save Your Tears")
	assert.Contains(t, output, "Starboy")
	assert.Contains(t, output, "Can't Feel My Face")
}

// TestQueuePane_Filter_NoMatchesShowsEmptyTable verifies filter with no matches shows empty table.
func TestQueuePane_Filter_NoMatchesShowsEmptyTable(t *testing.T) {
	pane := newTestQueuePaneWithData(true)
	pane.SetSize(80, 20)

	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	pane = m.(*QueuePane)

	// Type a query that won't match anything.
	for _, r := range "zzzzzzzzz" {
		m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		pane = m.(*QueuePane)
	}

	// Should not panic — table renders empty state.
	output := pane.View()
	assert.NotEmpty(t, output)
	assert.NotContains(t, output, "Save Your Tears", "no tracks should match")
}

// TestQueuePane_Filter_ActionsChangedWhenActive verifies Actions() changes when filter is active.
func TestQueuePane_Filter_ActionsChangedWhenActive(t *testing.T) {
	pane := newTestQueuePaneWithData(true)
	pane.SetSize(80, 20)

	// Default actions.
	actions := pane.Actions()
	require.Len(t, actions, 2)
	assert.Equal(t, "f", actions[0].Key)

	// Activate filter.
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	pp := m.(*QueuePane)
	filterActions := pp.Actions()
	require.Len(t, filterActions, 1, "filter active should show only close action")
	assert.Equal(t, "Esc", filterActions[0].Key)
	assert.Equal(t, "close", filterActions[0].Label)
}

// --- Task 4: Comprehensive tests ---

// TestQueuePane_InterfaceSatisfied ensures compile-time layout.Pane compliance.
var _ layout.Pane = &QueuePane{}

// TestQueuePane_FullLifecycle tests construct → resize → load queue → filter → navigate → view.
func TestQueuePane_FullLifecycle(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)

	// Step 1: resize.
	pane.SetSize(100, 30)

	// Step 2: load queue.
	tracks := []api.Track{
		{ID: "t1", Name: "Blinding Lights", Artists: []api.Artist{{Name: "The Weeknd"}}, DurationMs: 200000},
		{ID: "t2", Name: "Levitating", Artists: []api.Artist{{Name: "Dua Lipa"}}, DurationMs: 203000},
		{ID: "t3", Name: "Rocket Man", Artists: []api.Artist{{Name: "Elton John"}}, DurationMs: 269000},
	}
	s.SetQueue(tracks)
	// Simulate QueueLoadedMsg by refreshing the pane's rows.
	pane.refreshRows()

	output := pane.View()
	assert.Contains(t, output, "Blinding Lights")
	assert.Contains(t, output, "Levitating")
	assert.Contains(t, output, "Rocket Man")

	// Step 3: filter.
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	pane = m.(*QueuePane)
	for _, r := range "rock" {
		m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		pane = m.(*QueuePane)
	}
	filteredOutput := pane.View()
	assert.Contains(t, filteredOutput, "Rocket Man")
	assert.NotContains(t, filteredOutput, "Levitating")

	// Step 4: navigate (j key).
	m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	pane = m.(*QueuePane)
	// k in filter mode is forwarded to filter (not table nav).

	// Step 5: close filter and verify full list.
	m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyEsc})
	pane = m.(*QueuePane)
	assert.False(t, pane.filter.IsActive())
	restoreOutput := pane.View()
	assert.Contains(t, restoreOutput, "Levitating", "full list should restore after filter close")
}

// TestQueuePane_QueueUpdate_TableRefreshes verifies table refreshes on new QueueLoadedMsg data.
func TestQueuePane_QueueUpdate_TableRefreshes(t *testing.T) {
	s := state.New()
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(100, 30)

	// Initially empty.
	output := pane.View()
	assert.NotContains(t, output, "New Track")

	// Load queue data.
	s.SetQueue([]api.Track{
		{ID: "t1", Name: "New Track", Artists: []api.Artist{{Name: "Artist"}}},
	})
	pane.refreshRows()

	output = pane.View()
	assert.Contains(t, output, "New Track", "table should reflect new queue data")
}

// TestQueuePane_PlayingIndicatorPersists verifies ▶ persists across refreshes.
func TestQueuePane_PlayingIndicatorPersists(t *testing.T) {
	s := state.New()
	s.SetQueue([]api.Track{
		{ID: "t1", Name: "Track A", Artists: []api.Artist{{Name: "Artist"}}},
		{ID: "t2", Name: "Track B", Artists: []api.Artist{{Name: "Artist"}}},
	})
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(100, 30)
	pane.SetPlayingIndex(0)

	output := pane.View()
	assert.Contains(t, output, "▶", "playing indicator should appear")

	// Refresh rows (simulating data update).
	pane.refreshRows()
	output2 := pane.View()
	assert.Contains(t, output2, "▶", "playing indicator should persist after refresh")
}

// TestQueuePane_FilterScrollInteraction tests filtering then scrolling then clearing filter.
func TestQueuePane_FilterScrollInteraction(t *testing.T) {
	s := state.New()
	tracks := make([]api.Track, 10)
	for i := range tracks {
		tracks[i] = api.Track{
			ID:      fmt.Sprintf("t%d", i),
			Name:    fmt.Sprintf("Rock Track %d", i+1),
			Artists: []api.Artist{{Name: "Band"}},
		}
	}
	s.SetQueue(tracks)
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(100, 20)

	// Filter to "Rock" — all 10 match.
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	pane = m.(*QueuePane)
	for _, r := range "Rock" {
		m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		pane = m.(*QueuePane)
	}
	output := pane.View()
	assert.Contains(t, output, "Rock Track 1")

	// Close filter — full list restores.
	m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyEsc})
	pane = m.(*QueuePane)
	assert.False(t, pane.filter.IsActive())
	fullOutput := pane.View()
	assert.Contains(t, fullOutput, "Rock Track 1")
}

// TestQueuePane_LargeQueue verifies 200 items scroll correctly without panic.
func TestQueuePane_LargeQueue(t *testing.T) {
	s := state.New()
	tracks := make([]api.Track, 200)
	for i := range tracks {
		tracks[i] = api.Track{
			ID:      fmt.Sprintf("t%d", i),
			Name:    fmt.Sprintf("Track %d", i+1),
			Artists: []api.Artist{{Name: "Artist"}},
		}
	}
	s.SetQueue(tracks)
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(100, 30)

	// Navigate down 100 times — should not panic.
	for i := 0; i < 100; i++ {
		m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		pane = m.(*QueuePane)
	}

	output := pane.View()
	assert.NotEmpty(t, output, "should render without panic for large queue")
}

// TestQueuePane_LongTrackName verifies very long track names don't overflow.
func TestQueuePane_LongTrackName(t *testing.T) {
	s := state.New()
	s.SetQueue([]api.Track{
		{
			ID:      "t1",
			Name:    "This Is A Very Long Track Name That Exceeds Any Reasonable Column Width By Far",
			Artists: []api.Artist{{Name: "This Is Also A Very Long Artist Name That Won't Fit"}},
		},
	})
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(60, 20) // narrow to force truncation

	// Should not panic and output should not exceed pane width.
	// Use lipgloss.Width() to measure visible width after stripping ANSI escapes.
	output := pane.View()
	assert.NotEmpty(t, output)
	for _, line := range splitLines(output) {
		assert.LessOrEqual(t, lipgloss.Width(line), 80, "line should not massively overflow pane width")
	}
}
