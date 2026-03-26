package panes

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestNowPlayingPane creates a NowPlayingPane with a fresh store and black theme.
func newTestNowPlayingPane(focused bool) *NowPlayingPane {
	s := state.New()
	t := theme.Load("black")
	return NewNowPlayingPane(s, t, focused)
}

// newTestNowPlayingPaneWithState creates a NowPlayingPane pre-loaded with playback state.
func newTestNowPlayingPaneWithState(isPlaying bool, focused bool) *NowPlayingPane {
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
	return NewNowPlayingPane(s, t, focused)
}

// ── Task 1: Rename tests ─────────────────────────────────────────────────────

func TestNowPlayingPane_View_NowPlaying(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)
	output := pane.View()

	assert.Contains(t, output, "Blinding Lights", "should show track name")
	assert.Contains(t, output, "The Weeknd", "should show artist name")
	assert.Contains(t, output, "After Hours", "should show album name")
}

func TestNowPlayingPane_View_EmptyState(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.SetSize(80, 24)
	output := pane.View()

	assert.Contains(t, output, "Nothing playing", "should show empty state message")
}

func TestNowPlayingPane_Update_Space_WhenPlaying(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	_, cmd := pane.Update(spaceMsg)

	require.NotNil(t, cmd, "space when playing should return a command")
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok, "space cmd should return PlaybackRequestMsg, got %T", msg)
	assert.Equal(t, ActionPause, req.Action, "playing → space should request pause")
}

func TestNowPlayingPane_Update_Space_WhenPaused(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(false, true)

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	_, cmd := pane.Update(spaceMsg)

	require.NotNil(t, cmd, "space when paused should return a command")
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok, "space cmd should return PlaybackRequestMsg, got %T", msg)
	assert.Equal(t, ActionPlay, req.Action, "paused → space should request play")
}

func TestNowPlayingPane_Update_N_SkipsNext(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)

	nMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	_, cmd := pane.Update(nMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok)
	assert.Equal(t, ActionNext, req.Action)
}

func TestNowPlayingPane_Update_P_SkipsPrev(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)

	pMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	_, cmd := pane.Update(pMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok)
	assert.Equal(t, ActionPrevious, req.Action)
}

func TestNowPlayingPane_Update_Plus_VolUp(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)

	plusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}
	_, cmd := pane.Update(plusMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok)
	assert.Equal(t, ActionVolumeUp, req.Action)
}

func TestNowPlayingPane_Update_Minus_VolDown(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)

	minusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}}
	_, cmd := pane.Update(minusMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok)
	assert.Equal(t, ActionVolumeDown, req.Action)
}

func TestNowPlayingPane_Update_S_TogglesShuffle(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)

	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	_, cmd := pane.Update(sMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok)
	assert.Equal(t, ActionToggleShuffle, req.Action)
}

func TestNowPlayingPane_Update_R_CyclesRepeat(t *testing.T) {
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
			pane := NewNowPlayingPane(s, th, true)

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

func TestNowPlayingPane_Update_PlaybackFetched(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)

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

	updatedPane, ok := updatedModel.(*NowPlayingPane)
	require.True(t, ok)

	// localProgressMs should be reset to server value.
	assert.Equal(t, 90000, updatedPane.localProgressMs)
}

func TestNowPlayingPane_Update_PlaybackFetched_NilState(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.localProgressMs = 60000

	// Nil state (nothing playing) — store is cleared, then notification sent.
	pane.store.SetPlaybackState(nil)

	fetchedMsg := PlaybackStateFetchedMsg{}
	updatedModel, _ := pane.Update(fetchedMsg)

	updatedPane, ok := updatedModel.(*NowPlayingPane)
	require.True(t, ok)

	assert.Equal(t, 0, updatedPane.localProgressMs)
	assert.Nil(t, updatedPane.store.PlaybackState())
}

func TestNowPlayingPane_Update_IgnoresKeysWhenNotFocused(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, false) // not focused

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	_, cmd := pane.Update(spaceMsg)

	assert.Nil(t, cmd, "unfocused pane should return nil cmd for key events")
}

func TestNowPlayingPane_Update_TickIncrements(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.localProgressMs = 30000

	tickMsg := TickMsg{}
	updatedModel, _ := pane.Update(tickMsg)

	updatedPane, ok := updatedModel.(*NowPlayingPane)
	require.True(t, ok)

	assert.Equal(t, 31000, updatedPane.localProgressMs)
}

func TestNowPlayingPane_Update_TickNoIncrement_WhenPaused(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(false, true)
	pane.localProgressMs = 30000

	tickMsg := TickMsg{}
	updatedModel, _ := pane.Update(tickMsg)

	updatedPane, ok := updatedModel.(*NowPlayingPane)
	require.True(t, ok)

	assert.Equal(t, 30000, updatedPane.localProgressMs)
}

func TestNowPlayingPane_SetFocused(t *testing.T) {
	pane := newTestNowPlayingPane(false)
	assert.False(t, pane.focused)

	pane.SetFocused(true)
	assert.True(t, pane.focused)
}

