package panes

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestPlayerPane creates a PlayerPane with a fresh store and black theme.
func newTestPlayerPane(focused bool) *PlayerPane {
	s := state.New()
	t := theme.Load("black")
	return NewPlayerPane(s, t, focused)
}

// newTestPlayerPaneWithState creates a PlayerPane pre-loaded with playback state.
func newTestPlayerPaneWithState(isPlaying bool, focused bool) *PlayerPane {
	s := state.New()
	s.SetPlaybackState(&api.PlaybackState{
		IsPlaying:    isPlaying,
		ProgressMs:   30000,
		ShuffleState: false,
		RepeatState:  "off",
		Item: &api.Track{
			ID:         "track-1",
			Name:       "Blinding Lights",
			DurationMs: 252000,
			Artists:    []api.Artist{{ID: "a1", Name: "The Weeknd"}},
			Album:      api.Album{ID: "alb1", Name: "After Hours"},
		},
		Device: &api.Device{
			ID:            "dev-1",
			Name:          "MacBook Pro",
			VolumePercent: 65,
		},
	})
	t := theme.Load("black")
	return NewPlayerPane(s, t, focused)
}

func TestPlayerPane_View_NowPlaying(t *testing.T) {
	pane := newTestPlayerPaneWithState(true, true)
	pane.SetSize(80, 24)
	output := pane.View()

	assert.Contains(t, output, "Blinding Lights", "should show track name")
	assert.Contains(t, output, "The Weeknd", "should show artist name")
	assert.Contains(t, output, "After Hours", "should show album name")
}

func TestPlayerPane_View_EmptyState(t *testing.T) {
	pane := newTestPlayerPane(true)
	pane.SetSize(80, 24)
	output := pane.View()

	assert.Contains(t, output, "Nothing playing", "should show empty state message")
}

func TestPlayerPane_Update_Space_WhenPlaying(t *testing.T) {
	pane := newTestPlayerPaneWithState(true, true)

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	_, cmd := pane.Update(spaceMsg)

	require.NotNil(t, cmd, "space when playing should return a command")
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok, "space cmd should return PlaybackRequestMsg, got %T", msg)
	assert.Equal(t, ActionPause, req.Action, "playing → space should request pause")
}

func TestPlayerPane_Update_Space_WhenPaused(t *testing.T) {
	pane := newTestPlayerPaneWithState(false, true)

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	_, cmd := pane.Update(spaceMsg)

	require.NotNil(t, cmd, "space when paused should return a command")
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok, "space cmd should return PlaybackRequestMsg, got %T", msg)
	assert.Equal(t, ActionPlay, req.Action, "paused → space should request play")
}

func TestPlayerPane_Update_N_SkipsNext(t *testing.T) {
	pane := newTestPlayerPaneWithState(true, true)

	nMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	_, cmd := pane.Update(nMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok)
	assert.Equal(t, ActionNext, req.Action)
}

func TestPlayerPane_Update_P_SkipsPrev(t *testing.T) {
	pane := newTestPlayerPaneWithState(true, true)

	pMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	_, cmd := pane.Update(pMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok)
	assert.Equal(t, ActionPrevious, req.Action)
}

func TestPlayerPane_Update_Plus_VolUp(t *testing.T) {
	pane := newTestPlayerPaneWithState(true, true)

	plusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}
	_, cmd := pane.Update(plusMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok)
	assert.Equal(t, ActionVolumeUp, req.Action)
}

func TestPlayerPane_Update_Minus_VolDown(t *testing.T) {
	pane := newTestPlayerPaneWithState(true, true)

	minusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}}
	_, cmd := pane.Update(minusMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok)
	assert.Equal(t, ActionVolumeDown, req.Action)
}

func TestPlayerPane_Update_S_TogglesShuffle(t *testing.T) {
	pane := newTestPlayerPaneWithState(true, true)

	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	_, cmd := pane.Update(sMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok)
	assert.Equal(t, ActionToggleShuffle, req.Action)
}

func TestPlayerPane_Update_R_CyclesRepeat(t *testing.T) {
	tests := []struct {
		name        string
		startRepeat string
	}{
		{"off", "off"},
		{"context", "context"},
		{"track", "track"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := state.New()
			s.SetPlaybackState(&api.PlaybackState{
				IsPlaying:   true,
				RepeatState: tt.startRepeat,
				Item: &api.Track{
					ID:         "t1",
					Name:       "Track",
					DurationMs: 200000,
					Artists:    []api.Artist{{Name: "Artist"}},
				},
				Device: &api.Device{VolumePercent: 50},
			})
			th := theme.Load("black")
			pane := NewPlayerPane(s, th, true)

			rMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
			_, cmd := pane.Update(rMsg)

			require.NotNil(t, cmd)
			msg := cmd()
			req, ok := msg.(PlaybackRequestMsg)
			assert.True(t, ok)
			assert.Equal(t, ActionCycleRepeat, req.Action)
		})
	}
}

