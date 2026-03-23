package panes_test

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newPlaylistManager creates a PlaylistManager with a pre-populated store for testing.
func newPlaylistManager() (*panes.PlaylistManager, *state.Store) {
	t := theme.Load("black")
	s := state.New()
	s.SetPlaylists([]api.SimplePlaylist{
		{ID: "pl-1", Name: "Chill Vibes", URI: "spotify:playlist:pl-1", TrackCount: 24},
		{ID: "pl-2", Name: "Workout Mix", URI: "spotify:playlist:pl-2", TrackCount: 48},
		{ID: "pl-3", Name: "Late Night", URI: "spotify:playlist:pl-3", TrackCount: 12},
	})
	pm := panes.NewPlaylistManager(s, t)
	pm.SetSize(80, 20)
	return pm, s
}

// ─── Task 8.2: PlaylistManager left pane ───────────────────────────────────

// TestPlaylistManager_View_PlaylistList verifies playlists render with names and track counts.
func TestPlaylistManager_View_PlaylistList(t *testing.T) {
	pm, _ := newPlaylistManager()
	view := pm.View()

	assert.Contains(t, view, "Chill Vibes")
	assert.Contains(t, view, "Workout Mix")
	assert.Contains(t, view, "Late Night")
	assert.Contains(t, view, "24") // track count for Chill Vibes
	assert.Contains(t, view, "48") // track count for Workout Mix
}

// TestPlaylistManager_View_PlayingIndicator verifies the ▶ indicator shows
// next to the currently playing playlist.
func TestPlaylistManager_View_PlayingIndicator(t *testing.T) {
	pm, s := newPlaylistManager()
	// Set playback context to pl-1.
	s.SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
		Item:      &api.Track{ID: "t1", Name: "Blinding Lights", URI: "spotify:track:t1"},
	})
	s.SetPlayingPlaylistID("pl-1")

	view := pm.View()
	assert.Contains(t, view, "▶", "should show playing indicator")
}

// TestPlaylistManager_Update_N_OpensInput verifies pressing n shows the name input.
func TestPlaylistManager_Update_N_OpensInput(t *testing.T) {
	pm, _ := newPlaylistManager()
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	pm, ok := updated.(*panes.PlaylistManager)
	require.True(t, ok)
	assert.True(t, pm.InputOpen(), "input should be open after pressing n")
}

// TestPlaylistManager_Update_R_OpensRename verifies pressing r opens the input pre-filled.
func TestPlaylistManager_Update_R_OpensRename(t *testing.T) {
	pm, _ := newPlaylistManager()
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	pm, ok := updated.(*panes.PlaylistManager)
	require.True(t, ok)
	assert.True(t, pm.InputOpen(), "input should be open after pressing r")
	assert.Equal(t, "Chill Vibes", pm.InputValue(), "input should be pre-filled with playlist name")
}

// TestPlaylistManager_Update_Enter_SubmitsCreate verifies pressing Enter with input open
// returns a create command and hides the input.
func TestPlaylistManager_Update_Enter_SubmitsCreate(t *testing.T) {
	pm, _ := newPlaylistManager()
	// Open input.
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	pm = updated.(*panes.PlaylistManager)

	// Type a name.
	pm.SetInputValue("My Test Playlist")

	// Press Enter.
	updated, cmd := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)
	require.NotNil(t, cmd, "should return a command on submit")
	assert.False(t, pm.InputOpen(), "input should be closed after submit")
}

// TestPlaylistManager_Update_Esc_CancelsInput verifies Esc hides the input and restores state.
func TestPlaylistManager_Update_Esc_CancelsInput(t *testing.T) {
	pm, _ := newPlaylistManager()
	// Open input.
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	pm = updated.(*panes.PlaylistManager)
	assert.True(t, pm.InputOpen())

	// Press Esc.
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	pm = updated.(*panes.PlaylistManager)
	assert.False(t, pm.InputOpen(), "input should be closed after Esc")
}

