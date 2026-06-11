package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func newTestSeekBar(width int) *GradientSeekBar {
	t := theme.Load("black")
	b := NewGradientSeekBar(t)
	b.SetWidth(width)
	return b
}

func newTestGradientVolumeBar(width int) *GradientVolumeBar {
	t := theme.Load("black")
	b := NewGradientVolumeBar(t)
	b.SetWidth(width)
	return b
}

// --------------------------------------------------------------------------
// interpolateHex
// --------------------------------------------------------------------------

func TestInterpolateHex_AtZero(t *testing.T) {
	c := interpolateHex("#ff0000", "#0000ff", 0.0)
	assert.Equal(t, lipgloss.Color("#ff0000"), c)
}

func TestInterpolateHex_AtOne(t *testing.T) {
	c := interpolateHex("#ff0000", "#0000ff", 1.0)
	assert.Equal(t, lipgloss.Color("#0000ff"), c)
}

func TestInterpolateHex_AtHalf(t *testing.T) {
	c := interpolateHex("#ff0000", "#0000ff", 0.5)
	// Expect midpoint: R=128, G=0, B=128 → #800080
	// (255 * 0.5 = 127.5 → rounds to 128 = 0x80)
	assert.Equal(t, lipgloss.Color("#800080"), c)
}

func TestInterpolateHex_Clamp(t *testing.T) {
	// t < 0 should clamp to color1
	c := interpolateHex("#ff0000", "#0000ff", -0.5)
	assert.Equal(t, lipgloss.Color("#ff0000"), c)

	// t > 1 should clamp to color2
	c = interpolateHex("#ff0000", "#0000ff", 1.5)
	assert.Equal(t, lipgloss.Color("#0000ff"), c)
}

// --------------------------------------------------------------------------
// GradientSeekBar
// --------------------------------------------------------------------------

func TestGradientSeekBar_ZeroProgress(t *testing.T) {
	b := newTestSeekBar(50)
	out := b.Render(0, 300000)
	assert.NotContains(t, out, "█", "zero progress should show no filled chars")
	assert.Contains(t, out, "░", "zero progress should show empty chars")
}

func TestGradientSeekBar_HalfProgress(t *testing.T) {
	b := newTestSeekBar(50)
	out := b.Render(150000, 300000)
	assert.Contains(t, out, "█")
	assert.Contains(t, out, "░")
}

func TestGradientSeekBar_FullProgress(t *testing.T) {
	b := newTestSeekBar(50)
	out := b.Render(300000, 300000)
	assert.Contains(t, out, "█")
	assert.NotContains(t, out, "░", "full progress should show no empty chars")
}

func TestGradientSeekBar_TimeLabel_Format(t *testing.T) {
	b := newTestSeekBar(60)
	// 2:30 = 150000ms; 5:00 = 300000ms
	out := b.Render(150000, 300000)
	assert.Contains(t, out, "2:30", "should show elapsed time 2:30")
	assert.Contains(t, out, "5:00", "should show total time 5:00")
}

func TestGradientSeekBar_ZeroDuration(t *testing.T) {
	b := newTestSeekBar(50)
	// Must not panic.
	require.NotPanics(t, func() {
		out := b.Render(0, 0)
		assert.NotContains(t, out, "NaN")
	})
}

func TestGradientSeekBar_NegativeProgress(t *testing.T) {
	b := newTestSeekBar(50)
	// Negative progressMs should not produce a wider-than-expected bar or panic.
	require.NotPanics(t, func() {
		out := b.Render(-5000, 300000)
		// Bar should be all empty (no filled gradient chars).
		assert.NotEmpty(t, out)
	})
}

func TestGradientSeekBar_WidthChanges(t *testing.T) {
	b40 := newTestSeekBar(40)
	b80 := newTestSeekBar(80)

	out40 := b40.Render(120000, 300000)
	out80 := b80.Render(120000, 300000)
	// Wider bar should produce more characters (before ANSI stripping, compare lengths).
	assert.Greater(t, len(out80), len(out40), "wider bar should produce more output")
}

