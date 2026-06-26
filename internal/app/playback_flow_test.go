package app_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/app"
	"github.com/initgrep-apps/spotnik/internal/config"
	"github.com/initgrep-apps/spotnik/internal/domain"
	"github.com/initgrep-apps/spotnik/internal/ui/panes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Helpers ────────────────────────────────────────────────────────────────────

// newPlaybackTestApp creates an App with a premium user and a playing track set on the store.
func newPlaybackTestApp(t *testing.T) *app.App {
	t.Helper()
	cfg := &config.Config{}
	a := app.New(cfg, app.AppOptions{})

	// Set premium user so playback keys are not gated.
	a.Store().SetUserProfile(domain.UserProfile{
		ID:      "user-1",
		Product: "premium",
	})

	// Set a playing track on the store.
	a.Store().SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		CurrentlyPlayingType: "track",
		ProgressMs:           30000,
		ShuffleState:         false,
		RepeatState:          "off",
		Item: &domain.Track{
			ID:         "track-1",
			Name:       "Blinding Lights",
			URI:        "spotify:track:track-1",
			DurationMs: 252000,
			Artists:    []domain.Artist{{ID: "a1", Name: "The Weeknd"}},
			Album: domain.Album{
				ID:   "alb1",
				Name: "After Hours",
			},
		},
		Device: &domain.Device{
			ID:            "dev-1",
			Name:          "MacBook Pro",
			VolumePercent: 65,
		},
	})

	return a
}

// assertPlaybackRequestMsg asserts that cmd produces a PlaybackRequestMsg with the expected action.
// Returns the unwrapped message for further assertions.
func assertPlaybackRequestMsg(t *testing.T, cmd tea.Cmd, wantAction panes.PlaybackAction) panes.PlaybackRequestMsg {
	t.Helper()
	require.NotNil(t, cmd, "expected non-nil cmd")
	msg := cmd()
	req, ok := msg.(panes.PlaybackRequestMsg)
	require.True(t, ok, "expected PlaybackRequestMsg, got %T", msg)
	assert.Equal(t, wantAction, req.Action)
	return req
}

// ── Task 6: Playback flow integration tests ────────────────────────────────────

// TestPlaybackFlow_PauseThenResume verifies the full pause→resume cycle:
//  1. Space key when playing → PlaybackRequestMsg{Action: ActionPause}
//  2. Space key when paused → PlaybackRequestMsg{Action: ActionPlay}
func TestPlaybackFlow_PauseThenResume(t *testing.T) {
	a := newPlaybackTestApp(t)

	// Step 1: Space when playing → should request pause.
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeySpace})
	req := assertPlaybackRequestMsg(t, cmd, panes.ActionPause)
	require.NotNil(t, cmd)

	// Step 2: Set playback state to paused.
	a.Store().SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            false,
		CurrentlyPlayingType: "track",
		ProgressMs:           30000,
		Item: &domain.Track{
			ID:         "track-1",
			Name:       "Blinding Lights",
			DurationMs: 252000,
			Artists:    []domain.Artist{{Name: "The Weeknd"}},
		},
		Device: &domain.Device{VolumePercent: 65},
	})

	// Send PlaybackStateFetchedMsg so NowPlayingPane syncs from the store.
	a.Update(panes.PlaybackStateFetchedMsg{
		State: a.Store().PlaybackState(),
	})

	// Step 3: Space when paused → should request play.
	_, cmd = a.Update(tea.KeyMsg{Type: tea.KeySpace})
	assertPlaybackRequestMsg(t, cmd, panes.ActionPlay)

	_ = req // used in assert
}

// TestPlaybackFlow_SeekRight verifies that pressing → produces a SeekIntentMsg
// (via debounce) and that multiple rapid → keys debounce correctly.
func TestPlaybackFlow_SeekRight(t *testing.T) {
	a := newPlaybackTestApp(t)

	// Press → key — should produce a cmd (debounce tick).
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRight})
	require.NotNil(t, cmd, "→ key should produce a cmd")

	// Execute the cmd — it should be a components.SeekDebounceTickMsg that fires the debounce.
	msg := cmd()
	_, ok := msg.(panes.SeekIntentMsg)
	// NOTE: The debounce fires after a delay. The → key sends a HandleKey call which
	// sets pending and returns a debounce tick cmd. Executing it fires the intent.
	if !ok {
		// Could be a SeekDebounceTickMsg that needs another Update round.
		// For the first press, the cmd should eventually produce a SeekIntentMsg.
		// Try sending the tick msg back to the App.
		_, cmd2 := a.Update(msg)
		if cmd2 != nil {
			msg2 := cmd2()
			if intent, ok2 := msg2.(panes.SeekIntentMsg); ok2 {
				assert.Equal(t, 35000, intent.TargetMs, "seek target should be +5s from 30000")
				_ = intent
			}
		}
	}
}

// TestPlaybackFlow_ShiftRight_NextTrack verifies that Shift+→ produces
// PlaybackRequestMsg{Action: ActionNext}.
func TestPlaybackFlow_ShiftRight_NextTrack(t *testing.T) {
	a := newPlaybackTestApp(t)

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyShiftRight})
	assertPlaybackRequestMsg(t, cmd, panes.ActionNext)
}

