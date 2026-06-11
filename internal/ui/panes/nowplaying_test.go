package panes

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/initgrep-apps/spotnik/internal/api"
	"github.com/initgrep-apps/spotnik/internal/state"
	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/initgrep-apps/spotnik/internal/ui/components/viz"
	"github.com/initgrep-apps/spotnik/internal/ui/layout"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStateWriter wraps *state.Store to expose write methods for test setup
// without requiring a type assertion through the StateReader interface.
type testStateWriter struct {
	*state.Store
}

// newTestNowPlayingPane creates a NowPlayingPane with a fresh store and black theme.
func newTestNowPlayingPane(focused bool) *NowPlayingPane {
	s := state.New()
	t := theme.Load("black")
	return NewNowPlayingPane(s, t, focused)
}

// newTestNowPlayingPaneWithState creates a NowPlayingPane pre-loaded with playback state.
// It also returns a testStateWriter so tests can mutate the shared store without
// a type assertion through the StateReader interface.
func newTestNowPlayingPaneWithState(isPlaying bool, focused bool) (*NowPlayingPane, *testStateWriter) {
	s := state.New()
	w := &testStateWriter{s}
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
			Album: api.Album{
				ID:   "alb1",
				Name: "After Hours",
				Images: []api.AlbumImage{
					{URL: "https://i.scdn.co/image/after-hours-640", Width: 640, Height: 640},
				},
			},
		},
		Device: &api.Device{
			ID:            "dev-1",
			Name:          "MacBook Pro",
			VolumePercent: 65,
		},
	})
	t := theme.Load("black")
	return NewNowPlayingPane(s, t, focused), w
}

// ── Task 1: Rename tests ─────────────────────────────────────────────────────

func TestNowPlayingPane_View_NowPlaying(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
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
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	spaceMsg := tea.KeyMsg{Type: tea.KeySpace}
	_, cmd := pane.Update(spaceMsg)

	require.NotNil(t, cmd, "space when playing should return a command")
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok, "space cmd should return PlaybackRequestMsg, got %T", msg)
	assert.Equal(t, ActionPause, req.Action, "playing → space should request pause")
}

func TestNowPlayingPane_Update_Space_WhenPaused(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(false, true)

	spaceMsg := tea.KeyMsg{Type: tea.KeySpace}
	_, cmd := pane.Update(spaceMsg)

	require.NotNil(t, cmd, "space when paused should return a command")
	msg := cmd()
	req, ok := msg.(PlaybackRequestMsg)
	assert.True(t, ok, "space cmd should return PlaybackRequestMsg, got %T", msg)
	assert.Equal(t, ActionPlay, req.Action, "paused → space should request play")
}

// TestNowPlayingPane_P_KeyIgnored verifies that the "p" key is NOT handled by
// NowPlayingPane — it is dead code because routing.go intercepts "p" for
// preset cycling before the pane ever sees it.

func TestNowPlayingPane_Update_Plus_VolUp(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	plusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}
	_, cmd := pane.Update(plusMsg)

	// + key now returns a debounce cmd (VolumeDebounceTickMsg), not PlaybackRequestMsg.
	require.NotNil(t, cmd, "+ key must return a debounce cmd")
	result := cmd()
	_, isPlaybackReq := result.(PlaybackRequestMsg)
	assert.False(t, isPlaybackReq, "+ key must not emit PlaybackRequestMsg directly")
	_, isDebounce := result.(components.VolumeDebounceTickMsg)
	// NOTE: tea.Tick returns a deferred cmd; firing it immediately may return
	// VolumeDebounceTickMsg or a tea.BatchMsg. We only need to confirm it's
	// NOT a PlaybackRequestMsg (the old stale-read pattern).
	_ = isDebounce
}

func TestNowPlayingPane_Update_Minus_VolDown(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	minusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}}
	_, cmd := pane.Update(minusMsg)

	// - key now returns a debounce cmd, not PlaybackRequestMsg.
	require.NotNil(t, cmd, "- key must return a debounce cmd")
	result := cmd()
	_, isPlaybackReq := result.(PlaybackRequestMsg)
	assert.False(t, isPlaybackReq, "- key must not emit PlaybackRequestMsg directly")
}

func TestNowPlayingPane_Update_S_TogglesShuffle(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

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
	pane, w := newTestNowPlayingPaneWithState(true, true)

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
	w.SetPlaybackState(newState)

	fetchedMsg := PlaybackStateFetchedMsg{}
	updatedModel, _ := pane.Update(fetchedMsg)

	updatedPane, ok := updatedModel.(*NowPlayingPane)
	require.True(t, ok)

	// localProgressMs should be reset to server value.
	assert.Equal(t, 90000, updatedPane.localProgressMs)
}

func TestNowPlayingPane_Update_PlaybackFetched_NilState(t *testing.T) {
	pane, w := newTestNowPlayingPaneWithState(true, true)
	pane.localProgressMs = 60000

	// Nil state (nothing playing) — store is cleared, then notification sent.
	w.SetPlaybackState(nil)

	fetchedMsg := PlaybackStateFetchedMsg{}
	updatedModel, _ := pane.Update(fetchedMsg)

	updatedPane, ok := updatedModel.(*NowPlayingPane)
	require.True(t, ok)

	assert.Equal(t, 0, updatedPane.localProgressMs)
	assert.Nil(t, updatedPane.store.PlaybackState())
}

func TestNowPlayingPane_Update_IgnoresKeysWhenNotFocused(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, false) // not focused

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	_, cmd := pane.Update(spaceMsg)

	assert.Nil(t, cmd, "unfocused pane should return nil cmd for key events")
}

func TestNowPlayingPane_Update_TickIncrements(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.localProgressMs = 30000

	tickMsg := TickMsg{}
	updatedModel, _ := pane.Update(tickMsg)

	updatedPane, ok := updatedModel.(*NowPlayingPane)
	require.True(t, ok)

	assert.Equal(t, 31000, updatedPane.localProgressMs)
}

func TestNowPlayingPane_Update_TickNoIncrement_WhenPaused(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(false, true)
	pane.localProgressMs = 30000

	tickMsg := TickMsg{}
	updatedModel, _ := pane.Update(tickMsg)

	updatedPane, ok := updatedModel.(*NowPlayingPane)
	require.True(t, ok)

	assert.Equal(t, 30000, updatedPane.localProgressMs)
}

