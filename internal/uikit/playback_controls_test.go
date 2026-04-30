package uikit_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestPlaybackControls(playing, shuffle bool, repeat uikit.RepeatMode) uikit.PlaybackControls {
	return uikit.PlaybackControls{
		Playing:    playing,
		Shuffle:    shuffle,
		RepeatMode: repeat,
		Theme:      theme.Load("black"),
	}
}

// TestPlaybackControls_RenderUnicode_Playing verifies unicode render when playing
// with shuffle off and repeat off (defaults).
func TestPlaybackControls_RenderUnicode_Playing(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	c := newTestPlaybackControls(true, false, uikit.RepeatOff)
	out := c.Render()

	assert.Contains(t, out, "⏸", "playing state shows pause icon")
	assert.Contains(t, out, "≡", "queue icon always present")
	assert.Contains(t, out, "⇄", "shuffle icon always present")
	assert.Contains(t, out, "⟳", "repeat-off shows ⟳")
}

// TestPlaybackControls_RenderASCII_Playing verifies ASCII render when playing.
func TestPlaybackControls_RenderASCII_Playing(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	c := newTestPlaybackControls(true, false, uikit.RepeatOff)
	out := c.Render()

	assert.Contains(t, out, "||", "ASCII play-pause: ||")
	assert.Contains(t, out, "Q", "ASCII queue: Q")
	assert.Contains(t, out, "sh", "ASCII shuffle: sh")
	assert.Contains(t, out, "ro", "ASCII repeat-off: ro")

	// No unicode glyphs in ASCII mode
	assert.NotContains(t, out, "⏸")
	assert.NotContains(t, out, "≡")
	assert.NotContains(t, out, "⇄")
	assert.NotContains(t, out, "↻")
	assert.NotContains(t, out, "⏷")
}

// TestPlaybackControls_RoleTokens verifies that active positions use Theme.PlayingIndicator()
// and inactive positions use Theme.TextSecondary(). A regression swapping active/inactive
// style would not be caught by glyph-only tests.
//
// Black theme: PlayingIndicator=#00ff88 (0,255,136)   → "38;2;0;255;136"
//
//	TextSecondary=#888888   (136,136,136) → "38;2;136;136;136"
func TestPlaybackControls_RoleTokens(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	th := theme.Load("black")
	activeANSI := "38;2;0;255;136"     // PlayingIndicator #00ff88
	inactiveANSI := "38;2;136;136;136" // TextSecondary    #888888

	// Sanity: active and inactive colors must differ so the test is meaningful.
	require.NotEqual(t, th.PlayingIndicator(), th.TextSecondary(),
		"test precondition: PlayingIndicator and TextSecondary must differ for black theme")

	t.Run("all active except queue", func(t *testing.T) {
		// Playing=true, Shuffle=true, Repeat=RepeatAll →
		//   shuffle, play/pause, repeat: active color
		//   queue: always inactive
		c := uikit.PlaybackControls{
			Playing:    true,
			Shuffle:    true,
			RepeatMode: uikit.RepeatAll,
			Theme:      th,
		}
		out := c.Render()

		assert.Contains(t, out, activeANSI,
			"active positions (shuffle/play/repeat) must use PlayingIndicator color")
		assert.Contains(t, out, inactiveANSI,
			"queue position must always use TextSecondary (inactive) color")
	})

	t.Run("all inactive", func(t *testing.T) {
		// Playing=false, Shuffle=false, Repeat=RepeatOff →
		//   all four positions: inactive color only
		c := uikit.PlaybackControls{
			Playing:    false,
			Shuffle:    false,
			RepeatMode: uikit.RepeatOff,
			Theme:      th,
		}
		out := c.Render()

		assert.NotContains(t, out, activeANSI,
			"no active positions — PlayingIndicator color must NOT appear")
		assert.Contains(t, out, inactiveANSI,
			"all positions inactive — TextSecondary color must appear")
	})
}

// TestPlaybackControls_Paused_Unicode verifies that Playing=false renders the
// GlyphPausedPB glyph (▷) in unicode mode. GlyphPaused (⏸) is the "playing,
// press to pause" icon; GlyphPausedPB (▷) is the "paused, press to play" icon.
// A regression that swapped these two roles would not be caught by the
// Playing=true tests above.
func TestPlaybackControls_Paused_Unicode(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphUnicode)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	c := newTestPlaybackControls(false, false, uikit.RepeatOff)
	out := c.Render()

	assert.Contains(t, out, "▷", "paused state (Playing=false) must show ▷ (GlyphPausedPB) in unicode mode")
	assert.NotContains(t, out, "⏸", "paused state must NOT show ⏸ (GlyphPaused); that is the playing-state icon")
}

// TestPlaybackControls_Paused_ASCII verifies that Playing=false renders "|>"
// in ASCII mode. Catches a swap of GlyphPaused/GlyphPausedPB at the primitive level.
func TestPlaybackControls_Paused_ASCII(t *testing.T) {
	uikit.SetModeForTest(uikit.GlyphASCII)
	defer uikit.SetModeForTest(uikit.GlyphUnicode)

	c := newTestPlaybackControls(false, false, uikit.RepeatOff)
	out := c.Render()

	assert.Contains(t, out, "|>", "paused state (Playing=false) must show |> (GlyphPausedPB) in ASCII mode")
	assert.NotContains(t, out, "||", "paused state must NOT show || (GlyphPaused); that is the playing-state icon")
}

// TestPlaybackControls_RepeatModes verifies each repeat mode renders the correct glyph.
func TestPlaybackControls_RepeatModes(t *testing.T) {
	tests := []struct {
		name        string
		mode        uikit.RepeatMode
		glyphMode   uikit.GlyphMode
		wantContain string
		wantAbsent  string
	}{
		{
			name:        "RepeatOff unicode",
			mode:        uikit.RepeatOff,
			glyphMode:   uikit.GlyphUnicode,
			wantContain: "⟳",
			wantAbsent:  "↻",
		},
		{
			name:        "RepeatOff ascii",
			mode:        uikit.RepeatOff,
			glyphMode:   uikit.GlyphASCII,
			wantContain: "ro",
		},
		{
			name:        "RepeatAll unicode",
			mode:        uikit.RepeatAll,
			glyphMode:   uikit.GlyphUnicode,
			wantContain: "↻",
		},
		{
			name:        "RepeatAll ascii",
			mode:        uikit.RepeatAll,
			glyphMode:   uikit.GlyphASCII,
			wantContain: "rp",
		},
		{
			name:        "RepeatOne unicode",
			mode:        uikit.RepeatOne,
			glyphMode:   uikit.GlyphUnicode,
			wantContain: "↻¹",
		},
		{
			name:        "RepeatOne ascii",
			mode:        uikit.RepeatOne,
			glyphMode:   uikit.GlyphASCII,
			wantContain: "rp1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uikit.SetModeForTest(tt.glyphMode)
			defer uikit.SetModeForTest(uikit.GlyphUnicode)

			c := newTestPlaybackControls(false, false, tt.mode)
			out := c.Render()

			assert.Contains(t, out, tt.wantContain)
			if tt.wantAbsent != "" {
				assert.NotContains(t, out, tt.wantAbsent)
			}
		})
	}
}
