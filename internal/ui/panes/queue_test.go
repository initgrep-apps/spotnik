package panes

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/domain"
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

// makes a track QueueItem for tests.
func qiTrack(id, name, uri, artist string) domain.QueueItem {
	return domain.QueueItem{
		Type: domain.QueueItemTypeTrack,
		Track: &domain.Track{
			ID:      id,
			Name:    name,
			URI:     uri,
			Artists: []domain.Artist{{Name: artist}},
		},
	}
}

// makes an episode QueueItem for tests.
func qiEpisode(id, name, uri string, durMs int, showName string) domain.QueueItem {
	return domain.QueueItem{
		Type: domain.QueueItemTypeEpisode,
		Episode: &domain.Episode{
			ID:         id,
			Name:       name,
			URI:        uri,
			DurationMs: durMs,
			Show:       &domain.Show{Name: showName},
		},
	}
}

// newTestQueuePaneWithData creates a QueuePane pre-loaded with playback state and queue.
func newTestQueuePaneWithData(focused bool) *QueuePane {
	s := state.New()
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying: true,
		Item: &domain.Track{
			ID:      "now-1",
			Name:    "Blinding Lights",
			URI:     "spotify:track:now-1",
			Artists: []domain.Artist{{Name: "The Weeknd"}},
		},
	})
	s.SetQueue([]domain.QueueItem{
		qiTrack("q1", "Save Your Tears", "spotify:track:q1", "The Weeknd"),
		qiTrack("q2", "Starboy", "spotify:track:q2", "The Weeknd"),
		qiTrack("q3", "Can't Feel My Face", "spotify:track:q3", "The Weeknd"),
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
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying: true,
		Item:      &domain.Track{ID: "t1", Name: "Now", Artists: []domain.Artist{{Name: "A"}}},
	})

	// Create 25 items in the queue.
	items := make([]domain.QueueItem, 25)
	for i := range items {
		items[i] = qiTrack(fmt.Sprintf("q%d", i), fmt.Sprintf("Track %d", i+1), "uri", "Artist")
	}
	s.SetQueue(items)

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
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying: true,
		Item:      &domain.Track{ID: "t1", Name: "Now", Artists: []domain.Artist{{Name: "A"}}},
	})

	items := make([]domain.QueueItem, 25)
	for i := range items {
		items[i] = qiTrack(fmt.Sprintf("q%d", i), fmt.Sprintf("Track %d", i+1), "uri", "Artist")
	}
	s.SetQueue(items)

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
	s.SetPlaybackState(&domain.PlaybackState{
		IsPlaying:   true,
		RepeatState: "off",
		Item:        &domain.Track{ID: "t1", Name: "Now Playing", Artists: []domain.Artist{{Name: "A"}}},
	})
	s.SetQueue([]domain.QueueItem{
		qiTrack("q1", "Next Track", "uri", "B"),
	})

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

// TestQueuePane_Actions verifies that Actions() returns only filter action by default.
func TestQueuePane_Actions(t *testing.T) {
	pane := newTestQueuePane(false)
	actions := pane.Actions()
	require.Len(t, actions, 1)
	assert.Equal(t, "f", actions[0].Key)
	assert.Equal(t, "filter", actions[0].Label)
}

// --- Task 2: bubble-table rendering ---