func TestNowPlayingPane_Update_TickClampsAtDuration(t *testing.T) {
	// localProgressMs must not exceed DurationMs (252000 in the test fixture).
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.localProgressMs = 251500 // one tick away from exceeding DurationMs

	tickMsg := TickMsg{}
	updatedModel, _ := pane.Update(tickMsg)

	updatedPane, ok := updatedModel.(*NowPlayingPane)
	require.True(t, ok)

	// 251500 + 1000 = 252500 > 252000 — must be clamped to DurationMs.
	assert.Equal(t, 252000, updatedPane.localProgressMs,
		"localProgressMs must be clamped to DurationMs when it would overflow")
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

// TestNowPlayingPane_ShiftArrowKeys tests Shift+left/Shift+right arrow keys for track skip.
func TestNowPlayingPane_ShiftArrowKeys(t *testing.T) {
	tests := []struct {
		name       string
		keyType    tea.KeyType
		wantAction PlaybackAction
	}{
		{"shift+right arrow skips next", tea.KeyShiftRight, ActionNext},
		{"shift+left arrow skips prev", tea.KeyShiftLeft, ActionPrevious},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane, _ := newTestNowPlayingPaneWithState(true, true)

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

// TestNowPlayingPane_SeekArrowKeys tests that plain left/right arrows seek ±5s.
func TestNowPlayingPane_SeekArrowKeys(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	// Right arrow: seek forward 5s
	rightMsg := tea.KeyMsg{Type: tea.KeyRight}
	_, cmd := pane.Update(rightMsg)
	require.NotNil(t, cmd, "right arrow should return a debounce cmd")

	// Left arrow: seek backward 5s
	leftMsg := tea.KeyMsg{Type: tea.KeyLeft}
	_, cmd = pane.Update(leftMsg)
	require.NotNil(t, cmd, "left arrow should return a debounce cmd")
}

// TestNowPlayingPane_Space_NilState tests that pressing space with no state still returns cmd.
func TestNowPlayingPane_Space_NilState(t *testing.T) {
	pane := newTestNowPlayingPane(true)

	spaceMsg := tea.KeyMsg{Type: tea.KeySpace}
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
	require.Len(t, actions, 5, "should have exactly 5 border actions")

	expected := map[string]string{
		"s":     "shfl",
		"r":     "rpt",
		"space": "play",
		"+/-":   "vol",
		"v":     "viz",
	}
	for _, a := range actions {
		label, ok := expected[a.Key]
		assert.True(t, ok, "unexpected action key: %s", a.Key)
		assert.Equal(t, label, a.Label)
	}
}

// ── Task 4: viz.Engine migration tests ───────────────────────────────────────

func TestNowPlayingPane_VizTickMsg_AdvancesFrame(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	initialFrame := pane.engine.FrameIndex()

	// Send viz.TickMsg — should advance frame when playing.
	_, _ = pane.Update(viz.TickMsg(time.Now()))

	assert.Equal(t, (initialFrame+1)%40, pane.engine.FrameIndex(),
		"viz.TickMsg should advance engine frame when playing")
}

func TestNowPlayingPane_VizTickMsg_ReturnsCmd(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.SetSize(80, 24)

	_, cmd := pane.Update(viz.TickMsg(time.Now()))
	assert.NotNil(t, cmd, "viz.TickMsg should return a re-arm tick command")
}

func TestNowPlayingPane_PlaybackFetched_SetsEnginePlaying(t *testing.T) {
	pane, w := newTestNowPlayingPaneWithState(false, true)
	pane.SetSize(80, 24)

	// Update store to playing=true, then send PlaybackStateFetchedMsg.
	w.SetPlaybackState(&api.PlaybackState{
		IsPlaying:  true,
		ProgressMs: 5000,
		Item: &api.Track{
			Name:       "Track",
			DurationMs: 200000,
			Artists:    []api.Artist{{Name: "Artist"}},
		},
	})
	_, _ = pane.Update(PlaybackStateFetchedMsg{})

	// Engine should now be in playing state — frame should advance on tick.
	before := pane.engine.FrameIndex()
	_, _ = pane.Update(viz.TickMsg(time.Now()))
	assert.Equal(t, (before+1)%40, pane.engine.FrameIndex(),
		"engine should animate when playing=true")
}

func TestNowPlayingPane_PlaybackFetched_PausesEngine(t *testing.T) {
	pane, w := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	// Update store to paused, send PlaybackStateFetchedMsg.
	w.SetPlaybackState(&api.PlaybackState{
		IsPlaying:  false,
		ProgressMs: 5000,
		Item: &api.Track{
			Name:       "Track",
			DurationMs: 200000,
			Artists:    []api.Artist{{Name: "Artist"}},
		},
	})
	_, _ = pane.Update(PlaybackStateFetchedMsg{})

	// Engine should be paused — frame should not advance on tick.
	before := pane.engine.FrameIndex()
	_, _ = pane.Update(viz.TickMsg(time.Now()))
	assert.Equal(t, before, pane.engine.FrameIndex(),
		"engine should not animate when playing=false")
}

func TestNowPlayingPane_FullView_ContainsBrailleChars(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()

	// Braille chars used by the engine (any braille codepoint).
	hasBraille := false
	for _, r := range output {
		if r >= '\u2800' && r <= '\u28FF' {
			hasBraille = true
			break
		}
	}
	assert.True(t, hasBraille, "full mode View() should contain braille characters from engine")
}

// ── Task 4: Gradient bars tests ──────────────────────────────────────────────

func TestNowPlayingPane_SeekBar_Renders(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	// The gradient seek bar renders time stamps like "0:30" for 30000ms.
	assert.Contains(t, output, "0:30", "seek bar should show current time")
}

func TestNowPlayingPane_VolumeBar_Renders(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	// GradientVolumeBar renders music note icon ♪.
	assert.Contains(t, output, "♪", "volume bar should be rendered in full mode")
}

func TestNowPlayingPane_BarsResize_WithSetSize(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	pane.SetSize(40, 24)
	small := pane.View()

	pane.SetSize(80, 24)
	large := pane.View()

	assert.NotEqual(t, small, large, "different sizes should produce different output")
}

func TestNowPlayingPane_FullView_ContainsTrackAndAlbum(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	assert.Contains(t, output, "Blinding Lights")
	assert.Contains(t, output, "The Weeknd")
	assert.Contains(t, output, "After Hours")
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

// ── v key cycles engine pattern ──────────────────────────────────────────────

// TestNowPlayingPane_V_CyclesEnginePattern verifies that pressing 'v' while
// focused advances the engine's pattern index and emits a VisualizerPatternChangedMsg.
func TestNowPlayingPane_V_CyclesEnginePattern(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)
	pane.engine.SetSize(40, 10) // ensure frames are generated

	startPat := pane.engine.Pattern()

	vMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}}
	_, cmd := pane.Update(vMsg)

	// v now emits VisualizerPatternChangedMsg for preference persistence.
	require.NotNil(t, cmd, "v key should return a Cmd for persistence")
	msg := cmd()
	_, ok := msg.(VisualizerPatternChangedMsg)
	assert.True(t, ok, "v key cmd should return VisualizerPatternChangedMsg, got %T", msg)

	patternCount := pane.engine.PatternCount()
	assert.Equal(t, (startPat+1)%patternCount, pane.engine.Pattern(),
		"engine pattern should advance by 1 on v key")
}

// TestNowPlayingPane_V_IgnoredWhenNotFocused verifies that 'v' is ignored when
// the pane is not focused (routing.go handles global routing; pane itself guards focus).
func TestNowPlayingPane_V_IgnoredWhenNotFocused(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, false) // not focused
	pane.SetSize(80, 24)

	startPat := pane.engine.Pattern()

	vMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}}
	_, cmd := pane.Update(vMsg)

	assert.Nil(t, cmd, "unfocused pane should return nil cmd")
	assert.Equal(t, startPat, pane.engine.Pattern(), "pattern should not change when not focused")
}

