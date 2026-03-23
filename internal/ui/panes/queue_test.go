package panes

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/state"
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

// TestQueuePane_View_EmptyQueue verifies that an empty queue shows the centered message.
func TestQueuePane_View_EmptyQueue(t *testing.T) {
	pane := newTestQueuePane(true)
	pane.SetSize(40, 20)
	output := pane.View()

	assert.Contains(t, output, "Queue is empty", "should show empty message when no queue data")
}

// TestQueuePane_View_NowPlaying verifies that the NOW section shows the current track name and artist.
func TestQueuePane_View_NowPlaying(t *testing.T) {
	pane := newTestQueuePaneWithData(true)
	pane.SetSize(40, 20)
	output := pane.View()

	assert.Contains(t, output, "NOW", "should show NOW label")
	assert.Contains(t, output, "Blinding Lights", "should show current track name")
	assert.Contains(t, output, "The Weeknd", "should show current track artist")
}

// TestQueuePane_View_NextUp verifies that the NEXT UP section shows numbered items with artist.
func TestQueuePane_View_NextUp(t *testing.T) {
	pane := newTestQueuePaneWithData(true)
	pane.SetSize(40, 20)
	output := pane.View()

	assert.Contains(t, output, "NEXT UP", "should show NEXT UP label")
	assert.Contains(t, output, "Save Your Tears", "should show first queued track")
	assert.Contains(t, output, "Starboy", "should show second queued track")
	// Numbered list: check that "1" appears before first track
	idxNum := strings.Index(output, "1")
	idxTrack := strings.Index(output, "Save Your Tears")
	assert.Greater(t, idxTrack, -1, "first track should appear")
	assert.Greater(t, idxNum, -1, "track number should appear")
}

// TestQueuePane_View_ItemCount verifies that the track count footer is shown.
func TestQueuePane_View_ItemCount(t *testing.T) {
	pane := newTestQueuePaneWithData(true)
	pane.SetSize(40, 20)
	output := pane.View()

	assert.Contains(t, output, "3 tracks remaining", "should show remaining track count")
}

// TestQueuePane_Update_J_MovesDown verifies that pressing j moves the cursor down.
func TestQueuePane_Update_J_MovesDown(t *testing.T) {
	pane := newTestQueuePaneWithData(true)
	pane.SetSize(40, 20)

	initialCursor := pane.Cursor()
	updated, _ := pane.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	pp, ok := updated.(*QueuePane)
	require.True(t, ok)
	assert.Greater(t, pp.Cursor(), initialCursor, "cursor should move down after j")
}

// TestQueuePane_Update_K_MovesUp verifies that pressing k moves the cursor up.
func TestQueuePane_Update_K_MovesUp(t *testing.T) {
	pane := newTestQueuePaneWithData(true)
	pane.SetSize(40, 20)

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
	pane.SetSize(40, 20)

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
	pane.SetSize(40, 20)

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