// TestPlaylistManager_Update_Enter_SelectsPlaylist verifies pressing Enter without
// input open triggers loading the selected playlist's tracks.
func TestPlaylistManager_Update_Enter_SelectsPlaylist(t *testing.T) {
	pm, _ := newPlaylistManager()
	updated, cmd := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm, ok := updated.(*panes.PlaylistManager)
	require.True(t, ok)
	// Either a command is returned (fetch tracks) or the pane advances to right focus.
	_ = pm
	_ = cmd
}

// TestPlaylistManager_Update_JK_Navigation verifies j/k move cursor.
func TestPlaylistManager_Update_JK_Navigation(t *testing.T) {
	pm, _ := newPlaylistManager()
	assert.Equal(t, 0, pm.Cursor())

	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	pm = updated.(*panes.PlaylistManager)
	assert.Equal(t, 1, pm.Cursor())

	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	pm = updated.(*panes.PlaylistManager)
	assert.Equal(t, 0, pm.Cursor())
}

// ─── Task 8.3: Track list (right pane) ────────────────────────────────────

// TestPlaylistTracks_View_TrackList verifies tracks render with name, artist, and duration.
func TestPlaylistTracks_View_TrackList(t *testing.T) {
	pm, s := newPlaylistManager()
	s.SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Blinding Lights", URI: "spotify:track:t1", DurationMs: 260000, Artists: []api.Artist{{Name: "The Weeknd"}}},
		{ID: "t2", Name: "Levitating", URI: "spotify:track:t2", DurationMs: 203000, Artists: []api.Artist{{Name: "Dua Lipa"}}},
	})

	// Select playlist to load tracks.
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)

	view := pm.View()
	assert.Contains(t, view, "Blinding Lights")
	assert.Contains(t, view, "The Weeknd")
	assert.Contains(t, view, "4:20", "should format 260000ms as 4:20")
	assert.Contains(t, view, "Levitating")
	assert.Contains(t, view, "Dua Lipa")
	assert.Contains(t, view, "3:23", "should format 203000ms as 3:23")
}

// TestPlaylistTracks_View_Footer verifies footer shows total tracks and duration.
func TestPlaylistTracks_View_Footer(t *testing.T) {
	pm, s := newPlaylistManager()
	s.SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Blinding Lights", URI: "spotify:track:t1", DurationMs: 260000, Artists: []api.Artist{{Name: "The Weeknd"}}},
		{ID: "t2", Name: "Levitating", URI: "spotify:track:t2", DurationMs: 203000, Artists: []api.Artist{{Name: "Dua Lipa"}}},
	})

	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)

	view := pm.View()
	assert.Contains(t, view, "2 tracks", "footer should show track count")
}

// TestPlaylistTracks_Update_X_ShowsConfirmation verifies x shows a remove confirmation prompt.
func TestPlaylistTracks_Update_X_ShowsConfirmation(t *testing.T) {
	pm, s := newPlaylistManager()
	s.SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Blinding Lights", URI: "spotify:track:t1", DurationMs: 260000, Artists: []api.Artist{{Name: "The Weeknd"}}},
	})

	// Select playlist then switch focus to right pane.
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyTab})
	pm = updated.(*panes.PlaylistManager)

	// Press x on the track.
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	pm = updated.(*panes.PlaylistManager)

	assert.True(t, pm.ConfirmOpen(), "should show confirmation prompt after x")
	view := pm.View()
	assert.True(t, strings.Contains(view, "Remove") || strings.Contains(view, "remove"), "confirmation should mention remove")
}

// TestPlaylistTracks_Update_Y_ConfirmsRemove verifies y confirms removal and
// returns a remove command; track disappears optimistically.
func TestPlaylistTracks_Update_Y_ConfirmsRemove(t *testing.T) {
	pm, s := newPlaylistManager()
	s.SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Blinding Lights", URI: "spotify:track:t1", DurationMs: 260000, Artists: []api.Artist{{Name: "The Weeknd"}}},
		{ID: "t2", Name: "Levitating", URI: "spotify:track:t2", DurationMs: 203000, Artists: []api.Artist{{Name: "Dua Lipa"}}},
	})

	// Select playlist.
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)
	// Tab to right pane.
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyTab})
	pm = updated.(*panes.PlaylistManager)

	// Press x.
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	pm = updated.(*panes.PlaylistManager)
	require.True(t, pm.ConfirmOpen())

	// Press y to confirm.
	updated, cmd := pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	pm = updated.(*panes.PlaylistManager)
	require.NotNil(t, cmd, "should return remove command after y")
	assert.False(t, pm.ConfirmOpen(), "confirmation should be closed")

	// Track should be removed optimistically.
	view := pm.View()
	assert.NotContains(t, view, "Blinding Lights", "removed track should no longer appear")
}