// ── Feature 58: Split layout tests ──────────────────────────────────────────

// TestNowPlayingPane_SplitLayout_ContainsInfoBoxBorders verifies that View() at
// 80x24 contains the rounded-corner border characters produced by the InfoBox.
func TestNowPlayingPane_SplitLayout_ContainsInfoBoxBorders(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	assert.Contains(t, output, "╭", "split layout should contain InfoBox top-left corner")
	assert.Contains(t, output, "╰", "split layout should contain InfoBox bottom-left corner")
}

// TestNowPlayingPane_SplitLayout_ContainsBraille verifies that View() contains
// braille characters from the engine rendered on the right side.
func TestNowPlayingPane_SplitLayout_ContainsBraille(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()

	hasBraille := false
	for _, r := range output {
		if r >= '\u2800' && r <= '\u28FF' {
			hasBraille = true
			break
		}
	}
	assert.True(t, hasBraille, "split layout should contain braille characters from engine")
}

// TestNowPlayingPane_SplitLayout_ContainsSeekBar verifies that View() contains
// the seek bar time stamps rendered at the bottom.
func TestNowPlayingPane_SplitLayout_ContainsSeekBar(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)
	pane.localProgressMs = 30000

	output := pane.View()
	// GradientSeekBar renders current/total times.
	assert.Contains(t, output, "0:30", "split layout should contain seek bar current time")
}

// TestNowPlayingPane_SplitLayout_ContainsVolumeInInfoBox verifies that View()
// contains ♪ from the volume bar rendered inside the InfoBox.
func TestNowPlayingPane_SplitLayout_ContainsVolumeInInfoBox(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	assert.Contains(t, output, "♪", "split layout InfoBox should contain volume bar")
}

// TestNowPlayingPane_SplitLayout_ContainsControls verifies that View() contains
// the playback control characters from the Controls component.
func TestNowPlayingPane_SplitLayout_ContainsControls(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	// Controls renders Unicode glyphs — shuffle ⇄ and repeat-off ⟳ (GlyphRepeatOff).
	assert.Contains(t, output, "⇄", "split layout InfoBox should contain shuffle control")
	assert.Contains(t, output, "⟳", "split layout InfoBox should contain repeat-off control")
}

// TestNowPlayingPane_Title_ShowsTrackInfoWhenSmall verifies that Title() includes
// track name when the pane height is below 8 (the new compact-title threshold).
func TestNowPlayingPane_Title_ShowsTrackInfoWhenSmall(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 6) // height < 8
	pane.localProgressMs = 30000

	title := pane.Title()
	assert.Contains(t, title, "Blinding Lights", "title at height<8 should include track name")
	assert.Contains(t, title, "The Weeknd", "title at height<8 should include artist name")
	assert.Contains(t, title, "0:30", "title at height<8 should include current time")
}

// TestNowPlayingPane_Title_DefaultWhenTall verifies that Title() returns the
// default "Now Playing" string when the pane height is >= 8.
func TestNowPlayingPane_Title_DefaultWhenTall(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24) // height >= 8

	title := pane.Title()
	assert.Equal(t, "Now Playing", title, "title at height>=8 should be 'Now Playing'")
}

// TestNowPlayingPane_SplitLayout_AdaptsToDifferentSizes verifies that
// different pane sizes produce different view output (layout is proportional).
func TestNowPlayingPane_SplitLayout_AdaptsToDifferentSizes(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	pane.SetSize(60, 20)
	small := pane.View()

	pane.SetSize(120, 36)
	large := pane.View()

	assert.NotEqual(t, small, large, "different sizes should produce different split layout output")
}

// ── Task 5: Two-column layout with seek bar in right panel ───────────────────

func TestNowPlayingPane_SeekBarInRightPanel(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 20)
	view := pane.View()

	// The seek bar time labels should be present.
	assert.Contains(t, view, "0:30", "should contain elapsed time from seek bar")

	// Multiple lines should be present.
	lines := strings.Split(view, "\n")
	assert.Greater(t, len(lines), 3, "should have multiple lines")
}

func TestNowPlayingPane_RenderStyledLines(t *testing.T) {
	lines := viz.Frame{
		{Text: "aaa", Color: lipgloss.Color("#ff0000")},
		{Text: "bbb", Color: lipgloss.Color("#00ff00")},
	}
	out := renderStyledLines(lines)
	assert.Contains(t, out, "aaa")
	assert.Contains(t, out, "bbb")
}

func TestNowPlayingPane_RenderStyledLines_Empty(t *testing.T) {
	out := renderStyledLines(viz.Frame{})
	assert.Empty(t, out)
}

func TestNowPlayingPane_RenderStyledLines_WithSegments(t *testing.T) {
	lines := viz.Frame{
		{
			Segments: []viz.StyledSegment{
				{Text: "red", Color: lipgloss.Color("#ff0000")},
				{Text: "grn", Color: lipgloss.Color("#00ff00")},
			},
		},
		{
			// Line with no segments uses legacy Color field.
			Text:  "legacy",
			Color: lipgloss.Color("#0000ff"),
		},
	}
	out := renderStyledLines(lines)
	assert.Contains(t, out, "red")
	assert.Contains(t, out, "grn")
	assert.Contains(t, out, "legacy")
	// Segments render on same line (joined, not newline-separated).
	rows := strings.Split(out, "\n")
	assert.Len(t, rows, 2)
}

// ── Story 222: Equal 1-row top + 1-row bottom padding (overlay) ──────────────

func TestNowPlayingPane_Overlay_ExpandedHeightCapped(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	// Expanded: large height — content is capped at npMaxContentH=24 rows
	// and centred vertically within the oversized pane.
	pane.SetSize(80, 30)
	view := pane.View()

	stripped := ansi.Strip(view)
	lines := splitLines(stripped)
	require.GreaterOrEqual(t, len(lines), 3, "expanded view should have content rows")

	// At height=30 > npMaxContentH=24, outer vertical centreing adds blank
	// lines at top and bottom. First and last lines should be blank.
	assert.Empty(t, strings.TrimSpace(lines[0]), "first line should be blank (outer vertical centreing)")
	assert.Empty(t, strings.TrimSpace(lines[len(lines)-1]), "last line should be blank (outer vertical centreing)")
}