func TestGradientSeekBar_TimeLabelPadded(t *testing.T) {
	b := newTestSeekBar(50)
	// 1:01 = 61000ms — single digit seconds should be zero-padded
	out := b.Render(61000, 120000)
	assert.Contains(t, out, "1:01")
}

// --------------------------------------------------------------------------
// GradientVolumeBar
// --------------------------------------------------------------------------

func TestGradientVolumeBar_ZeroVolume(t *testing.T) {
	b := newTestGradientVolumeBar(30)
	b.SetConfirmed(0)
	out := b.Render()
	assert.Contains(t, out, "♪", "zero volume should show music note icon")
	assert.Contains(t, out, "0%")
	assert.NotContains(t, out, "■", "zero volume should show no filled chars (old ■ char)")
	assert.NotContains(t, out, "VOL", "should not use old VOL prefix")
}

func TestGradientVolumeBar_LowVolume_Gradient1(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	// 25% is in the 0-33% band — should use Gradient1 color.
	// In no-color terminal, just verify structural output.
	b.SetConfirmed(25)
	out := b.Render()
	assert.Contains(t, out, "♪")
	assert.Contains(t, out, "25%")
	assert.Contains(t, out, "█", "filled blocks use █")
	assert.NotContains(t, out, "■", "old ■ char no longer used")
	assert.NotContains(t, out, "VOL", "should not use old VOL prefix")
}

func TestGradientVolumeBar_MidVolume_Gradient2(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	// 50% is in the 34-66% band — should use Gradient2 color.
	b.SetConfirmed(50)
	out := b.Render()
	assert.Contains(t, out, "♪")
	assert.Contains(t, out, "50%")
	assert.Contains(t, out, "█", "filled blocks use █")
	assert.NotContains(t, out, "■", "old ■ char no longer used")
	assert.NotContains(t, out, "VOL", "should not use old VOL prefix")
}

func TestGradientVolumeBar_HighVolume_Gradient3(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	// 80% is in the 67-100% band — should use Gradient3 color.
	b.SetConfirmed(80)
	out := b.Render()
	assert.Contains(t, out, "♪")
	assert.Contains(t, out, "80%")
	assert.Contains(t, out, "█", "filled blocks use █")
	assert.NotContains(t, out, "■", "old ■ char no longer used")
	assert.NotContains(t, out, "VOL", "should not use old VOL prefix")
}

func TestGradientVolumeBar_FullVolume(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	b.SetConfirmed(100)
	out := b.Render()
	assert.Contains(t, out, "100%")
	assert.NotContains(t, out, "□", "full volume should have no empty chars (old □ char)")
	// At 100% there are no empty cells so ░ must not appear.
	assert.NotContains(t, out, "░", "full volume should have no empty cells")
}

func TestGradientVolumeBar_Format(t *testing.T) {
	b := newTestGradientVolumeBar(30)
	b.SetConfirmed(50)
	out := b.Render()
	assert.Contains(t, out, "♪", "should contain music note icon")
	assert.Contains(t, out, "█", "should contain full block character")
	// Empty cells now use ░ (GlyphBarEmpty) per design system §5.7.
	assert.Contains(t, out, "░", "should contain empty block character (░)")
	assert.Contains(t, out, "50%")
	assert.NotContains(t, out, "VOL", "should not use old VOL prefix")
	assert.NotContains(t, out, "■", "should not use old filled block character ■")
	assert.NotContains(t, out, "□", "should not use old empty block character □")
}

func TestGradientVolumeBar_MuteIcon_Volume0(t *testing.T) {
	b := newTestGradientVolumeBar(30)
	// Volume = 0: ♪ still present but in muted color
	b.SetConfirmed(0)
	out := b.Render()
	assert.Contains(t, out, "♪")
	assert.Contains(t, out, "0%")
}