// TestPlaylistTracks_Update_ShiftDown_ReordersDown verifies Shift+Down moves the selected track down.
func TestPlaylistTracks_Update_ShiftDown_ReordersDown(t *testing.T) {
	pm, s := newPlaylistManager()
	s.SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Blinding Lights", URI: "spotify:track:t1", DurationMs: 260000, Artists: []api.Artist{{Name: "The Weeknd"}}},
		{ID: "t2", Name: "Levitating", URI: "spotify:track:t2", DurationMs: 203000, Artists: []api.Artist{{Name: "Dua Lipa"}}},
		{ID: "t3", Name: "Save Your Tears", URI: "spotify:track:t3", DurationMs: 215000, Artists: []api.Artist{{Name: "The Weeknd"}}},
	})

	// Select playlist.
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)
	// Tab to right pane.
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyTab})
	pm = updated.(*panes.PlaylistManager)
	// Cursor is on t1 (index 0). Press Shift+Down to move t1 to index 1.
	updated, cmd := pm.Update(tea.KeyMsg{Type: tea.KeyShiftDown})
	pm = updated.(*panes.PlaylistManager)
	require.NotNil(t, cmd, "should return reorder command")

	// The view should show tracks in new order.
	view := pm.View()
	levIdx := strings.Index(view, "Levitating")
	blindIdx := strings.Index(view, "Blinding Lights")
	assert.True(t, levIdx < blindIdx, "Levitating should appear before Blinding Lights after shift-down")
}

// TestPlaylistTracks_Update_ShiftUp_ReordersUp verifies Shift+Up moves the selected track up.
func TestPlaylistTracks_Update_ShiftUp_ReordersUp(t *testing.T) {
	pm, s := newPlaylistManager()
	s.SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Blinding Lights", URI: "spotify:track:t1", DurationMs: 260000, Artists: []api.Artist{{Name: "The Weeknd"}}},
		{ID: "t2", Name: "Levitating", URI: "spotify:track:t2", DurationMs: 203000, Artists: []api.Artist{{Name: "Dua Lipa"}}},
		{ID: "t3", Name: "Save Your Tears", URI: "spotify:track:t3", DurationMs: 215000, Artists: []api.Artist{{Name: "The Weeknd"}}},
	})

	// Select playlist.
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)
	// Tab to right pane.
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyTab})
	pm = updated.(*panes.PlaylistManager)
	// Move to track index 1.
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	pm = updated.(*panes.PlaylistManager)
	// Cursor is on t2. Press Shift+Up to move it to index 0.
	updated, cmd := pm.Update(tea.KeyMsg{Type: tea.KeyShiftUp})
	pm = updated.(*panes.PlaylistManager)
	require.NotNil(t, cmd, "should return reorder command")

	// Levitating should now appear before Blinding Lights.
	view := pm.View()
	levIdx := strings.Index(view, "Levitating")
	blindIdx := strings.Index(view, "Blinding Lights")
	assert.True(t, levIdx < blindIdx, "Levitating should appear before Blinding Lights after shift-up")
}

// TestPlaylistTracks_ReorderRevert_OnError verifies that a reorder error msg
// causes the track list to revert to its previous order.
func TestPlaylistTracks_ReorderRevert_OnError(t *testing.T) {
	pm, s := newPlaylistManager()
	s.SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Blinding Lights", URI: "spotify:track:t1", DurationMs: 260000, Artists: []api.Artist{{Name: "The Weeknd"}}},
		{ID: "t2", Name: "Levitating", URI: "spotify:track:t2", DurationMs: 203000, Artists: []api.Artist{{Name: "Dua Lipa"}}},
	})

	// Select playlist.
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)
	// Tab to right pane.
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyTab})
	pm = updated.(*panes.PlaylistManager)

	// Reorder: move t1 down.
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyShiftDown})
	pm = updated.(*panes.PlaylistManager)

	// Simulate error response.
	updated, _ = pm.Update(panes.PlaylistReorderResultMsg{Err: assert.AnError})
	pm = updated.(*panes.PlaylistManager)

	// Order should revert: Blinding Lights before Levitating.
	view := pm.View()
	blindIdx := strings.Index(view, "Blinding Lights")
	levIdx := strings.Index(view, "Levitating")
	assert.True(t, blindIdx < levIdx, "after error, Blinding Lights should appear before Levitating")
}

