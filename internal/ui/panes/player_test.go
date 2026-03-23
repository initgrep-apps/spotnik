package panes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	// Execute the command and verify it returns a playbackCmdSentMsg.
	msg := cmd()
	_, ok := msg.(PlaybackCmdSentMsg)
	assert.True(t, ok, "space cmd should return PlaybackCmdSentMsg, got %T", msg)
}

func TestPlayerPane_Update_Space_WhenPaused(t *testing.T) {
	pane := newTestPlayerPaneWithState(false, true)

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	_, cmd := pane.Update(spaceMsg)

	require.NotNil(t, cmd, "space when paused should return a command")
	msg := cmd()
	_, ok := msg.(PlaybackCmdSentMsg)
	assert.True(t, ok, "space cmd should return PlaybackCmdSentMsg, got %T", msg)
}

func TestPlayerPane_Update_N_SkipsNext(t *testing.T) {
	pane := newTestPlayerPaneWithState(true, true)

	nMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	_, cmd := pane.Update(nMsg)

	require.NotNil(t, cmd, "n key should return a command")
	msg := cmd()
	_, ok := msg.(PlaybackCmdSentMsg)
	assert.True(t, ok, "n cmd should return PlaybackCmdSentMsg")
}

func TestPlayerPane_Update_P_SkipsPrev(t *testing.T) {
	pane := newTestPlayerPaneWithState(true, true)

	pMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	_, cmd := pane.Update(pMsg)

	require.NotNil(t, cmd, "p key should return a command")
	msg := cmd()
	_, ok := msg.(PlaybackCmdSentMsg)
	assert.True(t, ok, "p cmd should return PlaybackCmdSentMsg")
}

func TestPlayerPane_Update_Plus_VolUp(t *testing.T) {
	pane := newTestPlayerPaneWithState(true, true)

	plusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}
	_, cmd := pane.Update(plusMsg)

	require.NotNil(t, cmd, "+ key should return a command")
	msg := cmd()
	_, ok := msg.(PlaybackCmdSentMsg)
	assert.True(t, ok, "+ cmd should return PlaybackCmdSentMsg")
}

func TestPlayerPane_Update_Minus_VolDown(t *testing.T) {
	pane := newTestPlayerPaneWithState(true, true)

	minusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}}
	_, cmd := pane.Update(minusMsg)

	require.NotNil(t, cmd, "- key should return a command")
	msg := cmd()
	_, ok := msg.(PlaybackCmdSentMsg)
	assert.True(t, ok, "- cmd should return PlaybackCmdSentMsg")
}

func TestPlayerPane_Update_S_TogglesShuffle(t *testing.T) {
	pane := newTestPlayerPaneWithState(true, true)

	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	_, cmd := pane.Update(sMsg)

	require.NotNil(t, cmd, "s key should return a command")
	msg := cmd()
	_, ok := msg.(PlaybackCmdSentMsg)
	assert.True(t, ok, "s cmd should return PlaybackCmdSentMsg")
}

func TestPlayerPane_Update_R_CyclesRepeat(t *testing.T) {
	tests := []struct {
		name           string
		startRepeat    string
		expectedRepeat string
	}{
		{"off to context", "off", "context"},
		{"context to track", "context", "track"},
		{"track to off", "track", "off"},
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
			updatedModel, cmd := pane.Update(rMsg)
			_ = updatedModel

			require.NotNil(t, cmd, "r key should return a command")
		})
	}
}

func TestPlayerPane_Update_PlaybackFetched(t *testing.T) {
	pane := newTestPlayerPaneWithState(true, true)

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

	fetchedMsg := PlaybackStateFetchedMsg{State: newState}
	updatedModel, _ := pane.Update(fetchedMsg)

	// After update, store should reflect new state.
	updatedPane, ok := updatedModel.(*PlayerPane)
	require.True(t, ok)

	// localProgressMs should be reset to server value.
	assert.Equal(t, 90000, updatedPane.localProgressMs)
}

func TestPlayerPane_Update_PlaybackFetched_NilState(t *testing.T) {
	pane := newTestPlayerPaneWithState(true, true)
	pane.localProgressMs = 60000

	// Nil state (nothing playing).
	fetchedMsg := PlaybackStateFetchedMsg{State: nil}
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

	// When playing, localProgressMs should increment by 1000 on tick.
	assert.Equal(t, 31000, updatedPane.localProgressMs)
}