// TestQueuePane_Table_FiveTracks verifies the table has 5 rows for 5 queued tracks.
func TestQueuePane_Table_FiveTracks(t *testing.T) {
	s := state.New()
	items := make([]domain.QueueItem, 5)
	for i := range items {
		items[i] = qiTrack(fmt.Sprintf("t%d", i+1), fmt.Sprintf("Track %d", i+1), "uri", "Artist")
	}
	s.SetQueue(items)

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
	assert.Contains(t, output, "Title", "should contain Title column header")
	assert.Contains(t, output, "Artist", "should contain Artist column header")
	assert.Contains(t, output, "Duration", "should contain Duration column header")
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
	s.SetQueue([]domain.QueueItem{
		qiTrack("q1", "Rocket Man", "uri", "Elton John"),
		qiTrack("q2", "Rock and Roll", "uri", "Led Zeppelin"),
		qiTrack("q3", "Save Your Tears", "uri", "The Weeknd"),
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
	s.SetQueue([]domain.QueueItem{
		qiTrack("q1", "Blinding Lights", "uri", "The Weeknd"),
		qiTrack("q2", "Levitating", "uri", "Dua Lipa"),
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

// TestQueuePane_Filter_ActionsUnchangedWhenActive verifies Actions() always returns
// the {f, filter} hint.
func TestQueuePane_Filter_ActionsUnchangedWhenActive(t *testing.T) {
	pane := newTestQueuePaneWithData(true)
	pane.SetSize(80, 20)

	// Default actions: only filter.
	actions := pane.Actions()
	require.Len(t, actions, 1)
	assert.Equal(t, "f", actions[0].Key)

	// Activate filter — Actions() must still return {f, filter}, not {Esc, close}.
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	pp := m.(*QueuePane)
	filterActions := pp.Actions()
	require.Len(t, filterActions, 1, "filter active must still return {f, filter}")
	assert.Equal(t, "f", filterActions[0].Key)
	assert.Equal(t, "filter", filterActions[0].Label)
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
	items := []domain.QueueItem{
		qiTrack("t1", "Blinding Lights", "uri", "The Weeknd"),
		qiTrack("t2", "Levitating", "uri", "Dua Lipa"),
		qiTrack("t3", "Rocket Man", "uri", "Elton John"),
	}
	s.SetQueue(items)
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

	// Step 4: navigate (k key in filter mode).
	m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	pane = m.(*QueuePane)

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
	s.SetQueue([]domain.QueueItem{
		qiTrack("t1", "New Track", "uri", "Artist"),
	})
	pane.refreshRows()

	output = pane.View()
	assert.Contains(t, output, "New Track", "table should reflect new queue data")
}

// TestQueuePane_FilterScrollInteraction tests filtering then scrolling then clearing filter.
func TestQueuePane_FilterScrollInteraction(t *testing.T) {
	s := state.New()
	items := make([]domain.QueueItem, 10)
	for i := range items {
		items[i] = qiTrack(fmt.Sprintf("t%d", i), fmt.Sprintf("Rock Track %d", i+1), "uri", "Band")
	}
	s.SetQueue(items)
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
	items := make([]domain.QueueItem, 200)
	for i := range items {
		items[i] = qiTrack(fmt.Sprintf("t%d", i), fmt.Sprintf("Track %d", i+1), "uri", "Artist")
	}
	s.SetQueue(items)
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
	s.SetQueue([]domain.QueueItem{
		qiTrack("t1", "This Is A Very Long Track Name That Exceeds Any Reasonable Column Width By Far", "uri", "This Is Also A Very Long Artist Name That Won't Fit"),
	})
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(60, 20) // narrow to force truncation

	output := pane.View()
	assert.NotEmpty(t, output)
	for _, line := range splitLines(output) {
		assert.LessOrEqual(t, lipgloss.Width(line), 80, "line should not massively overflow pane width")
	}
}

// --- Story 120: dead pane action removal ---

// TestQueuePane_Actions_NoAddEntry verifies 'A' is not in queue Actions()
func TestQueuePane_Actions_NoAddEntry(t *testing.T) {
	pane := newTestQueuePane(false)
	for _, a := range pane.Actions() {
		assert.NotEqual(t, "A", a.Key, "Actions() must not include 'A' (no handler exists)")
	}
}

// --- Story 71 Task 2: column color tokens ---

// TestQueuePane_UsesColumnColors verifies that QueuePane column definitions
// use the new ColumnIndex/ColumnPrimary/ColumnSecondary/ColumnTertiary tokens.
func TestQueuePane_UsesColumnColors(t *testing.T) {
	th := theme.Load("black")
	q := NewQueuePane(state.New(), th, false)
	cols := q.table.Columns()
	require.Len(t, cols, 6, "QueuePane should have 6 columns")

	assert.Equal(t, th.ColumnIndex(), cols[0].Color, "# column should use ColumnIndex()")
	assert.Equal(t, th.ColumnSecondary(), cols[1].Color, "type column should use ColumnSecondary()")
	assert.Equal(t, th.ColumnPrimary(), cols[2].Color, "Title column should use ColumnPrimary()")
	assert.Equal(t, th.ColumnSecondary(), cols[3].Color, "Artist column should use ColumnSecondary()")
	assert.Equal(t, th.ColumnTertiary(), cols[4].Color, "Duration column should use ColumnTertiary()")
	assert.Equal(t, th.ColumnSecondary(), cols[5].Color, "icon column should use ColumnSecondary()")
}

// --- Story 173: Esc scroll-reset ---

// TableCurrentPage returns the current page of the queue pane's inner table.
func (q *QueuePane) TableCurrentPage() int { return q.table.CurrentPage() }

// TestQueuePane_Esc_ResetsScrollToPage1 verifies that pressing Esc when no filter
// is active resets the table scroll position back to page 1.
func TestQueuePane_Esc_ResetsScrollToPage1(t *testing.T) {
	s := state.New()
	items := make([]domain.QueueItem, 20)
	for i := range items {
		items[i] = qiTrack(fmt.Sprintf("q%d", i), fmt.Sprintf("Track %d", i+1), "uri", "Artist")
	}
	s.SetQueue(items)
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	// height=11 → pageSize=5 with ShowHeader=true (pageSize = height - 6).
	pane.SetSize(80, 11)

	// Scroll 8 rows down to advance past page 1.
	for i := 0; i < 8; i++ {
		m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		pane = m.(*QueuePane)
	}
	require.Greater(t, pane.TableCurrentPage(), 1, "should have scrolled past page 1")

	// Press Esc with no active filter — should reset to page 1.
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyEscape})
	pane = m.(*QueuePane)
	assert.Equal(t, 1, pane.TableCurrentPage(), "Esc should reset table to page 1")
}

// TestQueuePane_ActiveFilterQuery_ReturnsCommittedQuery verifies that
// ActiveFilterQuery() returns the committed filter query after f → type → Enter.
func TestQueuePane_ActiveFilterQuery_ReturnsCommittedQuery(t *testing.T) {
	s := state.New()
	s.SetQueue([]domain.QueueItem{
		qiTrack("uri:1", "Rock Track", "uri:1", "Artist"),
	})
	pane := NewQueuePane(s, theme.Load("black"), true)
	pane.SetSize(80, 20)

	assert.Equal(t, "", pane.ActiveFilterQuery(), "empty before filter applied")

	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	for _, r := range "rock" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	pane.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.Equal(t, "rock", pane.ActiveFilterQuery())
}

// TestQueuePane_Esc_ClearsCommittedFilter verifies that Esc clears a committed
// filter query (restoring all rows) before falling back to scroll-reset.
func TestQueuePane_Esc_ClearsCommittedFilter(t *testing.T) {
	s := state.New()
	s.SetQueue([]domain.QueueItem{
		qiTrack("uri:1", "Rock Track", "uri:1", "Artist"),
		qiTrack("uri:2", "Jazz Track", "uri:2", "Artist"),
	})
	pane := NewQueuePane(s, theme.Load("black"), true)
	pane.SetSize(80, 20)

	// Apply filter: f → "rock" → Enter
	pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	for _, r := range "rock" {
		pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Equal(t, "rock", pane.ActiveFilterQuery(), "filter must be committed")

	// Esc → clears filter
	pane.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.Equal(t, "", pane.ActiveFilterQuery(), "Esc must clear committed filter")
}

// --- Story 238: mixed content tests ---

// TestQueuePane_TypeColumn_TrackSymbol verifies ♪ appears in type column for tracks.
func TestQueuePane_TypeColumn_TrackSymbol(t *testing.T) {
	s := state.New()
	s.SetQueue([]domain.QueueItem{
		qiTrack("t1", "Track Song", "uri", "Artist"),
	})
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(80, 20)
	output := pane.View()

	assert.Contains(t, output, "♪", "♪ symbol should appear for tracks")
}

// TestQueuePane_TypeColumn_EpisodeSymbol verifies ◆ appears in type column for episodes.
func TestQueuePane_TypeColumn_EpisodeSymbol(t *testing.T) {
	s := state.New()
	s.SetQueue([]domain.QueueItem{
		qiEpisode("ep-1", "Episode Title", "spotify:episode:ep-1", 1800000, "Show Name"),
	})
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(80, 20)
	output := pane.View()

	assert.Contains(t, output, "◆", "◆ symbol should appear for episodes")
}

// TestQueuePane_TitleHeader verifies "Title" replaces "Track" as column header.
func TestQueuePane_TitleHeader(t *testing.T) {
	s := state.New()
	s.SetQueue([]domain.QueueItem{
		qiTrack("t1", "Song Name", "uri", "Artist"),
	})
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(80, 20)
	output := pane.View()

	assert.Contains(t, output, "Title", "should have Title header")
	assert.NotContains(t, output, "Track", "should not have Track header")
}

// TestQueuePane_ArtistColumn_EpisodeShowName verifies episode's show name in Artist column.
func TestQueuePane_ArtistColumn_EpisodeShowName(t *testing.T) {
	s := state.New()
	s.SetQueue([]domain.QueueItem{
		qiEpisode("ep-1", "Episode Title", "spotify:episode:ep-1", 1800000, "My Podcast Show"),
	})
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(80, 20)
	output := pane.View()

	assert.Contains(t, output, "Episode Title", "should show episode name in title column")
	assert.Contains(t, output, "My Podcast Show", "should show show name in artist column")
}

// TestQueuePane_EnterTrack_PlaysTrack verifies Enter on a track row emits PlayTrackMsg.
func TestQueuePane_EnterTrack_PlaysTrack(t *testing.T) {
	s := state.New()
	s.SetQueue([]domain.QueueItem{
		qiTrack("t1", "Track Song", "spotify:track:t1", "Artist"),
	})
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(80, 20)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	playMsg, ok := msg.(PlayTrackMsg)
	require.True(t, ok)
	assert.Equal(t, "spotify:track:t1", playMsg.TrackURI)
}

// TestQueuePane_EnterEpisode_PlaysEpisode verifies Enter on an episode row emits PlayEpisodeMsg.
func TestQueuePane_EnterEpisode_PlaysEpisode(t *testing.T) {
	s := state.New()
	s.SetQueue([]domain.QueueItem{
		qiEpisode("ep-1", "Episode", "spotify:episode:ep-1", 1200000, "Show"),
	})
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(80, 20)

	_, cmd := pane.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	epMsg, ok := msg.(PlayEpisodeMsg)
	require.True(t, ok)
	assert.Equal(t, "spotify:episode:ep-1", epMsg.EpisodeURI)
	assert.Equal(t, "", epMsg.PlaylistURI)
}

// TestQueuePane_MixedContent_RendersBoth verifies mixed tracks and episodes render correctly.
func TestQueuePane_MixedContent_RendersBoth(t *testing.T) {
	s := state.New()
	s.SetQueue([]domain.QueueItem{
		qiTrack("t1", "Track One", "uri", "Artist A"),
		qiEpisode("e1", "Episode One", "uri", 1800000, "Show A"),
		qiTrack("t2", "Track Two", "uri", "Artist B"),
	})
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(80, 20)
	output := pane.View()

	assert.Contains(t, output, "♪", "♪ for first track")
	assert.Contains(t, output, "◆", "◆ for episode")
	assert.Contains(t, output, "Track One")
	assert.Contains(t, output, "Episode One")
	assert.Contains(t, output, "Track Two")
}

// TestQueuePane_MixedContent_FilterEpisodes verifies filter works on episode content.
func TestQueuePane_MixedContent_FilterEpisodes(t *testing.T) {
	s := state.New()
	s.SetQueue([]domain.QueueItem{
		qiTrack("t1", "Rock Song", "uri", "Artist"),
		qiEpisode("e1", "Tech Podcast", "uri", 1800000, "Tech Show"),
		qiEpisode("e2", "News Cast", "uri", 1800000, "News Show"),
	})
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(80, 20)

	// Activate filter and type "tech" — matches episode name.
	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	pane = m.(*QueuePane)
	for _, r := range "tech" {
		m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		pane = m.(*QueuePane)
	}
	output := pane.View()
	assert.NotContains(t, output, "Rock Song", "track should be filtered out")
	assert.Contains(t, output, "Tech Podcast", "episode matching filter should show")
	assert.NotContains(t, output, "News Cast", "non-matching episode should be hidden")
}

// TestQueuePane_MixedContent_FilterByShowName verifies filter works on show name.
func TestQueuePane_MixedContent_FilterByShowName(t *testing.T) {
	s := state.New()
	s.SetQueue([]domain.QueueItem{
		qiTrack("t1", "Song", "uri", "Artist"),
		qiEpisode("e1", "Episode", "uri", 1800000, "Specific Show"),
	})
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(80, 20)

	m, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	pane = m.(*QueuePane)
	for _, r := range "Specific" {
		m, _ = pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		pane = m.(*QueuePane)
	}
	output := pane.View()
	assert.NotContains(t, output, "Song", "track should be filtered out")
	assert.Contains(t, output, "Episode", "episode with matching show should show")
}

// TestQueuePane_Duration_EpisodeFormat verifies episode durations are formatted correctly.
func TestQueuePane_Duration_EpisodeFormat(t *testing.T) {
	s := state.New()
	s.SetQueue([]domain.QueueItem{
		qiEpisode("ep-1", "Episode", "uri", 3661000, "Show"),
	})
	th := theme.Load("black")
	pane := NewQueuePane(s, th, true)
	pane.SetSize(80, 20)
	output := pane.View()

	assert.Contains(t, output, "1:01:01", "episode duration should be formatted as h:mm:ss")
}