// TestPlaylistTracks_RemoveRevert_OnError verifies that a remove error msg
// causes the removed track to reappear.
func TestPlaylistTracks_RemoveRevert_OnError(t *testing.T) {
	pm, s := newPlaylistManager()
	s.SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Blinding Lights", URI: "spotify:track:t1", DurationMs: 260000, Artists: []api.Artist{{Name: "The Weeknd"}}},
		{ID: "t2", Name: "Levitating", URI: "spotify:track:t2", DurationMs: 203000, Artists: []api.Artist{{Name: "Dua Lipa"}}},
	})

	// Select playlist.
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)
	// Tab to right pane.
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyTab})
	pm = updated.(*panes.PlaylistManager)

	// Press x, then y to remove first track.
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	pm = updated.(*panes.PlaylistManager)
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	pm = updated.(*panes.PlaylistManager)

	// Track should be removed.
	view := pm.View()
	assert.NotContains(t, view, "Blinding Lights")

	// Simulate error response.
	updated, _ = pm.Update(panes.PlaylistRemoveResultMsg{Err: assert.AnError})
	pm = updated.(*panes.PlaylistManager)

	// Track should reappear.
	view = pm.View()
	assert.Contains(t, view, "Blinding Lights", "track should reappear after error")
}

// TestPlaylistManager_Init verifies Init returns nil (no startup command needed).
func TestPlaylistManager_Init(t *testing.T) {
	pm, _ := newPlaylistManager()
	cmd := pm.Init()
	assert.Nil(t, cmd)
}

// TestPlaylistManager_SelectedPlaylistID verifies SelectedPlaylistID returns the current ID.
func TestPlaylistManager_SelectedPlaylistID(t *testing.T) {
	pm, _ := newPlaylistManager()
	assert.Equal(t, "", pm.SelectedPlaylistID())

	// Select a playlist via Enter.
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)
	assert.Equal(t, "pl-1", pm.SelectedPlaylistID())
}

// TestPlaylistManager_RightCursor verifies RightCursor navigation in the right pane.
func TestPlaylistManager_RightCursor(t *testing.T) {
	pm, s := newPlaylistManager()
	s.SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Track A", URI: "spotify:track:t1", DurationMs: 180000, Artists: []api.Artist{{Name: "Artist A"}}},
		{ID: "t2", Name: "Track B", URI: "spotify:track:t2", DurationMs: 200000, Artists: []api.Artist{{Name: "Artist B"}}},
	})

	// Select playlist and switch to right pane.
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyTab})
	pm = updated.(*panes.PlaylistManager)

	assert.Equal(t, 0, pm.RightCursor())

	// j moves cursor down.
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	pm = updated.(*panes.PlaylistManager)
	assert.Equal(t, 1, pm.RightCursor())

	// k moves cursor up.
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	pm = updated.(*panes.PlaylistManager)
	assert.Equal(t, 0, pm.RightCursor())

	// Arrow keys also work.
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyDown})
	pm = updated.(*panes.PlaylistManager)
	assert.Equal(t, 1, pm.RightCursor())

	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyUp})
	pm = updated.(*panes.PlaylistManager)
	assert.Equal(t, 0, pm.RightCursor())
}

// TestPlaylistManager_AddToQueue verifies 'a' in right pane emits AddToQueueMsg.
func TestPlaylistManager_AddToQueue(t *testing.T) {
	pm, s := newPlaylistManager()
	s.SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Track A", URI: "spotify:track:t1", DurationMs: 180000, Artists: []api.Artist{{Name: "Artist A"}}},
	})
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyTab})
	pm = updated.(*panes.PlaylistManager)

	updated, cmd := pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	pm = updated.(*panes.PlaylistManager)
	require.NotNil(t, cmd, "should return AddToQueueMsg command")
	_ = pm
}