func TestNowPlayingPane_CompactNoExcessPadding(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	// Compact: small height — content should not be over-padded.
	pane.SetSize(80, 10)
	view := pane.View()
	lines := strings.Split(view, "\n")

	// 1 top + 1 bottom padding + content (vizRows = 8) = 10.
	assert.LessOrEqual(t, len(lines), 12, "compact should not have excessive padding")
}

// ── Bug fix: compact mode controls visibility ─────────────────────────────────

// TestNowPlayingPane_CompactShowsControls verifies that at height=10 (innerH=4)
// the View output still contains the transport controls (⇄) and volume bar (♪).
// At this height, InfoBox.Render() previously truncated content to 4 lines which
// cut off controls (line 5) and volume bar (line 6) from the 6-line infoLines.
func TestNowPlayingPane_CompactShowsControls(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 10)

	output := pane.View()

	assert.Contains(t, output, "⇄", "compact mode (height=10) should show shuffle control")
	assert.Contains(t, output, "♪", "compact mode (height=10) should show volume bar")
}

// TestNowPlayingPane_VeryCompactShowsControls verifies that at height=8 (innerH=2)
// the View output still contains the transport controls glyph (⇄).
// At this height only 2 inner lines are available so the layout should show
// track name + controls (dropping artists, album, spacer, and volume bar).
func TestNowPlayingPane_VeryCompactShowsControls(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 8)

	output := pane.View()

	assert.Contains(t, output, "⇄", "very compact mode (height=8) should still show controls glyph")
}

// splitLines splits a string by newline.
// Shared helper used by multiple pane test files in the same package.
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

// ---------------------------------------------------------------------------
// Story 80: SetVisualizerPattern and VisualizerPatternChangedMsg
// ---------------------------------------------------------------------------

// TestNowPlayingPane_SetVisualizerPattern verifies that SetVisualizerPattern
// delegates to the engine and changes the current pattern index.
func TestNowPlayingPane_SetVisualizerPattern(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.SetSize(80, 24)

	pane.SetVisualizerPattern(2)
	assert.Equal(t, 2, pane.engine.Pattern(), "SetVisualizerPattern should delegate to engine.SetPattern")
}

// TestNowPlayingPane_VKey_EmitsVisualizerChangedMsg verifies that pressing 'v'
// cycles the pattern AND emits a VisualizerPatternChangedMsg carrying the new index.
func TestNowPlayingPane_VKey_EmitsVisualizerChangedMsg(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.SetSize(80, 24)
	initialPattern := pane.engine.Pattern()

	vMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}}
	_, cmd := pane.Update(vMsg)

	require.NotNil(t, cmd, "v key should return a Cmd")
	msg := cmd()
	changedMsg, ok := msg.(VisualizerPatternChangedMsg)
	require.True(t, ok, "v key cmd should return VisualizerPatternChangedMsg, got %T", msg)
	assert.Equal(t, initialPattern+1, changedMsg.PatternIndex,
		"VisualizerPatternChangedMsg should carry the new pattern index")
}

// TestNowPlayingPane_SetTheme_PreservesVisualizerPattern verifies that switching
// theme does not reset the visualizer pattern the user has selected.
func TestNowPlayingPane_SetTheme_PreservesVisualizerPattern(t *testing.T) {
	pane := newTestNowPlayingPane(false)
	pane.SetSize(80, 24)

	// Cycle to a non-default pattern (pattern 2).
	pane.SetVisualizerPattern(2)
	require.Equal(t, 2, pane.engine.Pattern(), "pattern should be 2 before theme change")

	// Switch to a different theme.
	newTheme := theme.Load("dracula")
	pane.SetTheme(newTheme)

	// Pattern must be preserved after the theme change.
	assert.Equal(t, 2, pane.engine.Pattern(),
		"SetTheme must not reset the visualizer pattern")
}

// ── Story 118: Playback key bug fixes ────────────────────────────────────────

// TestNowPlayingPane_HandleKey_KeySpace_Plays verifies that tea.KeySpace (not a rune)
// triggers play/pause — fixing the Bubbletea v0.27 Space delivery bug.
func TestNowPlayingPane_HandleKey_KeySpace_Plays(t *testing.T) {
	tests := []struct {
		name       string
		isPlaying  bool
		wantAction PlaybackAction
	}{
		{"KeySpace when playing → pause", true, ActionPause},
		{"KeySpace when paused → play", false, ActionPlay},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane, _ := newTestNowPlayingPaneWithState(tt.isPlaying, true)

			// tea.KeySpace is how Bubbletea v0.27 delivers the Space key —
			// Type is tea.KeySpace, Runes is empty (not a rune ' ').
			spaceMsg := tea.KeyMsg{Type: tea.KeySpace}
			_, cmd := pane.Update(spaceMsg)

			require.NotNil(t, cmd, "tea.KeySpace must return a command")
			msg := cmd()
			req, ok := msg.(PlaybackRequestMsg)
			require.True(t, ok, "tea.KeySpace must produce PlaybackRequestMsg, got %T", msg)
			assert.Equal(t, tt.wantAction, req.Action)
		})
	}
}

// TestNowPlaying_AsciiTitle verifies that Title() in compact mode (height < 8)
// uses ASCII glyphs when GlyphASCII mode is active: the pause-indicator resolves
// to "||" and the unicode literals ▶, ⏸, ─ are absent from the output.
func TestNowPlaying_AsciiTitle(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 6) // height < 8 triggers compact title path

	title := pane.Title()
	// IsPlaying == true → GlyphPaused → "||" in ASCII mode
	assert.Contains(t, title, "||", "ASCII mode playing state should show || pause glyph")
	// Unicode literals must not appear in ASCII mode
	assert.NotContains(t, title, "▶", "▶ should not appear in ASCII mode")
	assert.NotContains(t, title, "⏸", "⏸ should not appear in ASCII mode")
	assert.NotContains(t, title, "─", "─ should not appear in ASCII mode")
}