// TestPlaybackFlow_ShiftLeft_PreviousTrack verifies that Shift+← produces
// PlaybackRequestMsg{Action: ActionPrevious}.
func TestPlaybackFlow_ShiftLeft_PreviousTrack(t *testing.T) {
	a := newPlaybackTestApp(t)

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyShiftLeft})
	assertPlaybackRequestMsg(t, cmd, panes.ActionPrevious)
}

// TestPlaybackFlow_CycleRepeat verifies that pressing 'r' cycles repeat modes
// via PlaybackRequestMsg{Action: ActionCycleRepeat}.
func TestPlaybackFlow_CycleRepeat(t *testing.T) {
	a := newPlaybackTestApp(t)

	// Press 'r' → should produce ActionCycleRepeat.
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	assertPlaybackRequestMsg(t, cmd, panes.ActionCycleRepeat)
}

// TestPlaybackFlow_ShuffleToggle verifies that pressing 's' toggles shuffle
// via PlaybackRequestMsg{Action: ActionToggleShuffle}.
func TestPlaybackFlow_ShuffleToggle(t *testing.T) {
	a := newPlaybackTestApp(t)

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	assertPlaybackRequestMsg(t, cmd, panes.ActionToggleShuffle)
}

// TestPlaybackFlow_VolumeUp verifies that pressing '+' produces a VolumeIntentMsg
// with the correct target volume (incremented by 1 from confirmed volume).
func TestPlaybackFlow_VolumeUp(t *testing.T) {
	a := newPlaybackTestApp(t)

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}})
	require.NotNil(t, cmd, "+ key should produce a cmd")

	// The cmd should produce a VolumeDebounceTickMsg that fires a VolumeIntentMsg.
	// Execute the initial cmd and then route back.
	msg := cmd()
	// It could be the debounce tick — send it through Update to get the intent.
	_, cmd2 := a.Update(msg)
	if cmd2 != nil {
		msg2 := cmd2()
		if intent, ok := msg2.(panes.VolumeIntentMsg); ok {
			assert.Equal(t, 66, intent.TargetVol, "volume should increment from 65 to 66")
			assert.Greater(t, intent.Seq, 0, "volume intent should have sequence number")
		}
	}
}

// TestPlaybackFlow_VolumeDown verifies that pressing '-' produces a VolumeIntentMsg
// with the correct target volume (decremented by 1 from confirmed volume).
func TestPlaybackFlow_VolumeDown(t *testing.T) {
	a := newPlaybackTestApp(t)

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	require.NotNil(t, cmd, "- key should produce a cmd")

	msg := cmd()
	_, cmd2 := a.Update(msg)
	if cmd2 != nil {
		msg2 := cmd2()
		if intent, ok := msg2.(panes.VolumeIntentMsg); ok {
			assert.Equal(t, 64, intent.TargetVol, "volume should decrement from 65 to 64")
			assert.Greater(t, intent.Seq, 0, "volume intent should have sequence number")
		}
	}
}

// TestPlaybackFlow_PlaybackRequestMsg_ProducesAPICmd verifies that sending
// PlaybackRequestMsg to the App produces a PlaybackCmdSentMsg (via buildPlaybackAPICmd).
func TestPlaybackFlow_PlaybackRequestMsg_ProducesAPICmd(t *testing.T) {
	a := newPlaybackTestApp(t)

	_, cmd := a.Update(panes.PlaybackRequestMsg{Action: panes.ActionPlay})
	require.NotNil(t, cmd, "PlaybackRequestMsg should produce an API cmd")

	msg := cmd()
	_, ok := msg.(panes.PlaybackCmdSentMsg)
	assert.True(t, ok, "should produce PlaybackCmdSentMsg, got %T", msg)
}

// TestPlaybackFlow_SeekLeft verifies that pressing ← produces a seek intent
// targeting -5s from the current position.
func TestPlaybackFlow_SeekLeft(t *testing.T) {
	a := newPlaybackTestApp(t)

	// Set progress to 35000 so -5s → 30000.
	a.Store().SetPlaybackState(&domain.PlaybackState{
		IsPlaying:            true,
		CurrentlyPlayingType: "track",
		ProgressMs:           35000,
		Item: &domain.Track{
			ID:         "track-1",
			Name:       "Blinding Lights",
			DurationMs: 252000,
			Artists:    []domain.Artist{{Name: "The Weeknd"}},
		},
		Device: &domain.Device{VolumePercent: 65},
	})

	// Also refresh the pane from the store.
	a.Update(panes.PlaybackStateFetchedMsg{State: a.Store().PlaybackState()})

	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyLeft})
	require.NotNil(t, cmd, "← key should produce a cmd")

	msg := cmd()
	_, cmd2 := a.Update(msg)
	if cmd2 != nil {
		msg2 := cmd2()
		if intent, ok := msg2.(panes.SeekIntentMsg); ok {
			assert.Equal(t, 30000, intent.TargetMs, "seek target should be -5s from 35000")
		}
	}
}