func TestGradientVolumeBar_ClampHigh(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	// Volume > 100 should be clamped to 100.
	b.SetConfirmed(150)
	out := b.Render()
	assert.Contains(t, out, "100%")
}

func TestGradientVolumeBar_ClampLow(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	// Volume < 0 should be clamped to 0.
	b.SetConfirmed(-5)
	out := b.Render()
	assert.Contains(t, out, "0%")
}

func TestGradientVolumeBar_WidthChanges(t *testing.T) {
	b30 := newTestGradientVolumeBar(30)
	b60 := newTestGradientVolumeBar(60)

	b30.SetConfirmed(50)
	out30 := b30.Render()
	b60.SetConfirmed(50)
	out60 := b60.Render()

	lines30 := strings.Split(out30, "\n")[0]
	lines60 := strings.Split(out60, "\n")[0]

	// Wider bar should produce longer line.
	assert.Greater(t, len(lines60), len(lines30), "wider bar should produce longer line")
}

// --------------------------------------------------------------------------
// Threshold boundary tests
// --------------------------------------------------------------------------

func TestGradientVolumeBar_At33_Gradient1(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	b.SetConfirmed(33)
	out := b.Render()
	// 33% should still be in band 1 (0-33%).
	assert.Contains(t, out, "33%")
}

func TestGradientVolumeBar_At34_Gradient2(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	b.SetConfirmed(34)
	out := b.Render()
	// 34% crosses into band 2 (34-66%).
	assert.Contains(t, out, "34%")
}

func TestGradientVolumeBar_At66_Gradient2(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	b.SetConfirmed(66)
	out := b.Render()
	assert.Contains(t, out, "66%")
}

func TestGradientVolumeBar_At67_Gradient3(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	b.SetConfirmed(67)
	out := b.Render()
	// 67% crosses into band 3 (67-100%).
	assert.Contains(t, out, "67%")
}

// --------------------------------------------------------------------------
// Partial-block rendering tests (barWidth=14, the default)
// --------------------------------------------------------------------------