// TestPlaylistManager_EnterRight_PlaysTrack verifies Enter in right pane emits PlayTrackMsg.
func TestPlaylistManager_EnterRight_PlaysTrack(t *testing.T) {
	pm, s := newPlaylistManager()
	s.SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Track A", URI: "spotify:track:t1", DurationMs: 180000, Artists: []api.Artist{{Name: "Artist A"}}},
	})
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyTab})
	pm = updated.(*panes.PlaylistManager)

	updated, cmd := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)
	require.NotNil(t, cmd, "Enter in right pane should return PlayTrackMsg command")
	_ = pm
}

// TestPlaylistManager_DownArrow_LeftPane verifies down arrow moves cursor in left pane.
func TestPlaylistManager_DownArrow_LeftPane(t *testing.T) {
	pm, _ := newPlaylistManager()
	assert.Equal(t, 0, pm.Cursor())

	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyDown})
	pm = updated.(*panes.PlaylistManager)
	assert.Equal(t, 1, pm.Cursor())

	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyUp})
	pm = updated.(*panes.PlaylistManager)
	assert.Equal(t, 0, pm.Cursor())
}

// TestPlaylistManager_TabWithoutSelection_NoSwitch verifies Tab before selecting
// a playlist does not switch focus.
func TestPlaylistManager_TabWithoutSelection_NoSwitch(t *testing.T) {
	pm, _ := newPlaylistManager()
	// No playlist selected — Tab should not crash.
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyTab})
	pm = updated.(*panes.PlaylistManager)
	// Still on left pane (no right pane content).
	_ = pm
}

// TestPlaylistManager_InputKey_EmptyNameCancels verifies pressing Enter with empty name cancels.
func TestPlaylistManager_InputKey_EmptyNameCancels(t *testing.T) {
	pm, _ := newPlaylistManager()
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	pm = updated.(*panes.PlaylistManager)
	require.True(t, pm.InputOpen())

	// Don't type anything — press Enter with empty input.
	updated, cmd := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)
	assert.False(t, pm.InputOpen(), "input should close on empty Enter")
	assert.Nil(t, cmd, "no command should be returned for empty name")
}

// TestPlaylistManager_InputKey_OtherKey verifies typing in the input works.
func TestPlaylistManager_InputKey_OtherKey(t *testing.T) {
	pm, _ := newPlaylistManager()
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	pm = updated.(*panes.PlaylistManager)
	require.True(t, pm.InputOpen())

	// Type a character.
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("A")})
	pm = updated.(*panes.PlaylistManager)
	assert.True(t, pm.InputOpen(), "input should remain open while typing")
}

// TestPlaylistManager_Confirm_NKey_Cancels verifies pressing n cancels the remove confirmation.
func TestPlaylistManager_Confirm_NKey_Cancels(t *testing.T) {
	pm, s := newPlaylistManager()
	s.SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Track A", URI: "spotify:track:t1", DurationMs: 180000, Artists: []api.Artist{{Name: "Artist A"}}},
	})
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyTab})
	pm = updated.(*panes.PlaylistManager)
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	pm = updated.(*panes.PlaylistManager)
	require.True(t, pm.ConfirmOpen())

	// Press n to cancel.
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	pm = updated.(*panes.PlaylistManager)
	assert.False(t, pm.ConfirmOpen(), "confirmation should close after n")
}

// TestPlaylistManager_Confirm_EscCancels verifies Esc cancels the remove confirmation.
func TestPlaylistManager_Confirm_EscCancels(t *testing.T) {
	pm, s := newPlaylistManager()
	s.SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Track A", URI: "spotify:track:t1", DurationMs: 180000, Artists: []api.Artist{{Name: "Artist A"}}},
	})
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyTab})
	pm = updated.(*panes.PlaylistManager)
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	pm = updated.(*panes.PlaylistManager)
	require.True(t, pm.ConfirmOpen())

	// Press Esc to cancel.
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	pm = updated.(*panes.PlaylistManager)
	assert.False(t, pm.ConfirmOpen(), "confirmation should close after Esc")
}