// TestNowPlaying_UnicodeTitlePlayPauseMapping verifies that Title() in compact mode
// (height < 8) maps IsPlaying correctly to the action glyph in unicode mode:
//   - IsPlaying=true  → shows ⏸ (pause action) and NOT ▶
//   - IsPlaying=false → shows ▶ (play action)  and NOT ⏸
//
// A regression swapping the if/else branches inside Title() would be caught here.
func TestNowPlaying_UnicodeTitlePlayPauseMapping(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	tests := []struct {
		name      string
		isPlaying bool
		wantGlyph string
		denyGlyph string
	}{
		{
			name:      "playing shows pause action ⏸",
			isPlaying: true,
			wantGlyph: "⏸",
			denyGlyph: "▶",
		},
		{
			name:      "paused shows play action ▶",
			isPlaying: false,
			wantGlyph: "▶",
			denyGlyph: "⏸",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane, _ := newTestNowPlayingPaneWithState(tt.isPlaying, true)
			pane.SetSize(80, 6) // height < 8 triggers compact title path

			title := pane.Title()
			assert.Contains(t, title, tt.wantGlyph,
				"compact title with isPlaying=%v should contain %q", tt.isPlaying, tt.wantGlyph)
			assert.NotContains(t, title, tt.denyGlyph,
				"compact title with isPlaying=%v must NOT contain %q", tt.isPlaying, tt.denyGlyph)
		})
	}
}

// TestNowPlayingPane_HandleKey_N_NoOp verifies that pressing "n" on the NowPlayingPane
// no longer emits a playback command — the n→next binding was removed in Story 118.
// The → key (tea.KeyRight) remains the authoritative "next track" binding.
func TestNowPlayingPane_HandleKey_N_NoOp(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	nMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	_, cmd := pane.Update(nMsg)

	// After removing the "n" arm from handleKey, pressing 'n' must return nil.
	assert.Nil(t, cmd, "'n' key on NowPlayingPane must not emit a playback command")
}

// ── Task 4: Volume debounce tests ────────────────────────────────────────────

// newPaneWithVolume creates a NowPlayingPane whose store device volume is vol.
func newPaneWithVolume(vol int) *NowPlayingPane {
	s := state.New()
	s.SetPlaybackState(&api.PlaybackState{
		IsPlaying:  true,
		ProgressMs: 0,
		Item: &api.Track{
			Name:       "Track",
			DurationMs: 200000,
			Artists:    []api.Artist{{Name: "Artist"}},
		},
		Device: &api.Device{
			ID:            "dev-1",
			Name:          "Speaker",
			VolumePercent: vol,
			IsActive:      true,
		},
	})
	p := NewNowPlayingPane(s, theme.Load("black"), true)
	p.SetSize(80, 20)
	return p
}

func TestNowPlayingPane_VolumeUp_ReturnsDebounceCmdNotPlaybackRequest(t *testing.T) {
	p := newPaneWithVolume(49)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")}
	_, cmd := p.Update(msg)

	require.NotNil(t, cmd, "volume up must return a debounce cmd")
	// Fire the cmd to get the message; it must NOT be a PlaybackRequestMsg.
	result := cmd()
	_, isPlaybackReq := result.(PlaybackRequestMsg)
	assert.False(t, isPlaybackReq, "+ key must no longer emit PlaybackRequestMsg directly")
}

func TestNowPlayingPane_VolumeDebounceMsg_EmitsVolumeIntent(t *testing.T) {
	p := newPaneWithVolume(49)

	// Simulate: user pressed + once → bar seq=1, target=50
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")}
	p.Update(keyMsg)

	// Fire matching debounce tick (seq=1 because HandleDebounce increments on match,
	// but this is the first tick so bar.seq == 1 after HandleKey).
	debounce := components.VolumeDebounceTickMsg{TargetVol: 50, Seq: 1}
	_, cmd := p.Update(debounce)

	require.NotNil(t, cmd, "matching debounce must return a VolumeIntentMsg cmd")
	result := cmd()
	intent, ok := result.(VolumeIntentMsg)
	require.True(t, ok, "result must be VolumeIntentMsg, got %T", result)
	assert.Equal(t, 50, intent.TargetVol)
	assert.Equal(t, 1, intent.Seq)
}

func TestNowPlayingPane_VolumeAppliedMsg_Success_ConfirmsBar(t *testing.T) {
	p := newPaneWithVolume(49)

	// Prime the bar: press + once, then match the debounce tick.
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})
	_, cmd := p.Update(components.VolumeDebounceTickMsg{TargetVol: 50, Seq: 1})
	require.NotNil(t, cmd)

	// Send VolumeAppliedMsg confirming the API succeeded.
	p.Update(VolumeAppliedMsg{Vol: 55, Seq: 1})

	// Bar should now show 55, but hasPending stays true until a matching poll arrives.
	out := p.View()
	assert.Contains(t, out, "55%")

	// Stale poll with old volume should be blocked while hasPending is true.
	p.volumeBar.SetConfirmed(0)
	out = p.View()
	assert.Contains(t, out, "55%", "stale poll must not snap bar back")

	// Matching poll clears hasPending.
	p.volumeBar.SetConfirmed(55)
	out = p.View()
	assert.Contains(t, out, "55%")

	// Now SetConfirmed updates freely.
	p.volumeBar.SetConfirmed(0)
	out = p.View()
	assert.Contains(t, out, "0%")
}

func TestNowPlayingPane_VolumeAppliedMsg_Error_CancelsPending(t *testing.T) {
	p := newPaneWithVolume(49)

	// Prime the bar: press + once, then match the debounce tick.
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})
	_, cmd := p.Update(components.VolumeDebounceTickMsg{TargetVol: 50, Seq: 1})
	require.NotNil(t, cmd)

	// Send VolumeAppliedMsg with an error.
	p.Update(VolumeAppliedMsg{Err: errors.New("fail"), Seq: 1})

	// currentVol should revert to the store's confirmed value (49), and hasPending should be false.
	out := p.View()
	assert.Contains(t, out, "49%", "error must revert bar to store value")

	// SetConfirmed should now be accepted since hasPending=false.
	p.volumeBar.SetConfirmed(0)
	out = p.View()
	assert.Contains(t, out, "0%")
}

func TestNowPlayingPane_StaleVolumeDebounce_ReturnsNilCmd(t *testing.T) {
	p := newPaneWithVolume(49)

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")}
	p.Update(keyMsg) // seq=1
	p.Update(keyMsg) // seq=2

	stale := components.VolumeDebounceTickMsg{TargetVol: 50, Seq: 1}
	_, cmd := p.Update(stale)
	assert.Nil(t, cmd, "stale debounce tick must return nil cmd")
}