// TestGradientVolumeBar_PartialBlocks verifies the partial-block fill algorithm at barWidth=14
// for boundary volumes. When SetWidth is 0 the default barWidth of 14 is used.
//
// Empty cells now use ░ (GlyphBarEmpty, §5.7). The dead zone that previously skipped
// partial blocks when fraction < 1/8 has been removed — the §5.7 threshold algorithm
// emits ▏ for any non-zero remainder.
func TestGradientVolumeBar_PartialBlocks(t *testing.T) {
	// partialChars lists all 7 partial-block glyphs (▏▎▍▌▋▊▉) for negative assertions.
	partialChars := []string{"▏", "▎", "▍", "▌", "▋", "▊", "▉"}

	tests := []struct {
		name        string
		volume      int
		wantFull    int    // expected number of full █ blocks
		wantPartial string // expected partial-block character, or "" if none
		wantEmpty   int    // expected number of ░ empty characters
	}{
		// vol=0: filledF=0.0 → 0 full, no partial, 14 empty
		{name: "0pct", volume: 0, wantFull: 0, wantPartial: "", wantEmpty: 14},
		// vol=1: filledF=0.14 → 0 full, remainder=0.14 → ▏ (any remainder > 0), 13 empty
		{name: "1pct", volume: 1, wantFull: 0, wantPartial: "▏", wantEmpty: 13},
		// vol=7: filledF=0.98 → 0 full, remainder=0.98 ≥ 7/8 → ▉, 13 empty
		{name: "7pct", volume: 7, wantFull: 0, wantPartial: "▉", wantEmpty: 13},
		// vol=8: filledF=1.12 → 1 full, remainder=0.12 → ▏ (dead zone removed per §5.7), 12 empty
		{name: "8pct", volume: 8, wantFull: 1, wantPartial: "▏", wantEmpty: 12},
		// vol=14: filledF=1.96 → 1 full, remainder=0.96 ≥ 7/8 → ▉, 12 empty
		{name: "14pct", volume: 14, wantFull: 1, wantPartial: "▉", wantEmpty: 12},
		// vol=31: filledF=4.34 → 4 full, remainder=0.34 ≥ 3/8=0.375? No: 0.34 < 0.375 → ▏ (< 2/8=0.25? No)
		// 0.34 ≥ 2/8(0.25) → ▎, 9 empty
		{name: "31pct", volume: 31, wantFull: 4, wantPartial: "▎", wantEmpty: 9},
		// vol=50: filledF=7.0 → 7 full, remainder=0 → no partial, 7 empty
		{name: "50pct", volume: 50, wantFull: 7, wantPartial: "", wantEmpty: 7},
		// vol=99: filledF=13.86 → 13 full, remainder=0.86 ≥ 7/8=0.875? No: 0.86 < 0.875 → ≥ 6/8=0.75 → ▊, 0 empty
		{name: "99pct", volume: 99, wantFull: 13, wantPartial: "▊", wantEmpty: 0},
		// vol=100: filledF=14.0 → 14 full, remainder=0 → no partial, 0 empty
		{name: "100pct", volume: 100, wantFull: 14, wantPartial: "", wantEmpty: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newTestGradientVolumeBar(0) // 0 → use default barWidth=14
			b.SetConfirmed(tt.volume)
			out := b.Render()

			gotFull := strings.Count(out, "█")
			gotEmpty := strings.Count(out, "░") // empty char is now ░ (GlyphBarEmpty §5.7)
			assert.Equal(t, tt.wantFull, gotFull, "full block count mismatch for volume=%d", tt.volume)
			assert.Equal(t, tt.wantEmpty, gotEmpty, "empty block count mismatch for volume=%d", tt.volume)
			assert.NotContains(t, out, "□", "old □ char must not appear after migration")

			// Total cells must always equal barWidth (14) regardless of partial-block presence.
			partialPresent := 0
			if tt.wantPartial != "" {
				partialPresent = 1
			}
			assert.Equal(t, gradientVolumeBarWidth, gotFull+gotEmpty+partialPresent,
				"total cell count must equal barWidth for volume=%d", tt.volume)

			if tt.wantPartial == "" {
				for _, pc := range partialChars {
					assert.NotContains(t, out, pc, "should have no partial block for volume=%d", tt.volume)
				}
			} else {
				assert.Contains(t, out, tt.wantPartial, "partial block mismatch for volume=%d", tt.volume)
				// Exactly one partial block char should be present.
				gotPartialCount := strings.Count(out, tt.wantPartial)
				assert.Equal(t, 1, gotPartialCount, "should be exactly one partial block for volume=%d", tt.volume)
			}
		})
	}
}

// --------------------------------------------------------------------------
// GradientVolumeBar — smart component (debounce state)
// --------------------------------------------------------------------------

func TestVolumeBar_HandleKey_UpdatesCurrentVolImmediately(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	b.SetConfirmed(50)
	cmd := b.HandleKey(+1, 50)
	assert.NotNil(t, cmd, "HandleKey must return a non-nil debounce cmd")
	// Verify seq and hasPending via HandleDebounce.
	b.SetConfirmed(0)
	// If hasPending, SetConfirmed is ignored — currentVol stays at 51.
	matched, vol, _ := b.HandleDebounce(VolumeDebounceTickMsg{TargetVol: 51, Seq: 1})
	assert.True(t, matched)
	assert.Equal(t, 51, vol)
}

func TestVolumeBar_HandleKey_AccumulatesFromPending(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	b.SetConfirmed(50)
	b.HandleKey(+1, 50) // seq=1, currentVol=51, pending
	b.HandleKey(+1, 50) // must use currentVol=51 as base, not confirmedVol=50 → currentVol=52, seq=2
	matched, vol, _ := b.HandleDebounce(VolumeDebounceTickMsg{TargetVol: 52, Seq: 2})
	assert.True(t, matched, "second keypress uses pending vol as base")
	assert.Equal(t, 52, vol)
}