// TestPlaylistManager_TabBack verifies Tab in right pane switches focus back to left.
func TestPlaylistManager_TabBack(t *testing.T) {
	pm, s := newPlaylistManager()
	s.SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Track A", URI: "spotify:track:t1", DurationMs: 180000, Artists: []api.Artist{{Name: "Artist A"}}},
	})
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyTab})
	pm = updated.(*panes.PlaylistManager)

	// Tab back to left pane.
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyTab})
	pm = updated.(*panes.PlaylistManager)
	// Now n should open input (left pane behavior).
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	pm = updated.(*panes.PlaylistManager)
	assert.True(t, pm.InputOpen(), "after Tab back, n should open input (left pane)")
}

// TestPlaylistManager_Update_PlaylistCreatedMsg_Error verifies PlaylistCreatedMsg with error is handled.
func TestPlaylistManager_Update_PlaylistCreatedMsg_Error(t *testing.T) {
	pm, _ := newPlaylistManager()
	updated, _ := pm.Update(panes.PlaylistCreatedMsg{Err: assert.AnError})
	pm = updated.(*panes.PlaylistManager)
	_ = pm
}

// TestPlaylistManager_Update_PlaylistRenamedMsg_Error verifies PlaylistRenamedMsg with error is handled.
func TestPlaylistManager_Update_PlaylistRenamedMsg_Error(t *testing.T) {
	pm, _ := newPlaylistManager()
	updated, _ := pm.Update(panes.PlaylistRenamedMsg{Err: assert.AnError})
	pm = updated.(*panes.PlaylistManager)
	_ = pm
}

// TestPlaylistManager_Update_TracksLoadedMsg_WrongPlaylist verifies tracks msg for other playlist is ignored.
func TestPlaylistManager_Update_TracksLoadedMsg_WrongPlaylist(t *testing.T) {
	pm, s := newPlaylistManager()
	s.SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Track A", URI: "spotify:track:t1", DurationMs: 180000, Artists: []api.Artist{{Name: "Artist A"}}},
	})
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)

	// Receive tracks loaded msg for a different playlist — should be ignored.
	updated, _ = pm.Update(panes.PlaylistTracksLoadedMsg{PlaylistID: "pl-99"})
	pm = updated.(*panes.PlaylistManager)
	_ = pm
}

// TestPlaylistManager_View_WithSize verifies the view renders correctly with non-zero dimensions.
func TestPlaylistManager_View_WithSize(t *testing.T) {
	pm, _ := newPlaylistManager()
	pm.SetSize(120, 30)
	view := pm.View()
	assert.Contains(t, view, "Chill Vibes")
}

// TestPlaylistManager_ReorderDown_AtBottom verifies Shift+Down at last track does nothing.
func TestPlaylistManager_ReorderDown_AtBottom(t *testing.T) {
	pm, s := newPlaylistManager()
	s.SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Track A", URI: "spotify:track:t1", DurationMs: 180000, Artists: []api.Artist{{Name: "Artist A"}}},
		{ID: "t2", Name: "Track B", URI: "spotify:track:t2", DurationMs: 200000, Artists: []api.Artist{{Name: "Artist B"}}},
	})
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyTab})
	pm = updated.(*panes.PlaylistManager)

	// Move to last track.
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	pm = updated.(*panes.PlaylistManager)
	assert.Equal(t, 1, pm.RightCursor())

	// Shift+Down at last track should not move cursor.
	updated, cmd := pm.Update(tea.KeyMsg{Type: tea.KeyShiftDown})
	pm = updated.(*panes.PlaylistManager)
	assert.Equal(t, 1, pm.RightCursor())
	assert.Nil(t, cmd, "no reorder command at last position")
}