func TestPlayerPane_Update_PlaybackFetched(t *testing.T) {
	pane := newTestPlayerPaneWithState(true, true)

	// Simulate app.go updating the store and sending PlaybackStateFetchedMsg.
	newState := &api.PlaybackState{
		IsPlaying:  true,
		ProgressMs: 90000,
		Item: &api.Track{
			ID:         "track-2",
			Name:       "Save Your Tears",
			DurationMs: 215000,
			Artists:    []api.Artist{{Name: "The Weeknd"}},
			Album:      api.Album{Name: "After Hours"},
		},
		Device: &api.Device{VolumePercent: 70},
	}
	pane.store.SetPlaybackState(newState)

	fetchedMsg := PlaybackStateFetchedMsg{}
	updatedModel, _ := pane.Update(fetchedMsg)

	updatedPane, ok := updatedModel.(*PlayerPane)
	require.True(t, ok)

	// localProgressMs should be reset to server value.
	assert.Equal(t, 90000, updatedPane.localProgressMs)
}

func TestPlayerPane_Update_PlaybackFetched_NilState(t *testing.T) {
	pane := newTestPlayerPaneWithState(true, true)
	pane.localProgressMs = 60000

	// Nil state (nothing playing) — store is cleared, then notification sent.
	pane.store.SetPlaybackState(nil)

	fetchedMsg := PlaybackStateFetchedMsg{}
	updatedModel, _ := pane.Update(fetchedMsg)

	updatedPane, ok := updatedModel.(*PlayerPane)
	require.True(t, ok)

	assert.Equal(t, 0, updatedPane.localProgressMs)
	assert.Nil(t, updatedPane.store.PlaybackState())
}

func TestPlayerPane_Update_IgnoresKeysWhenNotFocused(t *testing.T) {
	pane := newTestPlayerPaneWithState(true, false) // not focused

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	_, cmd := pane.Update(spaceMsg)

	assert.Nil(t, cmd, "unfocused pane should return nil cmd for key events")
}

func TestPlayerPane_Update_TickIncrements(t *testing.T) {
	pane := newTestPlayerPaneWithState(true, true)
	pane.localProgressMs = 30000

	tickMsg := TickMsg{}
	updatedModel, _ := pane.Update(tickMsg)

	updatedPane, ok := updatedModel.(*PlayerPane)
	require.True(t, ok)

	assert.Equal(t, 31000, updatedPane.localProgressMs)
}

func TestPlayerPane_Update_TickNoIncrement_WhenPaused(t *testing.T) {
	pane := newTestPlayerPaneWithState(false, true)
	pane.localProgressMs = 30000

	tickMsg := TickMsg{}
	updatedModel, _ := pane.Update(tickMsg)

	updatedPane, ok := updatedModel.(*PlayerPane)
	require.True(t, ok)

	assert.Equal(t, 30000, updatedPane.localProgressMs)
}

func TestNextRepeatMode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"off", "context"},
		{"context", "track"},
		{"track", "off"},
		{"unknown", "off"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := nextRepeatMode(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPlayerPane_SetFocused(t *testing.T) {
	pane := newTestPlayerPane(false)
	assert.False(t, pane.focused)

	pane.SetFocused(true)
	assert.True(t, pane.focused)
}

func TestPlayerPane_IsFocused(t *testing.T) {
	pane := newTestPlayerPane(false)
	assert.False(t, pane.IsFocused())

	pane.SetFocused(true)
	assert.True(t, pane.IsFocused())
}

func TestPlayerPane_Init(t *testing.T) {
	pane := newTestPlayerPane(true)
	cmd := pane.Init()
	assert.Nil(t, cmd)
}

func TestDeviceName_NoDevice(t *testing.T) {
	s := state.New()
	name := DeviceName(s)
	assert.Empty(t, name)
}

func TestDeviceName_WithDevice(t *testing.T) {
	s := state.New()
	s.SetActiveDevice(&api.Device{Name: "MacBook Pro"})
	name := DeviceName(s)
	assert.Contains(t, name, "MacBook Pro")
}

// TestPlayerPane_ArrowKeys tests left/right arrow keys for navigation.
func TestPlayerPane_ArrowKeys(t *testing.T) {
	tests := []struct {
		name       string
		keyType    tea.KeyType
		wantAction PlaybackAction
	}{
		{"right arrow skips next", tea.KeyRight, ActionNext},
		{"left arrow skips prev", tea.KeyLeft, ActionPrevious},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane := newTestPlayerPaneWithState(true, true)

			keyMsg := tea.KeyMsg{Type: tt.keyType}
			_, cmd := pane.Update(keyMsg)

			require.NotNil(t, cmd)
			msg := cmd()
			req, ok := msg.(PlaybackRequestMsg)
			assert.True(t, ok)
			assert.Equal(t, tt.wantAction, req.Action)
		})
	}
}

// TestPlayerPane_Space_NilState tests that pressing space with no state still returns cmd.
func TestPlayerPane_Space_NilState(t *testing.T) {
	pane := newTestPlayerPane(true)

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	_, cmd := pane.Update(spaceMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok)
	assert.Equal(t, ActionPlay, req.Action, "nil state → space should request play")
}