func TestNowPlayingPane_IsFocused(t *testing.T) {
	pane := newTestNowPlayingPane(false)
	assert.False(t, pane.IsFocused())

	pane.SetFocused(true)
	assert.True(t, pane.IsFocused())
}

func TestNowPlayingPane_Init_ReturnsVisualizerCmd(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.SetSize(80, 24)
	cmd := pane.Init()
	// Init should return a command (visualizer tick).
	assert.NotNil(t, cmd, "Init should return visualizer tick command")
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

// TestNowPlayingPane_ArrowKeys tests left/right arrow keys for navigation.
func TestNowPlayingPane_ArrowKeys(t *testing.T) {
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
			pane := newTestNowPlayingPaneWithState(true, true)

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

// TestNowPlayingPane_Space_NilState tests that pressing space with no state still returns cmd.
func TestNowPlayingPane_Space_NilState(t *testing.T) {
	pane := newTestNowPlayingPane(true)

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	_, cmd := pane.Update(spaceMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok)
	assert.Equal(t, ActionPlay, req.Action, "nil state → space should request play")
}

// ── Task 2: layout.Pane interface tests ─────────────────────────────────────

// TestNowPlayingPane_ImplementsLayoutPane verifies compile-time interface satisfaction.
func TestNowPlayingPane_ImplementsLayoutPane(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	var _ layout.Pane = pane // compile-time check
	assert.NotNil(t, pane)
}

func TestNowPlayingPane_ID(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	assert.Equal(t, layout.PaneNowPlaying, pane.ID())
}

func TestNowPlayingPane_Title_DefaultIsNowPlaying(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.SetSize(80, 24) // full mode
	assert.Equal(t, "Now Playing", pane.Title())
}

func TestNowPlayingPane_ToggleKey(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	assert.Equal(t, 1, pane.ToggleKey())
}

func TestNowPlayingPane_Actions(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	actions := pane.Actions()
	require.Len(t, actions, 2)
	assert.Equal(t, "s", actions[0].Key)
	assert.Equal(t, "shuffle", actions[0].Label)
	assert.Equal(t, "r", actions[1].Key)
	assert.Equal(t, "repeat", actions[1].Label)
}

// ── Task 3: Visualizer tests ─────────────────────────────────────────────────

func TestNowPlayingPane_VisualizerTickMsg_AdvancesFrame(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	// Set visualizer to playing.
	pane.visualizer.SetPlaying(true)
	initialFrame := pane.visualizer.FrameIndex()

	// Send VisualizerTickMsg.
	_, _ = pane.Update(components.VisualizerTickMsg{})

	// Frame should have advanced (visualizer was playing).
	assert.Equal(t, (initialFrame+1)%40, pane.visualizer.FrameIndex(),
		"VisualizerTickMsg should advance frame when playing")
}

func TestNowPlayingPane_PlaybackFetched_SetsVisualizerPlaying(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(false, true)
	pane.SetSize(80, 24)

	// Update store to playing=true, then send PlaybackStateFetchedMsg.
	pane.store.SetPlaybackState(&api.PlaybackState{
		IsPlaying:  true,
		ProgressMs: 5000,
		Item: &api.Track{
			Name:       "Track",
			DurationMs: 200000,
			Artists:    []api.Artist{{Name: "Artist"}},
		},
	})
	_, _ = pane.Update(PlaybackStateFetchedMsg{})

	// Visualizer should now be in playing state.
	// Verify by sending a tick and checking frame advances.
	before := pane.visualizer.FrameIndex()
	_, _ = pane.Update(components.VisualizerTickMsg{})
	assert.Equal(t, (before+1)%40, pane.visualizer.FrameIndex(),
		"visualizer should animate when playing=true")
}

func TestNowPlayingPane_PlaybackFetched_PausesVisualizer(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)
	pane.visualizer.SetPlaying(true)

	// Update store to paused, send PlaybackStateFetchedMsg.
	pane.store.SetPlaybackState(&api.PlaybackState{
		IsPlaying:  false,
		ProgressMs: 5000,
		Item: &api.Track{
			Name:       "Track",
			DurationMs: 200000,
			Artists:    []api.Artist{{Name: "Artist"}},
		},
	})
	_, _ = pane.Update(PlaybackStateFetchedMsg{})

	// Visualizer should be paused — frame should not advance on tick.
	before := pane.visualizer.FrameIndex()
	_, _ = pane.Update(components.VisualizerTickMsg{})
	assert.Equal(t, before, pane.visualizer.FrameIndex(),
		"visualizer should not animate when playing=false")
}

func TestNowPlayingPane_FullView_ContainsBrailleChars(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)
	pane.visualizer.SetPlaying(true)

	output := pane.View()

	// Braille chars used by the visualizer (any of the 5 codepoints).
	hasBraille := false
	for _, r := range output {
		if r >= '\u2800' && r <= '\u28FF' {
			hasBraille = true
			break
		}
	}
	assert.True(t, hasBraille, "full mode View() should contain braille characters from visualizer")
}

// ── Task 4: Gradient bars tests ──────────────────────────────────────────────