func TestVolumeBar_HandleKey_ClampsAtMax(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	b.SetConfirmed(100)
	b.HandleKey(+1, 100)
	matched, vol, _ := b.HandleDebounce(VolumeDebounceTickMsg{TargetVol: 100, Seq: 1})
	assert.True(t, matched)
	assert.Equal(t, 100, vol, "clamped at 100")
}

func TestVolumeBar_HandleKey_ClampsAtMin(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	b.SetConfirmed(0)
	b.HandleKey(-1, 0)
	matched, vol, _ := b.HandleDebounce(VolumeDebounceTickMsg{TargetVol: 0, Seq: 1})
	assert.True(t, matched)
	assert.Equal(t, 0, vol, "clamped at 0")
}

func TestVolumeBar_HandleDebounce_StaleDiscarded(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	b.SetConfirmed(50)
	b.HandleKey(+1, 50) // seq=1
	b.HandleKey(+1, 50) // seq=2 — supersedes seq=1
	matched, _, _ := b.HandleDebounce(VolumeDebounceTickMsg{TargetVol: 51, Seq: 1})
	assert.False(t, matched, "stale seq must be discarded")
}

func TestVolumeBar_HandleDebounce_DoesNotClearPending(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	b.SetConfirmed(50)
	b.HandleKey(+1, 50)                                            // hasPending=true, seq=1
	b.HandleDebounce(VolumeDebounceTickMsg{TargetVol: 51, Seq: 1}) // must NOT clear pending
	b.SetConfirmed(30)                                             // must still be a no-op
	matched, vol, _ := b.HandleDebounce(VolumeDebounceTickMsg{TargetVol: 51, Seq: 1})
	assert.False(t, matched, "seq already consumed — second call is stale")
	_ = vol
	// Verify SetConfirmed was ignored: currentVol should still be 51.
	out := b.Render()
	assert.Contains(t, out, "51%", "currentVol must remain 51 after SetConfirmed(30) while pending")
	// After pending is still true, a fresh HandleKey uses the pending value as base.
	// HandleDebounce incremented seq to 2 on match. HandleKey increments to 3.
	b.HandleKey(+1, 30)
	matched2, vol2, _ := b.HandleDebounce(VolumeDebounceTickMsg{TargetVol: 52, Seq: 3})
	assert.True(t, matched2)
	assert.Equal(t, 52, vol2)
}

func TestVolumeBar_SetConfirmed_NoOpWhenPending(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	b.SetConfirmed(50)
	b.HandleKey(+1, 50) // hasPending=true, currentVol=51
	b.SetConfirmed(30)  // must be ignored
	// currentVol still 51; verify via HandleDebounce
	matched, vol, _ := b.HandleDebounce(VolumeDebounceTickMsg{TargetVol: 51, Seq: 1})
	assert.True(t, matched)
	assert.Equal(t, 51, vol)
}

func TestVolumeBar_SetConfirmed_UpdatesWhenNoPending(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	b.SetConfirmed(50)
	b.SetConfirmed(75) // no pending — should update
	b.HandleKey(+1, 75)
	matched, vol, _ := b.HandleDebounce(VolumeDebounceTickMsg{TargetVol: 76, Seq: 1})
	assert.True(t, matched)
	assert.Equal(t, 76, vol)
}

// --------------------------------------------------------------------------
// ConfirmFromAPI / CancelPending
// --------------------------------------------------------------------------