func TestPlayerPane_Update_TickNoIncrement_WhenPaused(t *testing.T) {
	pane := newTestPlayerPaneWithState(false, true)
	pane.localProgressMs = 30000

	tickMsg := TickMsg{}
	updatedModel, _ := pane.Update(tickMsg)

	updatedPane, ok := updatedModel.(*PlayerPane)
	require.True(t, ok)

	// When paused, localProgressMs should NOT increment.
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

func TestPlayerPane_Init(t *testing.T) {
	pane := newTestPlayerPane(true)
	cmd := pane.Init()
	// Init is a no-op — returns nil.
	assert.Nil(t, cmd)
}

func TestPlayerPane_SetPlayer(t *testing.T) {
	pane := newTestPlayerPane(true)
	assert.Nil(t, pane.player)

	p := api.NewPlayer("", "token")
	pane.SetPlayer(p)
	assert.NotNil(t, pane.player)
}

func TestFetchPlaybackStateCmd_NilPlayer(t *testing.T) {
	s := state.New()
	cmd := FetchPlaybackStateCmd(nil, s)
	require.NotNil(t, cmd)

	msg := cmd()
	fetchedMsg, ok := msg.(PlaybackStateFetchedMsg)
	require.True(t, ok)
	assert.Nil(t, fetchedMsg.State)
}

func TestFetchPlaybackStateCmd_WithPlayer(t *testing.T) {
	s := state.New()

	fixture := api.PlaybackState{
		IsPlaying:  true,
		ProgressMs: 10000,
		Item: &api.Track{
			ID:         "t1",
			Name:       "Test Song",
			DurationMs: 200000,
			Artists:    []api.Artist{{ID: "a1", Name: "Artist"}},
			Album:      api.Album{ID: "alb1", Name: "Album"},
		},
		Device: &api.Device{ID: "d1", Name: "Test Device", VolumePercent: 50},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(fixture)
	}))
	defer srv.Close()

	player := api.NewPlayer(srv.URL, "test-token")
	cmd := FetchPlaybackStateCmd(player, s)
	require.NotNil(t, cmd)

	msg := cmd()
	fetchedMsg, ok := msg.(PlaybackStateFetchedMsg)
	require.True(t, ok)
	require.NotNil(t, fetchedMsg.State)
	assert.Equal(t, "Test Song", fetchedMsg.State.Item.Name)
}

func TestFetchPlaybackStateCmd_PlayerError(t *testing.T) {
	s := state.New()

	// Server returns 503.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	player := api.NewPlayer(srv.URL, "test-token")
	cmd := FetchPlaybackStateCmd(player, s)

	msg := cmd()
	fetchedMsg, ok := msg.(PlaybackStateFetchedMsg)
	require.True(t, ok)
	// Error returns nil state.
	assert.Nil(t, fetchedMsg.State)
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

// TestPlayerPane_BuildPlaybackCmd_WithPlayer tests that buildPlaybackCmd sends
// the correct API call when a real player is injected.
func TestPlayerPane_BuildPlaybackCmd_WithPlayer(t *testing.T) {
	tests := []struct {
		name     string
		key      rune
		wantPath string
		isPlay   bool
	}{
		{"next", 'n', "/v1/me/player/next", false},
		{"previous", 'p', "/v1/me/player/previous", false},
		{"shuffle toggle", 's', "/v1/me/player/shuffle", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			s := state.New()
			s.SetPlaybackState(&api.PlaybackState{
				IsPlaying:    true,
				RepeatState:  "off",
				ShuffleState: false,
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
			pane.SetPlayer(api.NewPlayer(srv.URL, "test-token"))

			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tt.key}}
			_, cmd := pane.Update(keyMsg)
			require.NotNil(t, cmd)

			msg := cmd()
			result, ok := msg.(PlaybackCmdSentMsg)
			require.True(t, ok)
			assert.NoError(t, result.Err)
			assert.Equal(t, tt.wantPath, capturedPath)
		})
	}
}

// TestPlayerPane_BuildPlaybackCmd_Volume tests volume up/down with a real player.
func TestPlayerPane_BuildPlaybackCmd_Volume(t *testing.T) {
	var capturedVol string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedVol = r.URL.Query().Get("volume_percent")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	s := state.New()
	s.SetPlaybackState(&api.PlaybackState{
		IsPlaying: true,
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
	pane.SetPlayer(api.NewPlayer(srv.URL, "test-token"))

	plusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}
	_, cmd := pane.Update(plusMsg)
	require.NotNil(t, cmd)
	cmd()

	assert.Equal(t, "55", capturedVol, "volume up should send 50+5=55")
}

// TestPlayerPane_BuildPlaybackCmd_Repeat tests repeat cycling with a real player.
func TestPlayerPane_BuildPlaybackCmd_Repeat(t *testing.T) {
	var capturedMode string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMode = r.URL.Query().Get("state")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	s := state.New()
	s.SetPlaybackState(&api.PlaybackState{
		IsPlaying:   true,
		RepeatState: "off",
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
	pane.SetPlayer(api.NewPlayer(srv.URL, "test-token"))

	rMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	_, cmd := pane.Update(rMsg)
	require.NotNil(t, cmd)
	cmd()

	assert.Equal(t, "context", capturedMode, "repeat off → context on r key")
}

// TestPlayerPane_ArrowKeys tests left/right arrow keys for navigation.
func TestPlayerPane_ArrowKeys(t *testing.T) {
	tests := []struct {
		name    string
		keyType tea.KeyType
	}{
		{"right arrow skips next", tea.KeyRight},
		{"left arrow skips prev", tea.KeyLeft},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane := newTestPlayerPaneWithState(true, true)

			keyMsg := tea.KeyMsg{Type: tt.keyType}
			_, cmd := pane.Update(keyMsg)

			require.NotNil(t, cmd)
			msg := cmd()
			_, ok := msg.(PlaybackCmdSentMsg)
			assert.True(t, ok)
		})
	}
}

// TestPlayerPane_Play_WhenNilState tests that pressing space with no state still returns cmd.
func TestPlayerPane_Space_NilState(t *testing.T) {
	pane := newTestPlayerPane(true)

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	_, cmd := pane.Update(spaceMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(PlaybackCmdSentMsg)
	assert.True(t, ok)
}