func TestNowPlayingPane_SeekBar_Renders(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	// The gradient seek bar renders time stamps like "0:30" for 30000ms.
	assert.Contains(t, output, "0:30", "seek bar should show current time")
}

func TestNowPlayingPane_VolumeBar_Renders(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	// GradientVolumeBar always renders "VOL" prefix.
	assert.Contains(t, output, "VOL", "volume bar should be rendered in full mode")
}

func TestNowPlayingPane_BarsResize_WithSetSize(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)

	pane.SetSize(40, 24)
	small := pane.View()

	pane.SetSize(80, 24)
	large := pane.View()

	assert.NotEqual(t, small, large, "different sizes should produce different output")
}

// ── Task 5: Compact mode tests ───────────────────────────────────────────────

func TestNowPlayingPane_CompactMode_EnabledAtHeight3(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.SetSize(80, 3)
	assert.True(t, pane.compact, "height=3 should enable compact mode")
}

func TestNowPlayingPane_CompactMode_DisabledAtHeight10(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.SetSize(80, 10)
	assert.False(t, pane.compact, "height=10 should not enable compact mode")
}

func TestNowPlayingPane_CompactMode_DisabledAtHeight4(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.SetSize(80, 4)
	assert.False(t, pane.compact, "height=4 should not enable compact mode")
}

func TestNowPlayingPane_CompactTitle_IncludesTrackInfo(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 3) // compact mode
	pane.localProgressMs = 30000

	title := pane.Title()
	assert.Contains(t, title, "Blinding Lights", "compact title should include track name")
	assert.Contains(t, title, "The Weeknd", "compact title should include artist name")
	// Time format: "0:30" for 30000ms.
	assert.Contains(t, title, "0:30", "compact title should include current time")
}

func TestNowPlayingPane_CompactView_SingleContentLine(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 3)

	output := pane.View()
	lines := filterNonEmpty(output)

	// Compact mode should produce at most 1 content line.
	assert.LessOrEqual(t, len(lines), 1, "compact mode should have at most 1 content line")
}

func TestNowPlayingPane_CompactView_ContainsVol(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 3)

	output := pane.View()
	assert.Contains(t, output, "VOL", "compact view should contain volume bar")
}

func TestNowPlayingPane_NoVisualizerInCompactMode(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 3)
	pane.visualizer.SetPlaying(true)

	output := pane.View()

	// No braille chars in compact mode.
	hasBraille := false
	for _, r := range output {
		if r >= '\u2800' && r <= '\u28FF' {
			hasBraille = true
			break
		}
	}
	assert.False(t, hasBraille, "compact mode should not render braille visualizer")
}

func TestNowPlayingPane_FullView_ContainsTrackAndAlbum(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	assert.Contains(t, output, "Blinding Lights")
	assert.Contains(t, output, "The Weeknd")
	assert.Contains(t, output, "After Hours")
}

// TestNowPlayingPane_Transition_FullToCompact verifies resize triggers mode change.
func TestNowPlayingPane_Transition_FullToCompact(t *testing.T) {
	pane := newTestNowPlayingPaneWithState(true, true)

	pane.SetSize(80, 24) // full mode
	assert.False(t, pane.compact, "should start in full mode")

	pane.SetSize(80, 3) // compact mode
	assert.True(t, pane.compact, "should switch to compact after resize to height=3")

	pane.SetSize(80, 24) // back to full mode
	assert.False(t, pane.compact, "should return to full mode after resize to height=24")
}

// TestNowPlayingPane_CompactView_NilState verifies safe rendering when nothing is playing.
func TestNowPlayingPane_CompactView_NilState(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.SetSize(80, 3) // compact mode

	output := pane.View()
	assert.Contains(t, output, "Nothing playing", "compact nil state should show empty message")
}

// TestNowPlayingPane_ZeroDuration verifies seek bar handles zero duration gracefully.
func TestNowPlayingPane_ZeroDuration(t *testing.T) {
	s := state.New()
	s.SetPlaybackState(&api.PlaybackState{
		IsPlaying:  true,
		ProgressMs: 0,
		Item: &api.Track{
			Name:       "Track",
			DurationMs: 0, // zero duration edge case
			Artists:    []api.Artist{{Name: "Artist"}},
		},
	})
	th := theme.Load("black")
	pane := NewNowPlayingPane(s, th, true)
	pane.SetSize(80, 24)

	// Should not panic.
	output := pane.View()
	assert.NotEmpty(t, output)
}

// TestNowPlayingPane_VisualizerTickMsg_ReturnsCmd verifies the tick returns a re-arm command.
func TestNowPlayingPane_VisualizerTickMsg_ReturnsCmd(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.SetSize(80, 24)

	_, cmd := pane.Update(components.VisualizerTickMsg{})
	assert.NotNil(t, cmd, "VisualizerTickMsg should return a re-arm tick command")
}

// filterNonEmpty returns non-empty lines from a multi-line string.
func filterNonEmpty(s string) []string {
	var out []string
	for _, line := range splitLines(s) {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

// splitLines splits a string by newline.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