func TestVolumeBar_ConfirmFromAPI_ConfirmsOnSeqMatch(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	b.SetConfirmed(50)
	b.HandleKey(+1, 50) // seq=1, hasPending=true, currentVol=51

	matched, _, intentSeq := b.HandleDebounce(VolumeDebounceTickMsg{TargetVol: 51, Seq: 1})
	require.True(t, matched)
	// After HandleDebounce, b.seq == 2 (intentSeq+1).
	b.ConfirmFromAPI(intentSeq, 55)

	// currentVol should be 55, but hasPending stays true until a matching poll arrives.
	out := b.Render()
	assert.Contains(t, out, "55%")
	// Stale poll with old volume should be blocked while hasPending is true.
	b.SetConfirmed(50)
	out = b.Render()
	assert.Contains(t, out, "55%", "stale poll must not snap bar back")
	// Matching poll clears hasPending.
	b.SetConfirmed(55)
	out = b.Render()
	assert.Contains(t, out, "55%")
	// Now SetConfirmed should be accepted since hasPending=false.
	b.SetConfirmed(30)
	out = b.Render()
	assert.Contains(t, out, "30%")
}

func TestVolumeBar_ConfirmFromAPI_NoOpOnSeqMismatch(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	b.SetConfirmed(50)
	b.HandleKey(+1, 50) // seq=1

	matched, _, intentSeq := b.HandleDebounce(VolumeDebounceTickMsg{TargetVol: 51, Seq: 1})
	require.True(t, matched)
	// b.seq is now 2. Start a new burst (seq=3).
	b.HandleKey(+1, 51) // seq=3, currentVol=52

	// ConfirmFromAPI with the old intentSeq should be a no-op.
	b.ConfirmFromAPI(intentSeq, 55)
	// hasPending should still be true, currentVol should remain 52.
	out := b.Render()
	assert.Contains(t, out, "52%")
	// SetConfirmed should still be ignored.
	b.SetConfirmed(0)
	out = b.Render()
	assert.Contains(t, out, "52%")
}

func TestVolumeBar_CancelPending_ClearsOnSeqMatch(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	b.SetConfirmed(50)
	b.HandleKey(+1, 50) // seq=1, hasPending=true, currentVol=51

	matched, _, intentSeq := b.HandleDebounce(VolumeDebounceTickMsg{TargetVol: 51, Seq: 1})
	require.True(t, matched)
	// After HandleDebounce, b.seq == 2 (intentSeq+1).
	b.CancelPending(intentSeq, 50) // revert to last confirmed store value

	// hasPending should be cleared, currentVol should revert to confirmed store value.
	out := b.Render()
	assert.Contains(t, out, "50%", "error must revert bar to confirmed store value")
	// SetConfirmed should now be accepted.
	b.SetConfirmed(30)
	out = b.Render()
	assert.Contains(t, out, "30%")
}

func TestVolumeBar_CancelPending_NoOpOnSeqMismatch(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	b.SetConfirmed(50)
	b.HandleKey(+1, 50) // seq=1

	matched, _, intentSeq := b.HandleDebounce(VolumeDebounceTickMsg{TargetVol: 51, Seq: 1})
	require.True(t, matched)
	// b.seq is now 2. Start a new burst (seq=3).
	b.HandleKey(+1, 51) // seq=3, currentVol=52

	// CancelPending with the old intentSeq should be a no-op.
	b.CancelPending(intentSeq, 50)
	// hasPending should still be true, currentVol should remain 52.
	out := b.Render()
	assert.Contains(t, out, "52%")
	// SetConfirmed should still be ignored.
	b.SetConfirmed(0)
	out = b.Render()
	assert.Contains(t, out, "52%")
}