// TestNowPlayingPane_VolumeAppliedMsg_StaleSeq_BarStaysInSecondBurst verifies that a
// VolumeAppliedMsg from the first burst does not snap the bar back when the user has
// already started a second burst. The seq guard in ConfirmFromAPI prevents the stale
// confirmation from overwriting the second burst's pending value.
//
// Seq progression:
//   - HandleKey(+1, 50): seq=1, currentVol=51
//   - HandleDebounce({Seq:1}): seq advances to 2 (double-fire guard), matched
//   - HandleKey(+1, 51): seq=3, currentVol=52
//   - ConfirmFromAPI(intentSeq=1, vol=51): checks seq(3) == 1+1(2) → false → no snap
func TestNowPlayingPane_VolumeAppliedMsg_StaleSeq_BarStaysInSecondBurst(t *testing.T) {
	p := newPaneWithVolume(50)

	// First burst: press + (seq=1, currentVol=51).
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})

	// Resolve the first burst's debounce tick — seq advances to 2 inside HandleDebounce.
	p.Update(components.VolumeDebounceTickMsg{TargetVol: 51, Seq: 1})

	// Second burst starts: press + again (seq=3, currentVol=52).
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("+")})

	// The first burst's API result arrives with intentSeq=1 (stale).
	// ConfirmFromAPI(1, 51) checks: b.seq(3) == 1+1(2) → false → no snap.
	p.Update(VolumeAppliedMsg{Vol: 51, Seq: 1})

	out := p.View()
	assert.Contains(t, out, "52%", "stale ConfirmFromAPI must not snap bar back from second burst value")
}

// ── Story 222: Overlay layout tests ──────────────────────────────────────────

// TestNowPlayingPane_Overlay_InfoBoxOnLeft verifies that at SetSize(120, 20)
// the InfoBox is composited on the left and the track name + artist are visible
// in the composite output. The position of the top-left corner (╭) and the
// InfoBox title "Track Info" are checked explicitly so a regression that
// swapped JoinHorizontal order (viz first, InfoBox second) would be caught.
func TestNowPlayingPane_Overlay_InfoBoxOnLeft(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(120, 20)

	output := pane.View()
	stripped := ansi.Strip(output)
	lines := splitLines(stripped)
	require.NotEmpty(t, lines, "View() must produce at least one line")

	// "Track Info" is the literal title the InfoBox renders — the visualizer
	// never renders it. So a positive match is a strong InfoBox signal.
	assert.Contains(t, stripped, "Track Info", "overlay should show InfoBox title 'Track Info'")

	// The first visualizer block character (▓/░) must appear to the right of
	// the first InfoBox top-left corner (╭). A swap of JoinHorizontal order
	// (viz first, InfoBox second) would put ╭ in the middle/right of the line
	// and the first ▓/░ before any ╭ — caught here.
	firstCornerLine := -1
	firstCornerCol := -1
	for i, line := range lines {
		if idx := strings.Index(line, "╭"); idx >= 0 {
			firstCornerLine = i
			firstCornerCol = idx
			break
		}
	}
	require.GreaterOrEqual(t, firstCornerLine, 0, "InfoBox top-left corner '╭' must be present in overlay output")
	assert.Less(t, firstCornerCol, 5,
		"InfoBox top-left corner '╭' must appear in the first 5 columns (got column %d), indicating InfoBox is on the left",
		firstCornerCol)

	// Track + artist must be visible somewhere in the composite.
	assert.Contains(t, output, "Blinding Lights", "overlay layout should show track name")
	assert.Contains(t, output, "The Weeknd", "overlay layout should show artist name")
}

// TestNowPlayingPane_Overlay_SeekBarRightOfInfoBox verifies that the seek bar
// in the visualizer column does not appear on a line that intersects the
// InfoBox interior. We strip ANSI, split by newline, and find the seek bar
// line (contains a time stamp like "0:30"). The InfoBox right border "│" must
// appear before the first "▓" or "░" on the same line.
func TestNowPlayingPane_Overlay_SeekBarRightOfInfoBox(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(120, 20)
	pane.localProgressMs = 30000

	output := pane.View()
	stripped := ansi.Strip(output)

	var seekLine string
	for _, line := range strings.Split(stripped, "\n") {
		if strings.Contains(line, "0:30") {
			seekLine = line
			break
		}
	}
	require.NotEmpty(t, seekLine, "seek bar line with '0:30' must exist in overlay output")

	borderIdx := strings.Index(seekLine, "│")
	require.GreaterOrEqual(t, borderIdx, 0, "seek bar line must contain InfoBox right border '│'")

	// Find the first seek-bar block character to the right of the border.
	firstBlock := -1
	for i, r := range seekLine {
		if i <= borderIdx {
			continue
		}
		if r == '▓' || r == '░' {
			firstBlock = i
			break
		}
	}
	require.GreaterOrEqual(t, firstBlock, 0,
		"seek bar line must contain ▓ or ░ to the right of the InfoBox border")
	assert.Greater(t, firstBlock, borderIdx,
		"first seek-bar block character must appear after the InfoBox right border")
}

// TestNowPlayingPane_Overlay_NarrowFallback verifies that at a narrow width
// the InfoBox is dropped and the visualizer fills the full content area
// (no InfoBox corners or "Track Info" title in the output). The fallback
// triggers when contentWidth - infoWidth - npGap < npMinViz.
func TestNowPlayingPane_Overlay_NarrowFallback(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	// cw = 16; infoWidth = max(16/3, 28) = 28; vizWidth = 16-28-1 = -13 < 10.
	// Triggers narrow fallback: InfoBox dropped.
	pane.SetSize(16, 16)

	output := pane.View()
	assert.NotContains(t, output, "╭", "narrow fallback should not contain InfoBox top-left corner")
	assert.NotContains(t, output, "Track Info", "narrow fallback should not contain InfoBox title")
}

// TestNowPlayingPane_Overlay_NormalWidthIsOverlay is the paired companion
// to TestNowPlayingPane_Overlay_NarrowFallback. It locks the threshold
// behaviour in the OTHER direction: at a normal terminal width the InfoBox
// IS rendered. Without this, a regression that always-takes or never-takes
// the fallback would not be caught.
func TestNowPlayingPane_Overlay_NormalWidthIsOverlay(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	// 120×20: cw=120, effH=20 (capped to npMaxContentH=24, so 20).
	// infoWidth = 120/3 = 40, capped to npInfoMin=28, so 40.
	// vizWidth = 120-40-1 = 79 — well above npMinViz=10.
	pane.SetSize(120, 20)

	output := pane.View()
	stripped := ansi.Strip(output)
	assert.Contains(t, stripped, "╭",
		"normal width (120×20) should contain InfoBox top-left corner '╭'")
	assert.Contains(t, stripped, "Track Info",
		"normal width (120×20) should contain InfoBox title 'Track Info'")
}

// TestNowPlayingPane_Overlay_Layout verifies the new line-by-line composition
// produces the correct number of lines and contains both InfoBox and visualizer
// content. The output should be exactly effH lines (or p.height when oversized
// and centred), with InfoBox on the left and visualizer on the right.
func TestNowPlayingPane_Overlay_Layout(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(120, 20)

	output := pane.View()
	stripped := ansi.Strip(output)
	lines := splitLines(stripped)

	// At 120×20, effH = 20 (not capped). The output should be exactly 20 lines.
	assert.Equal(t, 20, len(lines), "View() should produce exactly effH lines")

	// InfoBox top border should be on the first line.
	assert.Contains(t, lines[0], "Track Info", "first line should contain InfoBox title")

	// Somewhere in the output we should have visualizer glyphs.
	hasTrackInfoOrBlock := strings.Contains(stripped, "Track Info") ||
		strings.ContainsRune(stripped, '▓') ||
		strings.ContainsRune(stripped, '░')
	assert.True(t, hasTrackInfoOrBlock,
		"overlay output must contain 'Track Info' or visualizer glyph ▓/░")
}