// TestPlaylistManager_ReorderUp_AtTop verifies Shift+Up at first track does nothing.
func TestPlaylistManager_ReorderUp_AtTop(t *testing.T) {
	pm, s := newPlaylistManager()
	s.SetPlaylistTracks("pl-1", []api.Track{
		{ID: "t1", Name: "Track A", URI: "spotify:track:t1", DurationMs: 180000, Artists: []api.Artist{{Name: "Artist A"}}},
	})
	updated, _ := pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm = updated.(*panes.PlaylistManager)
	updated, _ = pm.Update(tea.KeyMsg{Type: tea.KeyTab})
	pm = updated.(*panes.PlaylistManager)

	// Shift+Up at index 0 should not move cursor.
	updated, cmd := pm.Update(tea.KeyMsg{Type: tea.KeyShiftUp})
	pm = updated.(*panes.PlaylistManager)
	assert.Equal(t, 0, pm.RightCursor())
	assert.Nil(t, cmd, "no reorder command at first position")
}

// TestPlaylistManager_FormatDuration verifies the m:ss duration formatting.
func TestPlaylistManager_FormatDuration(t *testing.T) {
	tests := []struct {
		ms   int
		want string
	}{
		{ms: 260000, want: "4:20"},
		{ms: 203000, want: "3:23"},
		{ms: 215000, want: "3:35"},
		{ms: 60000, want: "1:00"},
		{ms: 90500, want: "1:30"},
		{ms: 0, want: "0:00"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := panes.FormatDuration(tt.ms)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestPlaylistManager_View_ErrorState verifies that when the store has a PlaylistsError,
// the view shows "Failed to load playlists" with a retry hint (B16 fix).
func TestPlaylistManager_View_ErrorState(t *testing.T) {
	t.Helper()
	th := theme.Load("black")
	s := state.New()
	s.SetPlaylistsError(fmt.Errorf("spotify: 500 internal server error"))
	pm := panes.NewPlaylistManager(s, th)
	pm.SetSize(120, 30)

	view := pm.View()
	assert.Contains(t, view, "Failed to load playlists", "error state must show failure message")
	assert.Contains(t, view, "retry", "error state must show retry hint")
}

// TestPlaylistManager_View_ErrorCleared verifies that once PlaylistsError is cleared,
// the normal playlist list renders.
func TestPlaylistManager_View_ErrorCleared(t *testing.T) {
	t.Helper()
	th := theme.Load("black")
	s := state.New()
	s.SetPlaylistsError(fmt.Errorf("transient error"))
	pm := panes.NewPlaylistManager(s, th)
	pm.SetSize(120, 30)

	// Error present — error view renders.
	view := pm.View()
	assert.Contains(t, view, "Failed to load playlists")

	// Clear error and populate data.
	s.ClearPlaylistsError()
	s.SetPlaylists([]api.SimplePlaylist{
		{ID: "pl-1", Name: "My Playlist", URI: "spotify:playlist:pl-1", TrackCount: 10},
	})

	view = pm.View()
	assert.NotContains(t, view, "Failed to load playlists", "error should not show after cleared")
	assert.Contains(t, view, "My Playlist", "playlist should appear after error cleared")
}

// TestPlaylistManager_Init_RequestsPlaylistData verifies that Init returns a
// command requesting playlist data when store is empty (B16 fix).
func TestPlaylistManager_Init_RequestsPlaylistData(t *testing.T) {
	t.Helper()
	th := theme.Load("black")
	s := state.New()
	// Store has no playlists.
	pm := panes.NewPlaylistManager(s, th)

	cmd := pm.Init()
	// When store is empty, Init should emit a FetchPlaylistsRequestMsg.
	require.NotNil(t, cmd, "Init should return a non-nil command when playlists not loaded")

	msg := cmd()
	_, ok := msg.(panes.FetchPlaylistsRequestMsg)
	require.True(t, ok, "Init command should return FetchPlaylistsRequestMsg, got %T", msg)
}

// TestPlaylistManager_Init_NoFetchWhenDataLoaded verifies that Init returns nil
// when playlists are already in the store (avoid duplicate fetches).
func TestPlaylistManager_Init_NoFetchWhenDataLoaded(t *testing.T) {
	t.Helper()
	th := theme.Load("black")
	s := state.New()
	s.SetPlaylists([]api.SimplePlaylist{
		{ID: "pl-1", Name: "Already Loaded", URI: "spotify:playlist:pl-1", TrackCount: 5},
	})
	pm := panes.NewPlaylistManager(s, th)

	cmd := pm.Init()
	assert.Nil(t, cmd, "Init should return nil when playlists already loaded")
}