// TestVolumeBar_SetConfirmed_BlocksStalePoll verifies that after ConfirmFromAPI,
// a stale poll with the old volume is blocked, and only a matching poll clears
// hasPending. This prevents the bar from flickering back and forth.
func TestVolumeBar_SetConfirmed_BlocksStalePoll(t *testing.T) {
	b := newTestGradientVolumeBar(40)
	b.SetConfirmed(50)
	b.HandleKey(+1, 50) // seq=1, hasPending=true, currentVol=51

	matched, _, intentSeq := b.HandleDebounce(VolumeDebounceTickMsg{TargetVol: 51, Seq: 1})
	require.True(t, matched)
	b.ConfirmFromAPI(intentSeq, 55) // currentVol=55, hasPending stays true

	// Stale poll with old volume must be blocked.
	b.SetConfirmed(50)
	out := b.Render()
	assert.Contains(t, out, "55%", "stale poll must not snap bar back")

	// Matching poll clears hasPending.
	b.SetConfirmed(55)
	out = b.Render()
	assert.Contains(t, out, "55%")

	// Now SetConfirmed updates freely.
	b.SetConfirmed(30)
	out = b.Render()
	assert.Contains(t, out, "30%")
}

// TestGradientVolumeBar_AsciiMode verifies that in ASCII mode the music-note
// glyph rendered by GradientVolumeBar is the ASCII fallback ("*") and NOT the
// unicode "♪" (GlyphMusicNote).
func TestGradientVolumeBar_AsciiMode(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	b := newTestGradientVolumeBar(40)
	b.SetConfirmed(50)
	out := b.Render()

	// Unicode music note must not appear in ASCII mode.
	assert.NotContains(t, out, "♪", "unicode music note ♪ must not appear in ASCII mode")

	// ASCII fallback for GlyphMusicNote is "*" per the glyph catalogue.
	assert.Contains(t, out, "*", "ASCII replacement '*' for ♪ (GlyphMusicNote) must appear in ASCII mode")
}

// --------------------------------------------------------------------------
// GradientSeekBar — interactive seek state
// --------------------------------------------------------------------------

func TestSeekBar_HandleKey_UpdatesImmediately(t *testing.T) {
	b := newTestSeekBar(50)
	cmd := b.HandleKey(5000, 30000, 180000) // +5s from 30s position, 3min track
	assert.NotNil(t, cmd, "HandleKey must return a non-nil debounce cmd")
	assert.Equal(t, 35000, b.Current(), "current should be 30000+5000")
	assert.True(t, b.HasPending(), "should have pending after HandleKey")
}

func TestSeekBar_HandleKey_SeeksBackward(t *testing.T) {
	b := newTestSeekBar(50)
	b.HandleKey(-5000, 30000, 180000) // -5s from 30s
	assert.Equal(t, 25000, b.Current(), "current should be 30000-5000")
}

func TestSeekBar_HandleKey_ClampsAtZero(t *testing.T) {
	b := newTestSeekBar(50)
	b.HandleKey(-5000, 2000, 180000) // -5s from 2s → clamp to 0
	assert.Equal(t, 0, b.Current(), "should clamp at 0")
}

func TestSeekBar_HandleKey_ClampsAtDuration(t *testing.T) {
	b := newTestSeekBar(50)
	b.HandleKey(5000, 178000, 180000) // +5s from 178s → clamp to 180s
	assert.Equal(t, 180000, b.Current(), "should clamp at duration")
}

func TestSeekBar_HandleKey_NoOpOnZeroDuration(t *testing.T) {
	b := newTestSeekBar(50)
	cmd := b.HandleKey(5000, 30000, 0) // no track loaded
	assert.Nil(t, cmd, "HandleKey should return nil when duration is 0")
	assert.Equal(t, 0, b.Current(), "current should not change")
	assert.False(t, b.HasPending(), "should not set pending")
}

func TestSeekBar_HandleKey_AccumulatesFromPending(t *testing.T) {
	b := newTestSeekBar(50)
	b.HandleKey(5000, 30000, 180000) // +5s from 30s → 35s, seq=1
	b.HandleKey(5000, 30000, 180000) // +5s from 35s (pending) → 40s, seq=2
	assert.Equal(t, 40000, b.Current(), "should accumulate from pending value")
}