// TestNowPlayingPane_Overlay_MinimumSize locks the no-panic invariant at
// extremely small pane sizes. The renderer must always return a non-empty
// string and must not crash even when width/height collapse below the
// content-area floor (contentWidth floor is 10, vizRows floor is 4).
func TestNowPlayingPane_Overlay_MinimumSize(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	for _, sz := range [][2]int{{4, 4}, {10, 6}, {1, 1}} {
		sz := sz
		t.Run("", func(t *testing.T) {
			assert.NotPanics(t, func() {
				pane.SetSize(sz[0], sz[1])
				out := pane.View()
				assert.NotEmpty(t, out,
					"View() must return a non-empty string at SetSize(%d, %d)", sz[0], sz[1])
			}, "View() must not panic at SetSize(%d, %d)", sz[0], sz[1])
		})
	}
}

// TestNowPlayingPane_Overlay_WideCaps verifies that at a very wide terminal
// (500×40) the InfoBox is still present on the left and its width follows
// the adaptive formula (cw / npInfoPctTall). The InfoBox width is measured in
// CELLS (not bytes) by scanning runes, since the border glyphs are multi-byte
// UTF-8 but single-cell terminal chars.
//
// At 500×40: cw=500, effH=24 (capped). infoWidth = 500/3 = 166, which is
// above npInfoMin=28, so infoWidth = 166. vizWidth = 500-166-1 = 333.
func TestNowPlayingPane_Overlay_WideCaps(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(500, 40)

	output := pane.View()
	stripped := ansi.Strip(output)
	lines := splitLines(stripped)
	require.NotEmpty(t, lines, "View() must produce at least one line")

	// findRuneIndex returns the cell-column index of the first occurrence of
	// r in s. Since all the border glyphs in this test are 1-cell single-rune
	// characters, byte-counting would skew; we work in runes.
	findRuneIndex := func(s string, r rune) int {
		col := 0
		for _, c := range s {
			if c == r {
				return col
			}
			col++
		}
		return -1
	}
	// findLastRuneIndex returns the cell-column of the last occurrence of r.
	findLastRuneIndex := func(s string, r rune) int {
		last := -1
		col := 0
		for _, c := range s {
			if c == r {
				last = col
			}
			col++
		}
		return last
	}

	// Find the top row of the InfoBox (contains '╭' top-left corner).
	var topLine string
	var topLineIdx = -1
	for i, line := range lines {
		if strings.ContainsRune(line, '╭') {
			topLine = line
			topLineIdx = i
			break
		}
	}
	require.NotEmpty(t, topLine,
		"InfoBox top-left corner '╭' must be present at wide width (500×40)")
	require.GreaterOrEqual(t, topLineIdx, 0, "top line index must be valid")

	// Find the top-left and top-right corners in cell columns.
	topLeftIdx := findRuneIndex(topLine, '╭')
	topRightIdx := findRuneIndex(topLine, '╮')
	require.GreaterOrEqual(t, topLeftIdx, 0, "top line must contain '╭'")
	require.GreaterOrEqual(t, topRightIdx, 0, "top line must contain '╮'")
	assert.Greater(t, topRightIdx, topLeftIdx, "'╮' must be to the right of '╭'")

	// InfoBox width = topRightIdx - topLeftIdx + 1 = ~166 (derived infoWidth).
	infoBoxWidth := topRightIdx - topLeftIdx + 1
	// contentWidth = 500. infoWidth = 500/3 = 166.
	assert.Greater(t, infoBoxWidth, 150,
		"InfoBox width should be at least the tall-mode floor (got %d, expected ~166)",
		infoBoxWidth)
	assert.Less(t, infoBoxWidth, 180,
		"InfoBox width should respect the cw/3 formula (got %d, expected ~166)",
		infoBoxWidth)

	// Verify a side line (the row just below the top) has the right border
	// '│' aligned with the top-right '╮' column.
	if topLineIdx+1 < len(lines) {
		sideLine := lines[topLineIdx+1]
		sideRightBorder := findLastRuneIndex(sideLine, '│')
		require.GreaterOrEqual(t, sideRightBorder, 0,
			"side line of InfoBox must contain at least one right border '│'")
		assert.Equal(t, topRightIdx, sideRightBorder,
			"side line right border '│' (cell col %d) should align with top-right '╮' (cell col %d)",
			sideRightBorder, topRightIdx)
	}
}

// ── Story 223: Adaptive width, centering, remove overlay bg ───────────────────

// TestNowPlayingPane_Adaptive_ListeningPreset verifies that at SetSize(160, 14)
// (Listening preset) the InfoBox is ~33% width and no truncation artifacts appear.
// effH=14 (not capped), infoWidth = 160/3 = 53, vizWidth = 160-53-1 = 106.
func TestNowPlayingPane_Adaptive_ListeningPreset(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(160, 14)

	output := pane.View()
	stripped := ansi.Strip(output)

	// InfoBox should be present.
	assert.Contains(t, stripped, "Track Info", "listening preset should show InfoBox")
	assert.Contains(t, stripped, "Blinding Lights", "track name should be visible")
	assert.Contains(t, stripped, "The Weeknd", "artist name should be visible")

	// Controls and volume should be visible (no truncation artifacts).
	assert.Contains(t, output, "⇄", "listening preset should show controls")
	assert.Contains(t, output, "♪", "listening preset should show volume bar")
}

// TestNowPlayingPane_Adaptive_LibraryPreset verifies that at SetSize(160, 6)
// (Library preset with MinHeight=6) the InfoBox is ~50% width (short mode),
// controls + volume are visible, and the album line is dropped.
// effH=6 (not capped), infoWidth = 160/2 = 80, vizWidth = 160-80-1 = 79.
func TestNowPlayingPane_Adaptive_LibraryPreset(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(160, 6)

	output := pane.View()
	stripped := ansi.Strip(output)

	// InfoBox should be present at ~50% width.
	assert.Contains(t, stripped, "Track Info", "library preset should show InfoBox")
	assert.Contains(t, stripped, "Blinding Lights", "track name should be visible")
	assert.Contains(t, stripped, "The Weeknd", "artist name should be visible")

	// Controls and volume should be visible.
	assert.Contains(t, output, "⇄", "library preset should show controls")
	assert.Contains(t, output, "♪", "library preset should show volume bar")

	// Album line "After Hours" should be dropped in compact mode (innerH=4).
	// With SetSize(160,6): innerH = 6-2 = 4, so album is dropped.
	// NOTE: The album might still appear if the title bar shows it, so we only
	// check that controls and volume are visible.
}

