package uikit_test

import (
	"testing"

	"github.com/initgrep-apps/spotnik/internal/ui/theme"
	"github.com/initgrep-apps/spotnik/internal/uikit"
	"github.com/stretchr/testify/assert"
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