func TestSeekBar_HandleDebounce_StaleRejected(t *testing.T) {
	b := newTestSeekBar(50)
	b.HandleKey(5000, 30000, 180000) // seq=1
	b.HandleKey(5000, 30000, 180000) // seq=2 — supersedes seq=1
	matched, _, _ := b.HandleDebounce(SeekDebounceTickMsg{TargetMs: 35000, Seq: 1})
	assert.False(t, matched, "stale seq must be discarded")
}

func TestSeekBar_HandleDebounce_CurrentAccepted(t *testing.T) {
	b := newTestSeekBar(50)
	b.HandleKey(5000, 30000, 180000) // seq=1
	matched, target, _ := b.HandleDebounce(SeekDebounceTickMsg{TargetMs: 35000, Seq: 1})
	assert.True(t, matched, "current seq should match")
	assert.Equal(t, 35000, target)
}

func TestSeekBar_ConfirmFromAPI_SeqMatch(t *testing.T) {
	b := newTestSeekBar(50)
	b.HandleKey(5000, 30000, 180000) // seq=1
	matched, _, tickSeq := b.HandleDebounce(SeekDebounceTickMsg{TargetMs: 35000, Seq: 1})
	requireTrue(t, matched)
	b.ConfirmFromAPI(tickSeq, 35000)
	assert.Equal(t, 35000, b.Current(), "should update to API-confirmed position")
	assert.True(t, b.HasPending(), "hasPending stays true until SetConfirmed matches")
}

func TestSeekBar_CancelPending_SeqMatch(t *testing.T) {
	b := newTestSeekBar(50)
	b.HandleKey(5000, 30000, 180000) // seq=1
	matched, _, tickSeq := b.HandleDebounce(SeekDebounceTickMsg{TargetMs: 35000, Seq: 1})
	requireTrue(t, matched)
	b.CancelPending(tickSeq, 30000) // revert to confirmed store value
	assert.Equal(t, 30000, b.Current(), "should revert to confirmed value")
	assert.False(t, b.HasPending(), "should clear pending")
}

func TestSeekBar_SetConfirmed_ClearsPendingOnMatch(t *testing.T) {
	b := newTestSeekBar(50)
	b.HandleKey(5000, 30000, 180000) // current=35000, pending
	b.SetPositionConfirmed(35000)    // value matches current → clear pending
	assert.Equal(t, 35000, b.Current())
	assert.False(t, b.HasPending(), "should clear pending when value matches current")
}

func TestSeekBar_SetConfirmed_NoOpWhenPendingMismatch(t *testing.T) {
	b := newTestSeekBar(50)
	b.HandleKey(5000, 30000, 180000) // current=35000, pending
	b.SetPositionConfirmed(30000)    // value != current → ignored
	assert.Equal(t, 35000, b.Current(), "should keep pending value")
	assert.True(t, b.HasPending(), "should keep pending flag")
}

func TestSeekBar_SetConfirmed_UpdatesWhenNoPending(t *testing.T) {
	b := newTestSeekBar(50)
	b.SetPositionConfirmed(30000) // no pending → updates directly
	assert.Equal(t, 30000, b.Current())
	assert.False(t, b.HasPending())
}

func TestSeekBar_Render_UsesPendingWhenActive(t *testing.T) {
	b := newTestSeekBar(50)
	b.SetTrackDuration(180000)
	b.SetPositionConfirmed(30000)
	b.HandleKey(5000, 30000, 180000) // pending: current=35000
	out := b.Render(30000, 180000)   // progress param is overridden by pending
	assert.Contains(t, out, "0:35", "should show pending position (35000ms)")
}

func TestSeekBar_Render_UsesParameterWhenNotPending(t *testing.T) {
	b := newTestSeekBar(50)
	b.SetTrackDuration(180000)
	b.SetPositionConfirmed(30000)
	out := b.Render(30000, 180000) // no pending, uses progress param
	assert.Contains(t, out, "0:30", "should show confirmed position (30000ms)")
}