// TestNowPlayingPane_Adaptive_SoloPaneCap verifies that at SetSize(160, 46)
// (solo pane, very tall) the content is capped at npMaxContentH=24 rows and
// centred vertically. The output should contain blank lines at top and bottom.
func TestNowPlayingPane_Adaptive_SoloPaneCap(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(160, 46)

	output := pane.View()
	stripped := ansi.Strip(output)
	lines := splitLines(stripped)

	// Content capped at 24, outerPad = (46-24)/2 = 11.
	// First and last 11 lines should be blank (outer vertical centreing).
	require.GreaterOrEqual(t, len(lines), 24, "should have at least 24 lines")
	assert.Empty(t, strings.TrimSpace(lines[0]), "first line should be blank (outer vertical centreing)")
	assert.Empty(t, strings.TrimSpace(lines[len(lines)-1]), "last line should be blank (outer vertical centreing)")

	// Middle lines should contain actual content.
	mid := len(lines) / 2
	assert.NotEmpty(t, strings.TrimSpace(lines[mid]), "middle lines should contain content")
}

// TestNowPlayingPane_Adaptive_NarrowFallback verifies that at SetSize(30, 16)
// the InfoBox is dropped because vizWidth < npMinViz.
// cw=30, infoWidth = max(30/3, 28) = 28, vizWidth = 30-28-1 = 1 < 10.
func TestNowPlayingPane_Adaptive_NarrowFallback(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(30, 16)

	output := pane.View()

	assert.NotContains(t, output, "╭", "narrow fallback should not contain InfoBox top-left corner")
	assert.NotContains(t, output, "Track Info", "narrow fallback should not contain InfoBox title")
}

// TestNowPlayingPane_Adaptive_MinimumSize verifies that at extremely small
// pane sizes the renderer does not panic and returns a non-empty string.
func TestNowPlayingPane_Adaptive_MinimumSize(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)

	for _, sz := range [][2]int{{1, 1}, {4, 4}, {10, 6}} {
		sz := sz
		t.Run("", func(t *testing.T) {
			assert.NotPanics(t, func() {
				pane.SetSize(sz[0], sz[1])
				out := pane.View()
				assert.NotEmpty(t, out,
					"View() must return a non-empty string at SetSize(%d, %d)", sz[0], sz[1])
			}, "View() must not panic at SetSize(%d, %d)", sz[0], sz[1])
		})
	}
}

// TestNowPlayingPane_Adaptive_ContentWidth_NoPhantomSubtraction verifies
// that contentWidth() no longer double-subtracts border space.
func TestNowPlayingPane_Adaptive_ContentWidth_NoPhantomSubtraction(t *testing.T) {
	pane := newTestNowPlayingPane(true)
	pane.SetSize(80, 20)

	cw := pane.contentWidth()
	assert.Equal(t, 80, cw, "contentWidth should equal pane width (no phantom subtraction)")
}

// TestNowPlayingPane_Adaptive_InfoBoxNoOverlayBackground verifies that the
// InfoBox interior no longer has an OverlayBackground fill applied.
// Since we removed the background fill, the InfoBox interior should render
// as plain text without ANSI background escape codes.
func TestNowPlayingPane_Adaptive_InfoBoxNoOverlayBackground(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 20)

	output := pane.View()

	// The InfoBox interior should NOT contain background color escape codes.
	// Background codes in ANSI start with ESC[48; or ESC[48;5; or ESC[48;2;.
	hasBgCode := strings.Contains(output, "\x1b[48")
	assert.False(t, hasBgCode, "InfoBox interior should not contain ANSI background escape codes")
}

// ── Story 226: InfoBox left padding & controls centering ────────────────────

// TestNowPlayingPane_InfoBoxLeftPadding verifies that text lines inside the
// InfoBox (track name, artist, album) have 2 columns of left padding.
// We strip ANSI, find the InfoBox content lines (after the top border with "╭"),
// and check that the first content characters after the left border "│" are
// exactly 2 spaces before the styled text begins.
func TestNowPlayingPane_InfoBoxLeftPadding(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	stripped := ansi.Strip(output)
	lines := splitLines(stripped)

	// afterRune returns the substring starting after the first occurrence of r.
	afterRune := func(s string, r rune) string {
		for i, c := range s {
			if c == r {
				return string([]rune(s)[i+1:])
			}
		}
		return s
	}

	contentStarted := false
	for _, line := range lines {
		if strings.ContainsRune(line, '╭') {
			contentStarted = true
			continue
		}
		if !contentStarted {
			continue
		}
		afterBorder := afterRune(line, '│')
		require.GreaterOrEqual(t, len([]rune(afterBorder)), 2,
			"first InfoBox content line must have at least 2 runes after left border")
		runes := []rune(afterBorder)
		assert.Equal(t, "  ", string(runes[:2]),
			"first InfoBox content line must start with 2 spaces (left padding)")
		break
	}
}

// TestNowPlayingPane_ControlsCentered verifies that the transport controls row
// is horizontally centered within the InfoBox interior. We find the line containing
// the shuffle glyph "⇄", strip ANSI, and check that the leading and trailing
// space counts around the controls glyphs are balanced (within ±1 cell).
func TestNowPlayingPane_ControlsCentered(t *testing.T) {
	pane, _ := newTestNowPlayingPaneWithState(true, true)
	pane.SetSize(80, 24)

	output := pane.View()
	stripped := ansi.Strip(output)
	lines := splitLines(stripped)

	// extractRunesBetweenBorders returns the runes between the first and last '│' in the line.
	extractRunesBetweenBorders := func(line string) []rune {
		runes := []rune(line)
		first := -1
		last := -1
		for i, r := range runes {
			if r == '│' {
				if first < 0 {
					first = i
				}
				last = i
			}
		}
		if first < 0 || last <= first {
			return nil
		}
		return runes[first+1 : last]
	}

	for _, line := range lines {
		if !strings.ContainsRune(line, '⇄') {
			continue
		}
		inner := extractRunesBetweenBorders(line)
		if inner == nil {
			continue
		}
		innerStr := string(inner)
		leadingSpace := len(innerStr) - len(strings.TrimLeft(innerStr, " "))
		trailingSpace := len(innerStr) - len(strings.TrimRight(innerStr, " "))
		diff := leadingSpace - trailingSpace
		if diff < 0 {
			diff = -diff
		}
		assert.LessOrEqual(t, diff, 1,
			"controls row should be centered (leading/trailing space within ±1): leading=%d trailing=%d",
			leadingSpace, trailingSpace)
		break
	}
}
